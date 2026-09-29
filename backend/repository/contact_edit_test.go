package repository

import (
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/tijanadmi/ugo_evid/models"
)

type editTestTx struct{ committed, rolledBack bool }

func (tx *editTestTx) Commit() error   { tx.committed = true; return nil }
func (tx *editTestTx) Rollback() error { tx.rolledBack = true; return nil }

func editStore(t *testing.T, query func(string, []driver.NamedValue) (driver.Rows, error), exec func(string, []driver.NamedValue) (driver.Result, error)) (*OracleStore, *editTestTx) {
	t.Helper()
	tx := &editTestTx{}
	store := testSchemaStore(t, &schemaConn{begin: func() (driver.Tx, error) { return tx, nil }, query: func(q string, a []driver.NamedValue) (driver.Rows, error) { checkBinds(t, q, a); return query(q, a) }, exec: func(q string, a []driver.NamedValue) (driver.Result, error) { checkBinds(t, q, a); return exec(q, a) }})
	return store, tx
}
func editRows(values ...driver.Value) driver.Rows {
	return &schemaRows{width: len(values), values: [][]driver.Value{values}}
}

func TestContactWriteGuards(t *testing.T) {
	for _, tc := range []struct {
		name                   string
		lease, version, linked int64
		deleting               bool
		want                   error
	}{
		{"expired or wrong token", 0, 3, 0, false, ErrContactLease},
		{"stale version", 1, 2, 0, false, ErrContactVersion},
		{"delete stale version", 1, 2, 0, true, ErrContactVersion},
		{"delete expired lease", 0, 3, 0, true, ErrContactLease},
		{"linked contact", 1, 3, 1, true, ErrContactLinked},
		{"update", 1, 3, 0, false, nil},
		{"delete", 1, 3, 0, true, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			writes, releases := 0, 0
			store, tx := editStore(t, func(q string, a []driver.NamedValue) (driver.Rows, error) {
				switch {
				case strings.Contains(q, "FOR UPDATE NOWAIT"):
					return editRows(int64(3), int64(9)), nil
				case strings.Contains(q, "FROM TED.UGO_EDIT_LOCK"):
					if !strings.Contains(q, "expires_at > SYSTIMESTAMP") || !strings.Contains(q, "user_id = :user_id") || !strings.Contains(q, "lock_token = :token") {
						t.Fatal("lease not guarded")
					}
					if a[1].Value != 7 || a[2].Value != strings.Repeat("a", 64) {
						t.Fatal("incorrect lease owner")
					}
					return editRows(tc.lease), nil
				case strings.Contains(q, "FROM TED.UGO_EVID"):
					return editRows(tc.linked), nil
				}
				t.Fatal(q)
				return nil, nil
			}, func(q string, a []driver.NamedValue) (driver.Result, error) {
				if strings.Contains(q, "DELETE FROM TED.UGO_EDIT_LOCK") {
					releases++
					return driver.RowsAffected(1), nil
				}
				if !strings.Contains(q, "version = :version") {
					t.Fatal("missing version predicate")
				}
				writes++
				return driver.RowsAffected(1), nil
			})
			var err error
			if tc.deleting {
				err = store.DeleteUgoDobLiceById(context.Background(), 8, tc.version, 7, strings.Repeat("a", 64))
			} else {
				m := &models.UgoDobLice{ID: 8, Version: tc.version, SapDobavljac: models.SapDobavljac{ID: 9}, Ime: "Ana", Status: "A"}
				_, err = store.UpdateUgoDobLice(context.Background(), m, 7, strings.Repeat("a", 64))
				if err == nil && m.Version != 4 {
					t.Fatal("version was not incremented")
				}
			}
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v want %v", err, tc.want)
			}
			if tc.want != nil {
				if writes != 0 || releases != 0 || tx.committed || !tx.rolledBack {
					t.Fatal("failed write changed data")
				}
			} else if writes != 1 || releases != 1 || !tx.committed {
				t.Fatal("write and release must commit together")
			}
		})
	}
}

func TestContactAcquireAndRenew(t *testing.T) {
	for _, tc := range []struct {
		name, action    string
		occupied, valid int64
		want            error
	}{
		{"acquire free", "acquire", 0, 0, nil},
		{"second editor", "acquire", 1, 0, ErrContactLocked},
		{"renew owned", "renew", 0, 1, nil},
		{"renew expired or foreign", "renew", 0, 0, ErrContactLease},
		{"release old token", "release", 0, 0, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var generated string
			inserts, renewals, releases, expiredDeletes := 0, 0, 0, 0
			store, tx := editStore(t, func(q string, a []driver.NamedValue) (driver.Rows, error) {
				if strings.Contains(q, "FOR UPDATE NOWAIT") {
					return editRows(int64(3), int64(9)), nil
				}
				if strings.Contains(q, "SELECT expires_at") {
					return editRows(time.Now().Add(2 * time.Minute)), nil
				}
				if strings.Contains(q, "expires_at > SYSTIMESTAMP") {
					return editRows(tc.valid), nil
				}
				return editRows(tc.occupied), nil
			}, func(q string, a []driver.NamedValue) (driver.Result, error) {
				switch {
				case strings.Contains(q, "INSERT INTO"):
					inserts++
					generated = a[2].Value.(string)
					if len(generated) != 64 || generated == strings.Repeat("a", 64) {
						t.Fatal("token not freshly generated")
					}
				case strings.Contains(q, "UPDATE TED.UGO_EDIT_LOCK"):
					renewals++
					if !strings.Contains(q, "expires_at > SYSTIMESTAMP") {
						t.Fatal("expired lease can be revived")
					}
				case strings.Contains(q, "expires_at <= SYSTIMESTAMP"):
					expiredDeletes++
				default:
					releases++
					if !strings.Contains(q, "lock_token = :token") || !strings.Contains(q, "user_id = :user_id") {
						t.Fatal("release can delete another owner's lock")
					}
				}
				return driver.RowsAffected(1), nil
			})
			lease, err := store.ContactLock(context.Background(), 8, 7, tc.action, strings.Repeat("a", 64))
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v want %v", err, tc.want)
			}
			if tc.want != nil {
				if tx.committed || !tx.rolledBack || inserts+renewals+releases != 0 {
					t.Fatal("conflicting lease was changed")
				}
				return
			}
			if !tx.committed {
				t.Fatal("lease not committed")
			}
			if tc.action == "acquire" && (expiredDeletes != 1 || inserts != 1 || lease.Token != generated) {
				t.Fatal("acquire must clear only expired lease and issue token")
			}
			if tc.action == "renew" && renewals != 1 {
				t.Fatal("lease not renewed")
			}
		})
	}
}

func TestContactTransactionContention(t *testing.T) {
	store, tx := editStore(t, func(string, []driver.NamedValue) (driver.Rows, error) {
		return nil, fmt.Errorf("ORA-00054: resource busy")
	}, func(string, []driver.NamedValue) (driver.Result, error) { t.Fatal("unexpected write"); return nil, nil })
	_, err := store.ContactLock(context.Background(), 8, 7, "acquire", "")
	if !errors.Is(err, ErrContactLocked) || !tx.rolledBack {
		t.Fatal(err)
	}
}

func TestContactWriteFailureRollsBackWithoutRelease(t *testing.T) {
	store, tx := editStore(t, func(q string, a []driver.NamedValue) (driver.Rows, error) {
		if strings.Contains(q, "FOR UPDATE") {
			return editRows(int64(3), int64(9)), nil
		}
		return editRows(int64(1)), nil
	}, func(q string, a []driver.NamedValue) (driver.Result, error) {
		if strings.Contains(q, "DELETE FROM TED.UGO_EDIT_LOCK") {
			t.Fatal("release after failed write")
		}
		return nil, fmt.Errorf("database unavailable")
	})
	_, err := store.UpdateUgoDobLice(context.Background(), &models.UgoDobLice{ID: 8, Version: 3, SapDobavljac: models.SapDobavljac{ID: 9}}, 7, strings.Repeat("a", 64))
	if err == nil || tx.committed || !tx.rolledBack {
		t.Fatal("failed mutation must rollback")
	}
}
