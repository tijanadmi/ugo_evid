package repository

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"github.com/tijanadmi/ugo_evid/models"
	"strings"
	"testing"
)

func TestRequestDocumentReservation(t *testing.T) {
	for _, tc := range []struct {
		name         string
		missing      bool
		locks, valid int64
		want         error
	}{
		{name: "linked upload", valid: 1},
		{name: "foreign missing or deleted request", missing: true, want: sql.ErrNoRows},
		{name: "request being edited", locks: 1, valid: 1, want: ErrDocumentLocked},
		{name: "unsupported request document type", want: ErrDocumentType},
	} {
		t.Run(tc.name, func(t *testing.T) {
			writes := 0
			store, tx := editStore(t, func(q string, a []driver.NamedValue) (driver.Rows, error) {
				switch {
				case strings.Contains(q, "FOR UPDATE OF z.version NOWAIT"):
					if !strings.Contains(q, "o.id_ugo_evid=:contract_id") || !strings.Contains(q, "z.status<>'OBRISAN'") {
						return nil, fmt.Errorf("unscoped request")
					}
					if tc.missing {
						return &schemaRows{width: 1}, nil
					}
					return editRows(int64(5)), nil
				case strings.Contains(q, "UGO_EDIT_LOCK"):
					return editRows(tc.locks), nil
				case strings.Contains(q, "sifra IN"):
					return editRows(tc.valid), nil
				default:
					return editRows(int64(1)), nil
				}
			}, func(q string, a []driver.NamedValue) (driver.Result, error) {
				if !strings.Contains(q, "id_raz_zahtev") || !strings.Contains(q, "NULLIF(:request_id,0)") {
					return nil, fmt.Errorf("missing association")
				}
				for _, v := range a {
					if v.Name == "request_id" && v.Value != 5 {
						return nil, fmt.Errorf("wrong request id")
					}
				}
				writes++
				return driver.RowsAffected(1), nil
			})
			err := store.ReserveDocument(context.Background(), &models.Document{ContractID: 9, DocumentMetadata: models.DocumentMetadata{TypeID: 2, RequestID: 5}})
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v want %v", err, tc.want)
			}
			if tc.want == nil && (writes != 1 || !tx.committed) {
				t.Fatal("not committed")
			}
			if tc.want != nil && (writes != 0 || tx.committed || !tx.rolledBack) {
				t.Fatal("failed upload persisted")
			}
		})
	}
}

func TestPeriodDocumentReservation(t *testing.T) {
	for _, tc := range []struct {
		name         string
		missing      bool
		locks, valid int64
		want         error
	}{
		{name: "linked upload", valid: 1},
		{name: "foreign missing or deleted period", missing: true, want: sql.ErrNoRows},
		{name: "period being edited", locks: 1, valid: 1, want: ErrDocumentLocked},
		{name: "unsupported period document type", want: ErrDocumentType},
	} {
		t.Run(tc.name, func(t *testing.T) {
			writes := 0
			store, tx := editStore(t, func(q string, a []driver.NamedValue) (driver.Rows, error) {
				switch {
				case strings.Contains(q, "FOR UPDATE OF z.version NOWAIT"):
					if !strings.Contains(q, "o.id_ugo_evid=:contract_id") {
						return nil, fmt.Errorf("unscoped period")
					}
					if tc.missing {
						return &schemaRows{width: 1}, nil
					}
					return editRows(int64(5)), nil
				case strings.Contains(q, "UGO_EDIT_LOCK"):
					return editRows(tc.locks), nil
				case strings.Contains(q, "sifra IN"):
					return editRows(tc.valid), nil
				default:
					return editRows(int64(1)), nil
				}
			}, func(q string, a []driver.NamedValue) (driver.Result, error) {
				if !strings.Contains(q, "id_teh_period") || !strings.Contains(q, "NULLIF(:period_id,0)") {
					return nil, fmt.Errorf("missing association")
				}
				for _, v := range a {
					if v.Name == "period_id" && v.Value != 5 {
						return nil, fmt.Errorf("wrong period id")
					}
				}
				writes++
				return driver.RowsAffected(1), nil
			})
			err := store.ReserveDocument(context.Background(), &models.Document{ContractID: 9, DocumentMetadata: models.DocumentMetadata{TypeID: 2, PeriodID: 5}})
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v want %v", err, tc.want)
			}
			if tc.want == nil && (writes != 1 || !tx.committed) {
				t.Fatal("not committed")
			}
			if tc.want != nil && (writes != 0 || tx.committed || !tx.rolledBack) {
				t.Fatal("failed upload persisted")
			}
		})
	}
}
