package repository

import (
	"context"
	"database/sql/driver"
	"errors"
	"github.com/tijanadmi/ugo_evid/models"
	"strings"
	"testing"
)

func TestMaintenanceTransactions(t *testing.T) {
	for _, tc := range []struct {
		name                                                                               string
		changeType, stale, existing, foreign, closed, wrongType, badDoc, duplicate, locked bool
		want                                                                               error
	}{
		{name: "create"}, {name: "type", changeType: true},
		{name: "stale type", changeType: true, stale: true, want: ErrMaintenanceConflict},
		{name: "type with maintenance", changeType: true, existing: true, want: ErrMaintenanceConflict},
		{name: "foreign", foreign: true, want: ErrUserOrganization},
		{name: "closed", closed: true, want: ErrDocumentClosed},
		{name: "wrong type", wrongType: true, want: ErrMaintenanceInput},
		{name: "bad document", badDoc: true, want: ErrMaintenanceInput},
		{name: "duplicate", duplicate: true, want: ErrMaintenanceConflict},
		{name: "concurrent writer", locked: true, want: ErrRegistrationBusy},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parentLocked := false
			writes := 0
			store, tx := editStore(t, func(q string, a []driver.NamedValue) (driver.Rows, error) {
				checkBinds(t, q, a)
				switch {
				case strings.Contains(q, "FROM TED.UGO_EVID"):
					if !strings.Contains(q, "FOR UPDATE NOWAIT") {
						t.Fatal("parent must serialize writes")
					}
					if tc.locked {
						return nil, errors.New("ORA-00054")
					}
					parentLocked = true
					org := int64(3)
					if tc.foreign {
						org = 4
					}
					return editRows(int64(1), int64(2), org, int64(101)), nil
				case strings.Contains(q, "FROM TED.SAP_UGOVORI"):
					if tc.closed {
						return editRows("-"), nil
					}
					return editRows("X"), nil
				case strings.Contains(q, "FROM TED.UGO_TIP_UGOVORA"):
					if tc.wrongType {
						return editRows("LICENCE"), nil
					}
					return editRows("ODRZAVANJE"), nil
				case strings.Contains(q, "FROM TED.UGO_ODRZAVANJE"):
					n := int64(0)
					if tc.existing {
						n = 1
					}
					return editRows(n), nil
				case strings.Contains(q, "FROM TED.UGO_TIP_DOC"):
					if !strings.Contains(q, "'POTVRDA_TEH','ZAPISNIK_TEH'") {
						t.Fatal("document types unrestricted")
					}
					n := int64(1)
					if tc.badDoc {
						n = 0
					}
					return editRows(n), nil
				}
				t.Fatal(q)
				return nil, nil
			}, func(q string, a []driver.NamedValue) (driver.Result, error) {
				checkBinds(t, q, a)
				if !parentLocked {
					t.Fatal("unlocked write")
				}
				writes++
				if tc.duplicate {
					return nil, errors.New("ORA-00001")
				}
				return driver.RowsAffected(1), nil
			})
			var err error
			if tc.changeType {
				version := int64(2)
				if tc.stale {
					version = 1
				}
				err = store.SetContractType(context.Background(), 9, 2, version, 3, 7)
			} else {
				err = store.CreateMaintenance(context.Background(), 9, models.Maintenance{Signed: "2026-01-01", Ends: "2026-12-31", DocumentType: 5, TechnicalReminder: 7}, 3, 7)
			}
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v want %v", err, tc.want)
			}
			if tc.want == nil {
				if !tx.committed || writes != 1 {
					t.Fatal("write not committed")
				}
			} else if tx.committed || !tx.rolledBack {
				t.Fatal("failed transaction committed")
			}
		})
	}
}
