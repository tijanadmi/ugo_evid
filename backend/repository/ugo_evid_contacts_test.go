package repository

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"github.com/tijanadmi/ugo_evid/models"
	"strings"
	"testing"
)

func TestContactsBatchMapping(t *testing.T) {
	a, b, c := 10, 20, 30
	items := []models.UgoEvidProsireni{{IDSapDobavljac: &a}, {IDSapDobavljac: &a}, {IDSapDobavljac: &b}, {}, {IDSapDobavljac: &c}}
	calls := 0
	store := testSchemaStore(t, &schemaConn{query: func(q string, args []driver.NamedValue) (driver.Rows, error) {
		calls++
		checkBinds(t, q, args)
		if len(args) != 3 || !strings.Contains(q, "ORDER BY r.id, l.id") {
			t.Fatalf("query=%s args=%v", q, args)
		}
		return &schemaRows{width: 7, values: [][]driver.Value{{int64(20), "B", nil, nil, nil, nil, int64(21)}, {int64(10), "A1", "Manager", "123", "a@example.test", "SLM", int64(11)}, {int64(10), "A2", nil, nil, nil, "Other", int64(12)}}}, nil
	}})
	store.DB.SetMaxOpenConns(1)
	if err := store.loadProsireniContacts(context.Background(), items); err != nil {
		t.Fatal(err)
	}
	if calls != 1 || len(items[0].LicaDobavljaca) != 2 || len(items[1].LicaDobavljaca) != 2 || items[2].LicaDobavljaca[0].Ime != "B" {
		t.Fatalf("incorrect mapping: %+v", items)
	}
	if items[3].LicaDobavljaca == nil || len(items[3].LicaDobavljaca) != 0 || items[4].LicaDobavljaca == nil || len(items[4].LicaDobavljaca) != 0 {
		t.Fatal("missing empty arrays")
	}
}

func TestProsireniDetailLookup(t *testing.T) {
	store := testSchemaStore(t, &schemaConn{query: func(q string, args []driver.NamedValue) (driver.Rows, error) {
		if !strings.Contains(q, "v.id_ugo_evid = :id") || args[0].Value != 7 {
			t.Fatal("wrong detail lookup")
		}
		row := make([]driver.Value, 43)
		row[0], row[1], row[2] = int64(7), int64(3), int64(99)
		return &schemaRows{width: 43, values: [][]driver.Value{row}}, nil
	}})
	item, err := store.GetUgoEvidProsireniByID(context.Background(), 7)
	if err != nil || item.IDUgoEvid != 7 || item.IDUgoDobLica != nil || item.LicaDobavljaca == nil {
		t.Fatalf("item=%+v err=%v", item, err)
	}
	empty := testSchemaStore(t, &schemaConn{query: func(string, []driver.NamedValue) (driver.Rows, error) { return &schemaRows{width: 43}, nil }})
	if _, err = empty.GetUgoEvidProsireniByID(context.Background(), 7); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("missing detail: %v", err)
	}
}
