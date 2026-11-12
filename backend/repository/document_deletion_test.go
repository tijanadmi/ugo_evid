package repository

import (
	"context"
	"database/sql/driver"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tijanadmi/ugo_evid/models"
)

func TestFinishDocumentDeletion(t *testing.T) {
	for _, mode := range []string{"normal", "missing file", "file error", "database failure", "already finished", "not queued"} {
		t.Run(mode, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "file.pdf")
			if mode == "file error" {
				if err := os.Mkdir(path, 0700); err != nil {
					t.Fatal(err)
				}
			} else if mode != "missing file" {
				if err := os.WriteFile(path, []byte("file"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			root, err := os.OpenRoot(dir)
			if err != nil {
				t.Fatal(err)
			}
			defer root.Close()
			writes := 0
			store, tx := editStore(t, func(q string, args []driver.NamedValue) (driver.Rows, error) {
				if !strings.Contains(q, "FOR UPDATE WAIT 5") {
					return nil, errors.New("missing row lock")
				}
				if strings.Contains(q, "UGO_DOC_BRISANJE") {
					if mode == "not queued" {
						return &schemaRows{width: 1}, nil
					}
					if mode == "already finished" {
						return editRows(time.Now()), nil
					}
					return editRows(nil), nil
				}
				if !strings.Contains(q, "status='D'") {
					return nil, errors.New("missing pending guard")
				}
				return editRows("file.pdf"), nil
			}, func(q string, args []driver.NamedValue) (driver.Result, error) {
				writes++
				if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
					return nil, errors.New("database changed before file removal")
				}
				if mode == "database failure" {
					return nil, errors.New("DB unavailable")
				}
				return driver.RowsAffected(1), nil
			})
			err = store.FinishDocumentDeletion(context.Background(), 11, root)
			success := mode == "normal" || mode == "missing file" || mode == "already finished"
			if (err == nil) != success {
				t.Fatalf("err=%v", err)
			}
			if mode == "normal" || mode == "missing file" {
				if writes != 3 || !tx.committed {
					t.Fatal("delete/audit not committed together")
				}
			}
			if mode == "file error" || mode == "not queued" || mode == "already finished" {
				if writes != 0 {
					t.Fatal("unexpected write")
				}
			}
			if !success && tx.committed {
				t.Fatal("failed deletion committed")
			}
		})
	}
}

func TestDeletionIntentFailureRollsBack(t *testing.T) {
	failure := errors.New("audit write failed")
	store, tx := editStore(t, func(q string, args []driver.NamedValue) (driver.Rows, error) {
		if strings.Contains(q, "FOR UPDATE NOWAIT") {
			return editRows(int64(1), int64(9)), nil
		}
		return editRows(int64(1)), nil
	}, func(q string, args []driver.NamedValue) (driver.Result, error) {
		if strings.Contains(q, "INSERT INTO TED.UGO_DOC_BRISANJE") {
			return nil, failure
		}
		if strings.Contains(q, "DELETE") {
			return nil, errors.New("lease released before audit saved")
		}
		return driver.RowsAffected(1), nil
	})
	err := store.ChangeDocument(context.Background(), &models.Document{ID: 11, Version: 1}, 7, strings.Repeat("a", 64), true)
	if !errors.Is(err, failure) || tx.committed || !tx.rolledBack {
		t.Fatalf("err=%v tx=%+v", err, tx)
	}
}

func TestPendingDeletesExcludeHistoricalSoftDeletes(t *testing.T) {
	store := testSchemaStore(t, &schemaConn{query: func(q string, args []driver.NamedValue) (driver.Rows, error) {
		if !strings.Contains(q, "FROM TED.UGO_DOC_BRISANJE WHERE zavrseno IS NULL") {
			return nil, errors.New("must select only explicitly queued deletions")
		}
		return editRows(int64(11)), nil
	}})
	ids, err := store.PendingDocumentDeletions(context.Background())
	if err != nil || len(ids) != 1 || ids[0] != 11 {
		t.Fatalf("ids=%v err=%v", ids, err)
	}
}
