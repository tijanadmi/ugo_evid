package repository

import (
	"context"
	"database/sql"
	"errors"
	"os"

	"github.com/tijanadmi/ugo_evid/util"
)

type DocumentDeletionStore interface {
	PendingDocumentDeletions(context.Context) ([]int, error)
	FinishDocumentDeletion(context.Context, int, *os.Root) error
}

// Only explicitly queued deletions are processed. Historical soft-deleted D
// rows have no queue entry and are deliberately left untouched.
func (r *OracleStore) PendingDocumentDeletions(ctx context.Context) ([]int, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT id_doc FROM TED.UGO_DOC_BRISANJE WHERE zavrseno IS NULL ORDER BY trazeno,id_doc`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []int{}
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// The committed D state is the durable intent. Serialize API/worker instances
// on the audit row while deleting the file, then atomically remove the document
// and mark the audit complete. A failed/ambiguous commit is safe to retry.
func (r *OracleStore) FinishDocumentDeletion(ctx context.Context, id int, root *os.Root) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var done sql.NullTime
	err = tx.QueryRowContext(ctx, `SELECT zavrseno FROM TED.UGO_DOC_BRISANJE WHERE id_doc=:id FOR UPDATE WAIT 5`, sql.Named("id", id)).Scan(&done)
	if err != nil {
		return err
	}
	if done.Valid {
		return nil
	}
	var path string
	err = tx.QueryRowContext(ctx, `SELECT putanja FROM TED.UGO_DOC WHERE id=:id AND status='D' FOR UPDATE WAIT 5`, sql.Named("id", id)).Scan(&path)
	if err != nil {
		return err
	}
	if err := util.RemoveDocument(root, path); err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `DELETE FROM TED.UGO_DOC WHERE id=:id AND status='D'`, sql.Named("id", id))
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return errors.New("pending document was not deleted")
	}
	if _, err = tx.ExecContext(ctx, `UPDATE TED.UGO_DOC_BRISANJE SET zavrseno=SYSTIMESTAMP WHERE id_doc=:id`, sql.Named("id", id)); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE TED.PORTAL_OBAVESTENJE SET status='RESENO',reseno=SYSTIMESTAMP,version=version+1 WHERE vrsta='DOC_DELETE' AND id_sloga=:id AND status<>'RESENO'`, sql.Named("id", id)); err != nil {
		return err
	}
	return tx.Commit()
}
