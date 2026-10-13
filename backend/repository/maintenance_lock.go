package repository

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"github.com/tijanadmi/ugo_evid/models"
)

var (
	ErrMaintenanceLocked  = errors.New("Odrzavanje trenutno menja drugi korisnik ili drugi tab.")
	ErrMaintenanceLease   = errors.New("Pravo izmene odrzavanja je isteklo. Ponovo otvorite odrzavanje.")
	ErrMaintenanceVersion = errors.New("Odrzavanje je promenjeno. Ponovo otvorite aktuelne podatke.")
)

const ownedMaintenanceLease = `entity_type = 'UGO_ODRZAVANJE' AND entity_id = :id AND user_id = :user_id AND lock_token = :token`

func requireMaintenanceLease(ctx context.Context, tx *sql.Tx, id, userID int, token string) error {
	var count int
	err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM TED.UGO_EDIT_LOCK WHERE `+ownedMaintenanceLease+` AND expires_at > SYSTIMESTAMP`, leaseArgs(id, userID, token)...).Scan(&count)
	if err != nil {
		return err
	}
	if count != 1 {
		return ErrMaintenanceLease
	}
	return nil
}

func releaseMaintenanceLease(ctx context.Context, tx *sql.Tx, id, userID int, token string) error {
	_, err := tx.ExecContext(ctx, `DELETE FROM TED.UGO_EDIT_LOCK WHERE `+ownedMaintenanceLease, leaseArgs(id, userID, token)...)
	return err
}

func (r *OracleStore) MaintenanceLock(ctx context.Context, id, orgID, userID int, action, token string) (*models.EditLock, error) {
	if id < 1 || userID < 1 {
		return nil, ErrMaintenanceLease
	}
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if action != "release" {
		if _, _, err = lockMaintenanceParent(ctx, tx, id, orgID); err != nil {
			return nil, err
		}
		var maintenanceID int
		if err = tx.QueryRowContext(ctx, `SELECT id FROM TED.UGO_ODRZAVANJE WHERE id_ugo_evid=:id`, sql.Named("id", id)).Scan(&maintenanceID); err != nil {
			return nil, err
		}
	} else {
		var owner int
		if err = tx.QueryRowContext(ctx, `SELECT id_ugo_org FROM TED.UGO_EVID WHERE id=:id FOR UPDATE NOWAIT`, sql.Named("id", id)).Scan(&owner); err != nil {
			return nil, err
		}
		if owner != orgID {
			return nil, ErrUserOrganization
		}
	}
	result := &models.EditLock{}
	switch action {
	case "acquire":
		// Remove only an expired lease, under the business-row lock.
		_, err = tx.ExecContext(ctx, `DELETE FROM TED.UGO_EDIT_LOCK WHERE entity_type = 'UGO_ODRZAVANJE' AND entity_id = :id AND expires_at <= SYSTIMESTAMP`, sql.Named("id", id))
		if err != nil {
			return nil, err
		}
		var count int
		err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM TED.UGO_EDIT_LOCK WHERE entity_type = 'UGO_ODRZAVANJE' AND entity_id = :id`, sql.Named("id", id)).Scan(&count)
		if err != nil {
			return nil, err
		}
		if count != 0 {
			return nil, ErrMaintenanceLocked
		}
		bytes := make([]byte, 32)
		if _, err = rand.Read(bytes); err != nil {
			return nil, err
		}
		token = hex.EncodeToString(bytes)
		_, err = tx.ExecContext(ctx, `INSERT INTO TED.UGO_EDIT_LOCK (entity_type, entity_id, user_id, lock_token, acquired_at, renewed_at, expires_at)
 VALUES ('UGO_ODRZAVANJE', :id, :user_id, :token, SYSTIMESTAMP, SYSTIMESTAMP, SYSTIMESTAMP + INTERVAL '2' MINUTE)`, leaseArgs(id, userID, token)...)
	case "renew":
		if err = requireMaintenanceLease(ctx, tx, id, userID, token); err != nil {
			return nil, err
		}
		var updated sql.Result
		updated, err = tx.ExecContext(ctx, `UPDATE TED.UGO_EDIT_LOCK SET renewed_at = SYSTIMESTAMP, expires_at = SYSTIMESTAMP + INTERVAL '2' MINUTE WHERE `+ownedMaintenanceLease+` AND expires_at > SYSTIMESTAMP`, leaseArgs(id, userID, token)...)
		if err == nil {
			var n int64
			n, err = updated.RowsAffected()
			if err == nil && n != 1 {
				err = ErrMaintenanceLease
			}
		}
	case "release":
		err = releaseMaintenanceLease(ctx, tx, id, userID, token)
	default:
		return nil, ErrMaintenanceLease
	}
	if err != nil {
		return nil, err
	}
	if action != "release" {
		result.Token = token
		err = tx.QueryRowContext(ctx, `SELECT expires_at FROM TED.UGO_EDIT_LOCK WHERE `+ownedMaintenanceLease, leaseArgs(id, userID, token)...).Scan(&result.ExpiresAt)
		if err != nil {
			return nil, err
		}
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return result, nil
}
