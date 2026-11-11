package repository

import (
	"context"
	"database/sql/driver"
	"strings"
	"testing"

	"github.com/tijanadmi/ugo_evid/models"
)

func TestLoadAccessMergesOrganizationRights(t *testing.T) {
	queries := 0
	store := testSchemaStore(t, &schemaConn{query: func(q string, args []driver.NamedValue) (driver.Rows, error) {
		queries++
		checkBinds(t, q, args)
		if queries == 1 {
			if !strings.Contains(q, "k.status=:status") || !strings.Contains(q, "kr.status='A'") || !strings.Contains(q, "GLOBALNO") {
				t.Fatal("missing active-user/calendar checks")
			}
			return &schemaRows{width: 2, values: [][]driver.Value{{int64(2), int64(1)}}}, nil
		}
		if !strings.Contains(q, "MAX(CASE") || !strings.Contains(q, "kr.opseg='SVE_OJ'") || !strings.Contains(q, "GROUP BY") {
			t.Fatal("grants must be additive, including all OJ")
		}
		return &schemaRows{width: 4, values: [][]driver.Value{{int64(1), "OJ1", "First", int64(0)}, {int64(4), "OJ4", "Fourth", int64(1)}}}, nil
	}})
	a, err := store.LoadAccess(context.Background(), "user", "A")
	if err != nil || !a.CalendarAdmin || a.UserID != 2 || !a.Allows(1, false) || a.Allows(1, true) || !a.Allows(4, true) || a.Allows(5, false) {
		t.Fatalf("access=%+v err=%v", a, err)
	}
}

func TestAccessPredicatesApplyBeforePagination(t *testing.T) {
	for _, kind := range []string{"contracts", "export", "partners", "reminders", "legacy", "organizations"} {
		t.Run(kind, func(t *testing.T) {
			calls := 0
			store := testSchemaStore(t, &schemaConn{query: func(q string, args []driver.NamedValue) (driver.Rows, error) {
				calls++
				checkBinds(t, q, args)
				if !strings.Contains(q, "=4") {
					t.Fatal("selected organization filter is missing")
				}
				for _, part := range []string{"ak.id_ugo_kor=2", "ak.status='A'", "ar.sifra IN ('GLE','IUD')", "ak.opseg='SVE_OJ'"} {
					if !strings.Contains(q, part) {
						t.Fatalf("%s: missing %s in %s", kind, part, q)
					}
				}
				if strings.HasPrefix(q, "SELECT COUNT(*)") {
					return &schemaRows{width: 1, values: [][]driver.Value{{int64(0)}}}, nil
				}
				return &schemaRows{width: 1}, nil
			}})
			ctx := models.WithAccess(context.Background(), &models.Access{UserID: 2, SelectedOrg: 4})
			var err error
			switch kind {
			case "contracts":
				_, _, err = store.GetUgoEvidProsireniPaged(ctx, true, 0, 20, 1, "")
			case "export":
				_, err = store.ExportOpenContracts(ctx, 1, "")
			case "partners":
				_, _, err = store.GetPartnersPaged(ctx, 1, 0, 20, "")
			case "reminders":
				_, err = store.ListReminders(ctx, 1)
			case "legacy":
				_, _, err = store.GetUgoEvidPaged(ctx, 1, 0, 20, "")
			case "organizations":
				_, _, err = store.GetUgoOrgPaged(ctx, 0, 20, "")
			}
			if err != nil || calls == 0 {
				t.Fatalf("calls=%d err=%v", calls, err)
			}
		})
	}
}

func TestSupplierWriteUsesIUDScope(t *testing.T) {
	store := testSchemaStore(t, &schemaConn{query: func(q string, args []driver.NamedValue) (driver.Rows, error) {
		checkBinds(t, q, args)
		if !strings.Contains(q, "ar.sifra IN ('IUD')") || strings.Contains(q, "'GLE'") || !strings.Contains(q, "su.id_sap_dobavljac = :supplier_id") {
			t.Fatal("supplier write must require IUD on a linked contract")
		}
		return &schemaRows{width: 1, values: [][]driver.Value{{int64(0)}}}, nil
	}})
	allowed, err := store.CanManagePartner(models.WithAccess(context.Background(), &models.Access{UserID: 2, WriteRequest: true}), 1, 20)
	if err != nil || allowed {
		t.Fatalf("allowed=%v err=%v", allowed, err)
	}
}
