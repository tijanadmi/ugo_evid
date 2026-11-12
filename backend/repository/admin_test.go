package repository

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"strings"
	"testing"

	"github.com/tijanadmi/ugo_evid/models"
)

func TestValidateAdminGrants(t *testing.T) {
	for _, tc := range []struct {
		name   string
		grants []models.RoleGrant
		valid  bool
	}{
		{"mixed", []models.RoleGrant{{RoleID: 1, Scope: "OJ", OrgID: 1}, {RoleID: 2, Scope: "OJ", OrgID: 4}, {RoleID: 4, Scope: "GLOBALNO"}}, true},
		{"all", []models.RoleGrant{{RoleID: 1, Scope: "SVE_OJ"}}, true},
		{"calendar", []models.RoleGrant{{RoleID: 3, Scope: "GLOBALNO"}}, true},
		{"admin OJ invalid", []models.RoleGrant{{RoleID: 4, Scope: "OJ", OrgID: 1}}, false},
		{"missing OJ", []models.RoleGrant{{RoleID: 2, Scope: "OJ"}}, false},
		{"global must not have OJ", []models.RoleGrant{{RoleID: 4, Scope: "GLOBALNO", OrgID: 1}}, false},
		{"duplicate", []models.RoleGrant{{RoleID: 1, Scope: "SVE_OJ"}, {RoleID: 1, Scope: "SVE_OJ"}}, false},
		{"unknown", []models.RoleGrant{{RoleID: 5, Scope: "GLOBALNO"}}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			u := models.AdminUser{Username: " Test.User ", Status: "A", Grants: tc.grants}
			err := validateAdminUser(&u, "A")
			if (err == nil) != tc.valid {
				t.Fatalf("err=%v", err)
			}
			if u.Username != "test.user" {
				t.Fatal("username not normalized")
			}
		})
	}
}

func TestAdminWritesAreSerializedAndAudited(t *testing.T) {
	for _, mode := range []string{"success", "last admin", "stale", "revoked actor", "audit failure", "zero updated rows", "create"} {
		t.Run(mode, func(t *testing.T) {
			queries, writes, audits := 0, 0, 0
			store, tx := editStore(t, func(q string, args []driver.NamedValue) (driver.Rows, error) {
				queries++
				if queries == 1 {
					if !strings.Contains(q, "PORTAL_ADMIN_LOCK WHERE id=1 FOR UPDATE WAIT 5") {
						return nil, errors.New("global lock must precede all reads")
					}
					return editRows(int64(1)), nil
				}
				if strings.Contains(q, "k.id=:actor") {
					if mode == "revoked actor" {
						return editRows(int64(0)), nil
					}
					return editRows(int64(1)), nil
				}
				if strings.Contains(q, "LEFT JOIN TED.UGO_KOR_ROLE") {
					return editRows(int64(2), "test.user", "Test", "A", int64(3), int64(4), "GLOBALNO", nil), nil
				}
				if strings.Contains(q, "SELECT COUNT(*) FROM TED.UGO_KOR") {
					if mode == "last admin" {
						return editRows(int64(0)), nil
					}
					return editRows(int64(1)), nil
				}
				return nil, errors.New("unexpected query")
			}, func(q string, args []driver.NamedValue) (driver.Result, error) {
				writes++
				if strings.Contains(q, "INSERT INTO TED.UGO_KOR (") {
					for _, arg := range args {
						if arg.Name == "id" {
							*arg.Value.(sql.Out).Dest.(*int) = 9
						}
					}
				}
				if strings.Contains(q, "UPDATE TED.UGO_KOR SET") && mode == "zero updated rows" {
					return driver.RowsAffected(0), nil
				}
				if strings.Contains(q, "PORTAL_ADMIN_AUDIT") {
					audits++
					if mode == "audit failure" {
						return nil, errors.New("audit unavailable")
					}
					if args[0].Value != 7 || !strings.Contains(args[5].Value.(string), "roles") {
						return nil, errors.New("audit missing actor or role snapshot")
					}
				}
				return driver.RowsAffected(1), nil
			})
			u := &models.AdminUser{ID: 2, Username: "test.user", Name: "Updated", Status: "A", Version: 3, Grants: []models.RoleGrant{{RoleID: 1, Scope: "OJ", OrgID: 1}}}
			if mode == "stale" {
				u.Version = 2
			}
			if mode == "create" {
				u.ID = 0
				u.Version = 0
			}
			err := store.SaveAdminUser(context.Background(), 7, "A", u)
			if mode == "success" || mode == "create" {
				if err != nil || !tx.committed || audits != 1 {
					t.Fatalf("err=%v audits=%d tx=%+v", err, audits, tx)
				}
			} else {
				if err == nil || tx.committed || !tx.rolledBack {
					t.Fatalf("failed write committed: %v %+v", err, tx)
				}
			}
			if mode == "last admin" && !errors.Is(err, ErrLastAdmin) {
				t.Fatal(err)
			}
			if (mode == "stale" || mode == "revoked actor") && writes != 0 {
				t.Fatal("unauthorized/stale write reached database")
			}
		})
	}
}

func TestADMINDoesNotGrantContractAccess(t *testing.T) {
	store := testSchemaStore(t, &schemaConn{query: func(q string, args []driver.NamedValue) (driver.Rows, error) {
		checkBinds(t, q, args)
		if strings.Contains(q, "SELECT k.id") {
			if !strings.Contains(q, "r.sifra IN ('ADM_KALENDAR','ADMIN')") {
				t.Fatal("ADMIN must include calendar")
			}
			return editRows(int64(7), int64(1), int64(1)), nil
		}
		if !strings.Contains(q, "r.sifra IN ('GLE','IUD')") {
			t.Fatal("ADMIN must not grant OJ access")
		}
		return &schemaRows{width: 4}, nil
	}})
	a, err := store.LoadAccess(context.Background(), "admin", "A")
	if err != nil || !a.Admin || !a.CalendarAdmin || len(a.Organizations) != 0 || a.Allows(1, true) {
		t.Fatalf("access=%+v err=%v", a, err)
	}
}
