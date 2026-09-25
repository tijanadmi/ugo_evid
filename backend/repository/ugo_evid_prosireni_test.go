package repository

import (
	"context"
	"database/sql/driver"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestProsireniViewMappingAndPredicates(t *testing.T) {
	for _, open := range []bool{true, false} {
		label := "closed"
		if open {
			label = "open"
		}
		t.Run(label, func(t *testing.T) {
			calls := 0
			now := time.Date(2026, 9, 17, 0, 0, 0, 0, time.UTC)
			store := testSchemaStore(t, &schemaConn{query: func(q string, args []driver.NamedValue) (driver.Rows, error) {
				calls++
				if strings.Contains(q, "FROM TED.UGO_DOB_LICA l") {
					checkBinds(t, q, args)
					if len(args) != 1 || args[0].Value != 400 {
						t.Fatalf("supplier binds: %v", args)
					}
					return &schemaRows{width: 7, values: [][]driver.Value{{int64(400), "SLM", nil, "011", "slm@example.test", "Service Level Manager", int64(81)}}}, nil
				}
				predicate := "v.otvoren_ug = 'X'"
				if !open {
					predicate = "v.otvoren_ug IS NULL"
				}
				if !strings.Contains(q, "FROM TED.UGO_EVID_PROSIRENI_V v WHERE "+predicate) {
					t.Fatalf("wrong view/filter: %s", q)
				}
				if strings.HasPrefix(q, "SELECT COUNT(*)") {
					return &schemaRows{width: 1, values: [][]driver.Value{{int64(12)}}}, nil
				}
				checkBinds(t, q, args)
				if args[0].Value != 10 || args[1].Value != 5 {
					t.Fatalf("pagination: %v", args)
				}
				if !strings.Contains(q, "ORDER BY v.id_ugo_evid DESC") {
					t.Fatal("missing stable order")
				}
				row := []driver.Value{
					int64(100), int64(2), int64(300), int64(400), "2026", "U-1", "DMS", "JN", "PLAN", "Predmet", "Dobavljac", "X", "Z",
					now, nil, float64(125.5), "RSD", nil, "KG", "MB", "Komerc", "Grupa", "Vrsta", "SL",
					"1", "Prvi", "2", "Drugi", "3", "Treci", "4", "Cetvrti", "5", "Peti", "6", "Sesti",
					"Kontakt", nil, "kontakt@example.test", "A", now, nil, int64(81), "Ulica 12", "Beograd",
				}
				if !open {
					row[11] = nil
				}
				return &schemaRows{width: 45, values: [][]driver.Value{row}}, nil
			}})
			items, total, err := store.GetUgoEvidProsireniPaged(context.Background(), open, 10, 5, 0)
			if err != nil {
				t.Fatal(err)
			}
			if calls != 3 || total != 12 || len(items) != 1 {
				t.Fatalf("calls=%d total=%d items=%d", calls, total, len(items))
			}
			m := items[0]
			if m.Adresa != "Ulica 12" || m.Grad != "Beograd" {
				t.Fatal("supplier address mapping")
			}
			if m.IDUgoDobLica == nil || *m.IDUgoDobLica != 81 || m.LicaDobavljaca[0].ID != 81 {
				t.Fatal("contact ID mapping")
			}
			if len(m.LicaDobavljaca) != 1 || m.LicaDobavljaca[0].RolaLica != "Service Level Manager" || m.LicaDobavljaca[0].RadnoMesto != "" {
				t.Fatal("supplier contacts mapping")
			}
			if m.IDUgoEvid != 100 || m.IDUgoOrg != 2 || m.IDSapUgovor != 300 || m.IDSapDobavljac == nil || *m.IDSapDobavljac != 400 {
				t.Fatal("ID mapping")
			}
			if m.PocetakUG == nil || !m.PocetakUG.Equal(now) || m.KrajUG != nil || m.DatIzm != nil || m.KursUG != nil {
				t.Fatal("nullable dates/numbers")
			}
			if m.VrednostUG == nil || *m.VrednostUG != 125.5 || m.Naziv != "Dobavljac" || m.Sluzba != "SL" || m.Telefon != "" || m.Email != "kontakt@example.test" {
				t.Fatal("value mapping")
			}
			if m.OdgZap1 != "1" || m.NazivOdgZap2 != "Drugi" || m.OdgZap3 != "3" || m.NazivOdgZap4 != "Cetvrti" || m.OdgZap5 != "5" || m.NazivOdgZap6 != "Sesti" {
				t.Fatal("responsible persons mapping")
			}
			if !open && m.OtvorenUG != "" {
				t.Fatal("NULL open flag")
			}
		})
	}
}

func TestProsireniEmptyPageKeepsTotal(t *testing.T) {
	store := testSchemaStore(t, &schemaConn{query: func(q string, _ []driver.NamedValue) (driver.Rows, error) {
		if strings.HasPrefix(q, "SELECT COUNT(*)") {
			return &schemaRows{width: 1, values: [][]driver.Value{{int64(3)}}}, nil
		}
		return &schemaRows{width: 45}, nil
	}})
	items, total, err := store.GetUgoEvidProsireniPaged(context.Background(), true, 100, 20, 0)
	if err != nil || total != 3 || items == nil || len(items) != 0 {
		t.Fatalf("items=%v total=%d err=%v", items, total, err)
	}
}

func TestProsireniDatabaseFailure(t *testing.T) {
	want := errors.New("view unavailable")
	store := testSchemaStore(t, &schemaConn{query: func(string, []driver.NamedValue) (driver.Rows, error) { return nil, want }})
	if _, _, err := store.GetUgoEvidProsireniPaged(context.Background(), false, 0, 20, 0); !errors.Is(err, want) {
		t.Fatalf("got %v", err)
	}
}

func TestProsireniOrganizationFilter(t *testing.T) {
	for _, open := range []bool{true, false} {
		for _, orgID := range []int{0, 7} {
			calls := 0
			store := testSchemaStore(t, &schemaConn{query: func(q string, args []driver.NamedValue) (driver.Rows, error) {
				calls++
				checkBinds(t, q, args)
				hasFilter := strings.Contains(q, "AND v.id_ugo_org = :org_id")
				if hasFilter != (orgID > 0) {
					t.Fatalf("incorrect organization filter: %s", q)
				}
				if orgID > 0 && (args[0].Name != "org_id" || args[0].Value != orgID) {
					t.Fatalf("wrong organization bind: %v", args)
				}
				if strings.HasPrefix(q, "SELECT COUNT(*)") {
					return &schemaRows{width: 1, values: [][]driver.Value{{int64(6)}}}, nil
				}
				return &schemaRows{width: 45}, nil
			}})
			items, total, err := store.GetUgoEvidProsireniPaged(context.Background(), open, 100, 20, orgID)
			if err != nil || calls != 2 || total != 6 || len(items) != 0 {
				t.Fatalf("calls=%d total=%d err=%v", calls, total, err)
			}
		}
	}
}
