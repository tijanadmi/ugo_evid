package repository

import (
	"context"
	"database/sql/driver"
	"errors"
	"github.com/tijanadmi/ugo_evid/models"
	"strings"
	"testing"
)

func TestDocumentMutationGuards(t *testing.T) {
	for _, remove := range []bool{false, true} {
		for _, tc := range []struct {
			lease, version, open int64
			want                 error
		}{
			{0, 3, 1, ErrDocumentLease}, {1, 2, 1, ErrDocumentVersion}, {1, 3, 0, ErrDocumentClosed}, {1, 3, 1, nil},
		} {
			writes, releases := 0, 0
			store, tx := editStore(t, func(q string, a []driver.NamedValue) (driver.Rows, error) {
				if strings.Contains(q, "FOR UPDATE NOWAIT") {
					if !strings.Contains(q, "status='A'") {
						t.Fatal(q)
					}
					return editRows(int64(3), int64(9)), nil
				}
				if strings.Contains(q, "UGO_EDIT_LOCK") {
					if !strings.Contains(q, "entity_type = 'UGO_DOC'") || !strings.Contains(q, "expires_at > SYSTIMESTAMP") || a[1].Value != 7 || a[2].Value != strings.Repeat("a", 64) {
						t.Fatal("unguarded document lease")
					}
					return editRows(tc.lease), nil
				}
				if strings.Contains(q, "UGO_EVID_PROSIRENI_V") {
					return editRows(tc.open), nil
				}
				if strings.Contains(q, "UGO_TIP_DOC") {
					return editRows(int64(1)), nil
				}
				t.Fatal(q)
				return nil, nil
			}, func(q string, a []driver.NamedValue) (driver.Result, error) {
				if strings.Contains(q, "DELETE FROM TED.UGO_EDIT_LOCK") {
					releases++
					return driver.RowsAffected(1), nil
				}
				if strings.Contains(q, "DELETE FROM TED.UGO_DOC") {
					t.Fatal("physical deletion")
				}
				if !strings.Contains(q, "version=:version") || !strings.Contains(q, "version=version+1") {
					t.Fatal(q)
				}
				if remove && !strings.Contains(q, "status='D'") {
					t.Fatal(q)
				}
				writes++
				return driver.RowsAffected(1), nil
			})
			err := store.ChangeDocument(context.Background(), &models.Document{ID: 8, Version: tc.version, DocumentMetadata: models.DocumentMetadata{TypeID: 1, Name: "Doc"}}, 7, strings.Repeat("a", 64), remove)
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v want %v", err, tc.want)
			}
			if tc.want != nil && (writes != 0 || releases != 0 || tx.committed) {
				t.Fatal("failed mutation wrote data")
			}
			if tc.want == nil && (writes != 1 || releases != 1 || !tx.committed) {
				t.Fatal("mutation/lease not atomic")
			}
		}
	}
}
func TestDocumentFolderUsesPersistedName(t *testing.T) {
	store, tx := editStore(t, func(q string, _ []driver.NamedValue) (driver.Rows, error) {
		if !strings.Contains(q, "FOR UPDATE WAIT 5") {
			t.Fatal("folder allocation not serialized")
		}
		return editRows("7_existing_folder"), nil
	}, func(string, []driver.NamedValue) (driver.Result, error) {
		t.Fatal("renamed existing folder")
		return nil, nil
	})
	folder, err := store.DocumentFolder(context.Background(), 7, "7_changed_subject")
	if err != nil || folder != "7_existing_folder" || !tx.committed {
		t.Fatalf("%s %v", folder, err)
	}
}
func TestDocumentLockRejectsOtherTab(t *testing.T) {
	store, tx := editStore(t, func(q string, _ []driver.NamedValue) (driver.Rows, error) {
		if strings.Contains(q, "FOR UPDATE NOWAIT") {
			return editRows(int64(1), int64(9)), nil
		}
		return editRows(int64(1)), nil
	}, func(q string, _ []driver.NamedValue) (driver.Result, error) {
		if !strings.Contains(q, "expires_at <= SYSTIMESTAMP") {
			t.Fatal("changed live lock")
		}
		return driver.RowsAffected(0), nil
	})
	_, err := store.DocumentLock(context.Background(), 8, 7, "acquire", "")
	if !errors.Is(err, ErrDocumentLocked) || tx.committed {
		t.Fatalf("%v", err)
	}
}
