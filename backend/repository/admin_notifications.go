package repository

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/tijanadmi/ugo_evid/models"
)

var ErrNotificationPending = errors.New("Brisanje još nije završeno; obaveštenje će biti rešeno automatski nakon uspešnog brisanja.")

const notificationSelect = `SELECT n.id,n.vrsta,n.id_sloga,n.naslov,n.poruka,n.status,NVL(n.id_preuzeo,0),k.ad_sifra,
 TO_CHAR(n.prvi_put,'YYYY-MM-DD"T"HH24:MI:SS'),TO_CHAR(n.poslednji_put,'YYYY-MM-DD"T"HH24:MI:SS'),n.broj_ponavljanja,
 TO_CHAR(n.reseno,'YYYY-MM-DD"T"HH24:MI:SS'),n.version,b.originalni_naziv,NVL(b.id_ugo_evid,0),NVL(b.broj_pokusaja,0)
 FROM TED.PORTAL_OBAVESTENJE n LEFT JOIN TED.UGO_KOR k ON k.id=n.id_preuzeo
 LEFT JOIN TED.UGO_DOC_BRISANJE b ON n.vrsta='DOC_DELETE' AND b.id_doc=n.id_sloga`

func scanNotification(row interface{ Scan(...any) error }, n *models.AdminNotification) error {
	return scanNullable(row, &n.ID, &n.Kind, &n.EntityID, &n.Title, &n.Message, &n.Status, &n.AssignedID, &n.AssignedName, &n.First, &n.Last, &n.Count, &n.Resolved, &n.Version, &n.OriginalName, &n.ContractID, &n.Attempts)
}
func (r *OracleStore) ListAdminNotifications(ctx context.Context, status string, offset, limit int) ([]models.AdminNotification, int, error) {
	where := " WHERE n.status<>'RESENO'"
	switch status {
	case "":
	case "SVE":
		where = " WHERE 1=1"
	case "OTVORENO", "PREUZETO", "RESENO":
		where = " WHERE n.status='" + status + "'"
	default:
		return nil, 0, ErrAdminInput
	}
	var total int
	if err := r.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM TED.PORTAL_OBAVESTENJE n`+where).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.DB.QueryContext(ctx, notificationSelect+where+` ORDER BY n.id DESC OFFSET :off ROWS FETCH NEXT :lim ROWS ONLY`, sql.Named("off", offset), sql.Named("lim", limit))
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := []models.AdminNotification{}
	for rows.Next() {
		var n models.AdminNotification
		if err := scanNotification(rows, &n); err != nil {
			return nil, 0, err
		}
		items = append(items, n)
	}
	return items, total, rows.Err()
}
func (r *OracleStore) ChangeAdminNotification(ctx context.Context, actor int, active string, id int, version int64, action string) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := lockAdmin(ctx, tx, actor, active); err != nil {
		return err
	}
	var locked int
	if err := tx.QueryRowContext(ctx, `SELECT id FROM TED.PORTAL_OBAVESTENJE WHERE id=:id FOR UPDATE WAIT 5`, sql.Named("id", id)).Scan(&locked); err != nil {
		return err
	}
	var before models.AdminNotification
	if err := scanNotification(tx.QueryRowContext(ctx, notificationSelect+` WHERE n.id=:id`, sql.Named("id", id)), &before); err != nil {
		return err
	}
	if before.Version != version {
		return ErrAdminConflict
	}
	after := before
	switch action {
	case "PREUZMI":
		if before.Status != "OTVORENO" {
			return ErrAdminConflict
		}
		after.Status = "PREUZETO"
		after.AssignedID = actor
	case "VRATI":
		if before.Status != "PREUZETO" || before.AssignedID != actor {
			return ErrAdminConflict
		}
		after.Status = "OTVORENO"
		after.AssignedID = 0
	case "RESI":
		if before.Status == "RESENO" {
			return ErrAdminConflict
		}
		if before.Kind == "DOC_DELETE" {
			var pending int
			if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM TED.UGO_DOC_BRISANJE WHERE id_doc=:id AND zavrseno IS NULL`, sql.Named("id", before.EntityID)).Scan(&pending); err != nil {
				return err
			}
			if pending > 0 {
				return ErrNotificationPending
			}
		}
		after.Status = "RESENO"
	default:
		return ErrAdminInput
	}
	after.Version++
	_, err = tx.ExecContext(ctx, `UPDATE TED.PORTAL_OBAVESTENJE SET status=:status,id_preuzeo=NULLIF(:assigned,0),version=version+1,reseno=CASE WHEN :resolved=1 THEN SYSTIMESTAMP ELSE NULL END WHERE id=:id`, sql.Named("status", after.Status), sql.Named("assigned", after.AssignedID), sql.Named("resolved", boolInt(after.Status == "RESENO")), sql.Named("id", id))
	if err != nil {
		return err
	}
	// Error details remain in the notification; audit the administrative state
	// change without duplicating long filesystem/database error messages.
	state := func(n models.AdminNotification) any {
		return struct {
			Status     string `json:"status"`
			AssignedID int    `json:"id_preuzeo"`
			Version    int64  `json:"version"`
		}{n.Status, n.AssignedID, n.Version}
	}
	if err := adminAudit(ctx, tx, actor, id, action, "OBAVESTENJE", state(before), state(after)); err != nil {
		return err
	}
	return tx.Commit()
}

type DeletionAttemptStore interface {
	RecordDocumentDeletionAttempt(context.Context, int, string) error
}

func limitedError(message string) string {
	message = strings.ReplaceAll(message, "\x00", "")
	if len(message) > 2000 {
		message = message[:2000]
		for !utf8.ValidString(message) {
			message = message[:len(message)-1]
		}
	}
	return message
}

// Audit-row serialization also prevents duplicate notifications when several
// instances report the same failed deletion. Do not reopen a resolved deletion.
func (r *OracleStore) RecordDocumentDeletionAttempt(ctx context.Context, id int, message string) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var done sql.NullTime
	var overdue int
	err = tx.QueryRowContext(ctx, `SELECT zavrseno,CASE WHEN trazeno<=SYSTIMESTAMP-INTERVAL '15' MINUTE THEN 1 ELSE 0 END FROM TED.UGO_DOC_BRISANJE WHERE id_doc=:id FOR UPDATE WAIT 5`, sql.Named("id", id)).Scan(&done, &overdue)
	if err != nil {
		return err
	}
	message = limitedError(message)
	if done.Valid {
		message = ""
	}
	_, err = tx.ExecContext(ctx, `UPDATE TED.UGO_DOC_BRISANJE SET broj_pokusaja=broj_pokusaja+1,poslednji_pokusaj=SYSTIMESTAMP,poslednja_greska=:message WHERE id_doc=:id`, sql.Named("message", message), sql.Named("id", id))
	if err != nil {
		return err
	}
	if !done.Valid && overdue == 1 && message != "" {
		_, err = tx.ExecContext(ctx, `MERGE INTO TED.PORTAL_OBAVESTENJE n USING (SELECT :source_key source_key FROM dual) s ON (n.izvor_kljuc=s.source_key)
 WHEN MATCHED THEN UPDATE SET n.poruka=:message_update,n.poslednji_put=SYSTIMESTAMP,n.broj_ponavljanja=n.broj_ponavljanja+1,n.version=n.version+1
 WHEN NOT MATCHED THEN INSERT (izvor_kljuc,vrsta,id_sloga,naslov,poruka) VALUES (s.source_key,'DOC_DELETE',:doc_id,'Brisanje dokumenta zahteva proveru',:message_insert)`, sql.Named("source_key", "DOC_DELETE:"+strconv.Itoa(id)), sql.Named("message_update", message), sql.Named("doc_id", id), sql.Named("message_insert", message))
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}
