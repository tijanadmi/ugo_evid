package repository

import (
	"context"
	"database/sql/driver"
	"errors"
	"strings"
	"testing"
)

func TestUserOrganization(t *testing.T) {
	for _, count := range []int{0, 1, 2} {
		store := testSchemaStore(t, &schemaConn{query: func(q string, args []driver.NamedValue) (driver.Rows, error) {
			checkBinds(t, q, args)
			if !strings.Contains(q, "kr.status = 'A'") || !strings.Contains(q, "k.status = :active_status") || args[0].Value != "ad.user" {
				t.Fatal("identity/active-role filter missing")
			}
			rows := &schemaRows{width: 1}
			for i := 0; i < count; i++ {
				rows.values = append(rows.values, []driver.Value{int64(i + 3)})
			}
			return rows, nil
		}})
		id, err := store.GetUserOrganization(context.Background(), "ad.user", "A")
		if count == 1 {
			if err != nil || id != 3 {
				t.Fatalf("id=%d err=%v", id, err)
			}
		} else if !errors.Is(err, ErrUserOrganization) {
			t.Fatalf("count=%d err=%v", count, err)
		}
	}
}

func TestPartnersScopedAndBatched(t *testing.T) {
	calls := 0
	store := testSchemaStore(t, &schemaConn{query: func(q string, args []driver.NamedValue) (driver.Rows, error) {
		calls++
		checkBinds(t, q, args)
		if strings.Contains(q, "FROM TED.UGO_DOB_LICA l") {
			return &schemaRows{width: 7, values: [][]driver.Value{{int64(20), "Kontakt", nil, "011", nil, "Service Level Manager", int64(71)}}}, nil
		}
		if !strings.Contains(q, "WHERE EXISTS") || !strings.Contains(q, "e.id_ugo_org = :org_id") || args[0].Value != 3 || strings.Contains(q, "otvoren_ug") {
			t.Fatalf("incorrect org scope: %s %v", q, args)
		}
		if strings.HasPrefix(q, "SELECT COUNT(*)") {
			return &schemaRows{width: 1, values: [][]driver.Value{{int64(2)}}}, nil
		}
		if args[1].Value != 0 || args[2].Value != 20 {
			t.Fatal("pagination")
		}
		return &schemaRows{width: 6, values: [][]driver.Value{{int64(20), "S1", "Partner 1", "Ulica", "Grad", nil}, {int64(21), "S2", "Partner 2", nil, nil, nil}}}, nil
	}})
	store.DB.SetMaxOpenConns(1)
	items, total, err := store.GetPartnersPaged(context.Background(), 3, 0, 20)
	if err != nil || total != 2 || len(items) != 2 || calls != 3 {
		t.Fatalf("items=%v total=%d calls=%d err=%v", items, total, calls, err)
	}
	if len(items[0].LicaDobavljaca) != 1 || items[0].LicaDobavljaca[0].ID != 71 || items[1].LicaDobavljaca == nil || len(items[1].LicaDobavljaca) != 0 {
		t.Fatal("contacts mapping")
	}
}

func TestPartnersEmptyPage(t *testing.T) {
	store := testSchemaStore(t, &schemaConn{query: func(q string, _ []driver.NamedValue) (driver.Rows, error) {
		if strings.HasPrefix(q, "SELECT COUNT(*)") {
			return &schemaRows{width: 1, values: [][]driver.Value{{int64(2)}}}, nil
		}
		return &schemaRows{width: 6}, nil
	}})
	items, total, err := store.GetPartnersPaged(context.Background(), 3, 100, 20)
	if err != nil || total != 2 || items == nil || len(items) != 0 {
		t.Fatalf("items=%v total=%d err=%v", items, total, err)
	}
}
