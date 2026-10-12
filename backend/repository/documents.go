package repository

import (
	"context"
	"database/sql"
	"errors"
	"github.com/tijanadmi/ugo_evid/models"
	"strings"
)

var ErrDocumentType = errors.New("Tip dokumenta nije aktivan.")
var ErrDocumentClosed = errors.New("Dokumenta zatvorenog ugovora nije moguce menjati.")

const documentSelect = `SELECT d.id, d.id_ugo_evid, d.id_tip_doc, t.naziv, d.naziv,
 d.broj_dokumenta, TO_CHAR(d.datum_dokumenta,'YYYY-MM-DD'), d.napomena,
 d.originalni_naziv, d.putanja, d.mime_tip, d.velicina_bajt, d.sha256, d.status,
 d.id_korisnika_unosa, d.datpri, d.datizm, d.version
 FROM TED.UGO_DOC d JOIN TED.UGO_TIP_DOC t ON t.id = d.id_tip_doc`

func scanDocument(row interface{ Scan(...any) error }, d *models.Document) error {
	return scanNullable(row, &d.ID, &d.ContractID, &d.TypeID, &d.TypeName, &d.Name,
		&d.Number, &d.Date, &d.Note, &d.OriginalName, &d.Path, &d.MIME, &d.Size,
		&d.SHA256, &d.Status, &d.UserID, &d.Created, &d.Updated, &d.Version)
}
func (r *OracleStore) ListDocumentTypes(ctx context.Context) ([]models.DocumentType, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT id,sifra,naziv FROM TED.UGO_TIP_DOC WHERE status='A' ORDER BY redosled,id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.DocumentType{}
	for rows.Next() {
		var d models.DocumentType
		if err := rows.Scan(&d.ID, &d.Code, &d.Name); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}
func (r *OracleStore) ListDocuments(ctx context.Context, contractID int) ([]models.Document, error) {
	rows, err := r.DB.QueryContext(ctx, documentSelect+` WHERE d.id_ugo_evid=:id AND d.status='A' ORDER BY d.id DESC`, sql.Named("id", contractID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.Document{}
	for rows.Next() {
		var d models.Document
		if err := scanDocument(rows, &d); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}
func (r *OracleStore) GetDocument(ctx context.Context, id int) (*models.Document, error) {
	d := &models.Document{}
	err := scanDocument(r.DB.QueryRowContext(ctx, documentSelect+` WHERE d.id=:id AND d.status='A'`, sql.Named("id", id)), d)
	return d, err
}
func (r *OracleStore) DocumentFolder(ctx context.Context, contractID int, proposed string) (string, error) {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	var folder sql.NullString
	// Concurrent first uploads serialize here; the persisted name wins.
	err = tx.QueryRowContext(ctx, `SELECT doc_folder FROM TED.UGO_EVID WHERE id=:id FOR UPDATE WAIT 5`, sql.Named("id", contractID)).Scan(&folder)
	if err != nil {
		return "", err
	}
	if !folder.Valid {
		_, err = tx.ExecContext(ctx, `UPDATE TED.UGO_EVID SET doc_folder=:folder WHERE id=:id`, sql.Named("folder", proposed), sql.Named("id", contractID))
		if err != nil {
			return "", err
		}
		folder.String = proposed
	}
	return folder.String, tx.Commit()
}
func activeDocumentType(ctx context.Context, tx *sql.Tx, id int) error {
	var active int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM TED.UGO_TIP_DOC WHERE id=:id AND status='A'`, sql.Named("id", id)).Scan(&active); err != nil {
		return err
	}
	if active != 1 {
		return ErrDocumentType
	}
	return nil
}
func openDocumentContract(ctx context.Context, tx *sql.Tx, id int) error {
	var active int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM TED.UGO_EVID_PROSIRENI_V WHERE id_ugo_evid=:id AND otvoren_ug='X'`, sql.Named("id", id)).Scan(&active); err != nil {
		return err
	}
	if active != 1 {
		return ErrDocumentClosed
	}
	return nil
}
func (r *OracleStore) ReserveDocument(ctx context.Context, d *models.Document) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = activeDocumentType(ctx, tx, d.TypeID); err != nil {
		return err
	}
	if err = openDocumentContract(ctx, tx, d.ContractID); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO TED.UGO_DOC
 (id_ugo_evid,id_tip_doc,naziv,broj_dokumenta,datum_dokumenta,napomena,originalni_naziv,putanja,mime_tip,velicina_bajt,sha256,status,id_korisnika_unosa)
 VALUES (:contract_id,:doc_type,:doc_name,:doc_number,TO_DATE(:doc_date,'YYYY-MM-DD'),:doc_note,:original_name,:file_path,:file_mime,:file_size,:file_hash,'P',:user_id) RETURNING id INTO :out_id`,
		sql.Named("contract_id", d.ContractID), sql.Named("doc_type", d.TypeID), sql.Named("doc_name", d.Name), sql.Named("doc_number", d.Number), sql.Named("doc_date", d.Date), sql.Named("doc_note", d.Note),
		sql.Named("original_name", d.OriginalName), sql.Named("file_path", d.Path), sql.Named("file_mime", d.MIME), sql.Named("file_size", d.Size), sql.Named("file_hash", d.SHA256), sql.Named("user_id", d.UserID), sql.Named("out_id", sql.Out{Dest: &d.ID}))
	if err != nil {
		return err
	}
	return tx.Commit()
}
func (r *OracleStore) FinalizeDocument(ctx context.Context, id int) error {
	result, err := r.DB.ExecContext(ctx, `UPDATE TED.UGO_DOC SET status='A' WHERE id=:id AND status='P'`, sql.Named("id", id))
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return sql.ErrNoRows
	}
	return nil
}
func documentTransaction(ctx context.Context, db *sql.DB, id int) (*sql.Tx, int64, int, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, 0, 0, err
	}
	var version int64
	var contract int
	err = tx.QueryRowContext(ctx, `SELECT version,id_ugo_evid FROM TED.UGO_DOC WHERE id=:id AND status='A' FOR UPDATE NOWAIT`, sql.Named("id", id)).Scan(&version, &contract)
	if err != nil {
		tx.Rollback()
		if strings.Contains(err.Error(), "ORA-00054") {
			err = ErrDocumentLocked
		}
		return nil, 0, 0, err
	}
	return tx, version, contract, nil
}
func (r *OracleStore) ChangeDocument(ctx context.Context, d *models.Document, userID int, token string, remove bool) error {
	tx, version, contract, err := documentTransaction(ctx, r.DB, d.ID)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = requireDocumentLease(ctx, tx, d.ID, userID, token); err != nil {
		return err
	}
	if version != d.Version {
		return ErrDocumentVersion
	}
	if err = openDocumentContract(ctx, tx, contract); err != nil {
		return err
	}
	args := []any{sql.Named("id", d.ID), sql.Named("version", version), sql.Named("user_id", userID)}
	query := `UPDATE TED.UGO_DOC SET status='D',version=version+1,datizm=SYSTIMESTAMP,id_korisnika_izmene=:user_id WHERE id=:id AND version=:version AND status='A'`
	if !remove {
		if err = activeDocumentType(ctx, tx, d.TypeID); err != nil {
			return err
		}
		query = `UPDATE TED.UGO_DOC SET id_tip_doc=:doc_type,naziv=:doc_name,broj_dokumenta=:doc_number,datum_dokumenta=TO_DATE(:doc_date,'YYYY-MM-DD'),napomena=:doc_note,version=version+1,datizm=SYSTIMESTAMP,id_korisnika_izmene=:user_id WHERE id=:id AND version=:version AND status='A'`
		args = append(args, sql.Named("doc_type", d.TypeID), sql.Named("doc_name", d.Name), sql.Named("doc_number", d.Number), sql.Named("doc_date", d.Date), sql.Named("doc_note", d.Note))
	}
	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return ErrDocumentVersion
	}
	if err = releaseDocumentLease(ctx, tx, d.ID, userID, token); err != nil {
		return err
	}
	return tx.Commit()
}

// Operator recovery selects pending rows older than an hour. Stop uploads on
// all instances before using the reconciliation tool.
func (r *OracleStore) PendingDocuments(ctx context.Context) ([]models.Document, error) {
	rows, err := r.DB.QueryContext(ctx, documentSelect+` WHERE d.status='P' AND d.datpri < SYSTIMESTAMP - INTERVAL '1' HOUR ORDER BY d.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.Document{}
	for rows.Next() {
		var d models.Document
		if err := scanDocument(rows, &d); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}
