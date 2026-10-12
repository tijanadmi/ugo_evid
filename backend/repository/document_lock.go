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
	ErrDocumentLocked  = errors.New("Dokument trenutno menja drugi korisnik ili drugi tab.")
	ErrDocumentLease   = errors.New("Pravo izmene dokumenta je isteklo. Ponovo otvorite dokument.")
	ErrDocumentVersion = errors.New("Dokument je promenjen. Ponovo otvorite aktuelne podatke.")
)

const ownedDocumentLease = `entity_type = 'UGO_DOC' AND entity_id = :id AND user_id = :user_id AND lock_token = :token`

func requireDocumentLease(ctx context.Context, tx *sql.Tx, id, userID int, token string) error {
	var count int
	err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM TED.UGO_EDIT_LOCK WHERE `+ownedDocumentLease+` AND expires_at > SYSTIMESTAMP`, leaseArgs(id, userID, token)...).Scan(&count)
	if err != nil {
		return err
	}
	if count != 1 {
		return ErrDocumentLease
	}
	return nil
}

func releaseDocumentLease(ctx context.Context, tx *sql.Tx, id, userID int, token string) error {
	_, err := tx.ExecContext(ctx, `DELETE FROM TED.UGO_EDIT_LOCK WHERE `+ownedDocumentLease, leaseArgs(id, userID, token)...)
	return err
}

func (r *OracleStore) DocumentLock(ctx context.Context, id, userID int, action, token string) (*models.EditLock, error) {
	if id < 1 || userID < 1 {
		return nil, ErrDocumentLease
	}
	tx, _, contractID, err := documentTransaction(ctx, r.DB, id)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if action != "release" {
		if err = openDocumentContract(ctx, tx, contractID); err != nil {
			return nil, err
		}
	}
	result := &models.EditLock{}
	switch action {
	case "acquire":
		// Remove only an expired lease, under the business-row lock.
		_, err = tx.ExecContext(ctx, `DELETE FROM TED.UGO_EDIT_LOCK WHERE entity_type = 'UGO_DOC' AND entity_id = :id AND expires_at <= SYSTIMESTAMP`, sql.Named("id", id))
		if err != nil {
			return nil, err
		}
		var count int
		err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM TED.UGO_EDIT_LOCK WHERE entity_type = 'UGO_DOC' AND entity_id = :id`, sql.Named("id", id)).Scan(&count)
		if err != nil {
			return nil, err
		}
		if count != 0 {
			return nil, ErrDocumentLocked
		}
		bytes := make([]byte, 32)
		if _, err = rand.Read(bytes); err != nil {
			return nil, err
		}
		token = hex.EncodeToString(bytes)
		_, err = tx.ExecContext(ctx, `INSERT INTO TED.UGO_EDIT_LOCK (entity_type, entity_id, user_id, lock_token, acquired_at, renewed_at, expires_at)
 VALUES ('UGO_DOC', :id, :user_id, :token, SYSTIMESTAMP, SYSTIMESTAMP, SYSTIMESTAMP + INTERVAL '2' MINUTE)`, leaseArgs(id, userID, token)...)
	case "renew":
		if err = requireDocumentLease(ctx, tx, id, userID, token); err != nil {
			return nil, err
		}
		var updated sql.Result
		updated, err = tx.ExecContext(ctx, `UPDATE TED.UGO_EDIT_LOCK SET renewed_at = SYSTIMESTAMP, expires_at = SYSTIMESTAMP + INTERVAL '2' MINUTE WHERE `+ownedDocumentLease+` AND expires_at > SYSTIMESTAMP`, leaseArgs(id, userID, token)...)
		if err == nil {
			var n int64
			n, err = updated.RowsAffected()
			if err == nil && n != 1 {
				err = ErrDocumentLease
			}
		}
	case "release":
		err = releaseDocumentLease(ctx, tx, id, userID, token)
	default:
		return nil, ErrDocumentLease
	}
	if err != nil {
		return nil, err
	}
	if action != "release" {
		result.Token = token
		err = tx.QueryRowContext(ctx, `SELECT expires_at FROM TED.UGO_EDIT_LOCK WHERE `+ownedDocumentLease, leaseArgs(id, userID, token)...).Scan(&result.ExpiresAt)
		if err != nil {
			return nil, err
		}
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return result, nil
}
