package repository

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"strings"

	"github.com/tijanadmi/ugo_evid/models"
)

var (
	ErrContactLocked  = errors.New("Lice trenutno uređuje drugi korisnik ili drugi tab. Pokušajte kasnije.")
	ErrContactLease   = errors.New("Pravo izmene je isteklo. Ponovo preuzmite lice i proverite aktuelne podatke.")
	ErrContactVersion = errors.New("Podaci su promenjeni. Učitajte aktuelne podatke pre ponovnog čuvanja.")
	ErrContactLinked  = errors.New("Lice je povezano sa evidencijom ugovora. Umesto brisanja izaberite status Neaktivan.")
)

// Every lease operation and mutation first locks the same business row. This
// serializes acquisition, expiry takeover, renewal and writes across instances.
// Transactions last only for the request, never for the lifetime of the form.
func contactTransaction(ctx context.Context, db *sql.DB, id int) (*sql.Tx, int64, int, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, 0, 0, err
	}
	var version int64
	var supplier int
	err = tx.QueryRowContext(ctx, `SELECT version, id_sap_dobavljac FROM TED.UGO_DOB_LICA WHERE id = :id FOR UPDATE NOWAIT`, sql.Named("id", id)).Scan(&version, &supplier)
	if err != nil {
		tx.Rollback()
		if strings.Contains(err.Error(), "ORA-00054") {
			err = ErrContactLocked
		}
		return nil, 0, 0, err
	}
	return tx, version, supplier, nil
}

func leaseArgs(id, userID int, token string) []any {
	return []any{sql.Named("id", id), sql.Named("user_id", userID), sql.Named("token", token)}
}

const ownedLease = `entity_type = 'UGO_DOB_LICA' AND entity_id = :id AND user_id = :user_id AND lock_token = :token`

func requireLease(ctx context.Context, tx *sql.Tx, id, userID int, token string) error {
	var count int
	err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM TED.UGO_EDIT_LOCK WHERE `+ownedLease+` AND expires_at > SYSTIMESTAMP`, leaseArgs(id, userID, token)...).Scan(&count)
	if err != nil {
		return err
	}
	if count != 1 {
		return ErrContactLease
	}
	return nil
}

func releaseLease(ctx context.Context, tx *sql.Tx, id, userID int, token string) error {
	_, err := tx.ExecContext(ctx, `DELETE FROM TED.UGO_EDIT_LOCK WHERE `+ownedLease, leaseArgs(id, userID, token)...)
	return err
}

func (r *OracleStore) ContactLock(ctx context.Context, id, userID int, action, token string) (*models.EditLock, error) {
	if id < 1 || userID < 1 {
		return nil, ErrContactLease
	}
	tx, _, _, err := contactTransaction(ctx, r.DB, id)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	result := &models.EditLock{}
	switch action {
	case "acquire":
		// Remove only an expired lease, under the business-row lock.
		_, err = tx.ExecContext(ctx, `DELETE FROM TED.UGO_EDIT_LOCK WHERE entity_type = 'UGO_DOB_LICA' AND entity_id = :id AND expires_at <= SYSTIMESTAMP`, sql.Named("id", id))
		if err != nil {
			return nil, err
		}
		var count int
		err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM TED.UGO_EDIT_LOCK WHERE entity_type = 'UGO_DOB_LICA' AND entity_id = :id`, sql.Named("id", id)).Scan(&count)
		if err != nil {
			return nil, err
		}
		if count != 0 {
			return nil, ErrContactLocked
		}
		bytes := make([]byte, 32)
		if _, err = rand.Read(bytes); err != nil {
			return nil, err
		}
		token = hex.EncodeToString(bytes)
		_, err = tx.ExecContext(ctx, `INSERT INTO TED.UGO_EDIT_LOCK (entity_type, entity_id, user_id, lock_token, acquired_at, renewed_at, expires_at)
 VALUES ('UGO_DOB_LICA', :id, :user_id, :token, SYSTIMESTAMP, SYSTIMESTAMP, SYSTIMESTAMP + INTERVAL '2' MINUTE)`, leaseArgs(id, userID, token)...)
	case "renew":
		if err = requireLease(ctx, tx, id, userID, token); err != nil {
			return nil, err
		}
		var updated sql.Result
		updated, err = tx.ExecContext(ctx, `UPDATE TED.UGO_EDIT_LOCK SET renewed_at = SYSTIMESTAMP, expires_at = SYSTIMESTAMP + INTERVAL '2' MINUTE WHERE `+ownedLease+` AND expires_at > SYSTIMESTAMP`, leaseArgs(id, userID, token)...)
		if err == nil {
			var n int64
			n, err = updated.RowsAffected()
			if err == nil && n != 1 {
				err = ErrContactLease
			}
		}
	case "release":
		err = releaseLease(ctx, tx, id, userID, token)
	default:
		return nil, ErrContactLease
	}
	if err != nil {
		return nil, err
	}
	if action != "release" {
		result.Token = token
		err = tx.QueryRowContext(ctx, `SELECT expires_at FROM TED.UGO_EDIT_LOCK WHERE `+ownedLease, leaseArgs(id, userID, token)...).Scan(&result.ExpiresAt)
		if err != nil {
			return nil, err
		}
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *OracleStore) UpdateUgoDobLice(ctx context.Context, m *models.UgoDobLice, userID int, token string) (*models.UgoDobLice, error) {
	tx, version, supplier, err := contactTransaction(ctx, r.DB, m.ID)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if err = requireLease(ctx, tx, m.ID, userID, token); err != nil {
		return nil, err
	}
	if m.Version != version || m.SapDobavljac.ID != supplier {
		return nil, ErrContactVersion
	}
	result, err := tx.ExecContext(ctx, `UPDATE TED.UGO_DOB_LICA SET ime = :ime, radno_mesto = :radno_mesto,
 telefon = :telefon, email = :email, id_ugo_dob_lica_rola = :rola, status = :status, datzm = SYSDATE, version = version + 1
 WHERE id = :id AND version = :version`, sql.Named("ime", m.Ime), sql.Named("radno_mesto", m.RadnoMesto),
		sql.Named("telefon", m.Telefon), sql.Named("email", m.Email), sql.Named("rola", optionalID(m.UgoDobLicaRola.ID)),
		sql.Named("status", m.Status), sql.Named("id", m.ID), sql.Named("version", m.Version))
	if err != nil {
		return nil, err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if n != 1 {
		return nil, ErrContactVersion
	}
	if err = releaseLease(ctx, tx, m.ID, userID, token); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	m.Version++
	return m, nil
}

func (r *OracleStore) DeleteUgoDobLiceById(ctx context.Context, id int, version int64, userID int, token string) error {
	tx, current, _, err := contactTransaction(ctx, r.DB, id)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = requireLease(ctx, tx, id, userID, token); err != nil {
		return err
	}
	if version != current {
		return ErrContactVersion
	}
	var linked int
	if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM TED.UGO_EVID WHERE id_ugo_dob_lica = :id`, sql.Named("id", id)).Scan(&linked); err != nil {
		return err
	}
	if linked > 0 {
		return ErrContactLinked
	}
	result, err := tx.ExecContext(ctx, `DELETE FROM TED.UGO_DOB_LICA WHERE id = :id AND version = :version`, sql.Named("id", id), sql.Named("version", version))
	if err != nil {
		if strings.Contains(err.Error(), "ORA-02292") {
			return ErrContactLinked
		}
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return ErrContactVersion
	}
	if err = releaseLease(ctx, tx, id, userID, token); err != nil {
		return err
	}
	return tx.Commit()
}
