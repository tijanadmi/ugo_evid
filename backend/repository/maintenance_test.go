package repository

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"github.com/tijanadmi/ugo_evid/models"
	"strings"
	"testing"
	"time"
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
				case strings.Contains(q, "TO_CHAR(s.pocetak_ug"):
					return editRows("2026-01-01", "2026-12-31"), nil
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
				if strings.Contains(q, "INSERT INTO TED.UGO_ODRZAVANJE") {
					*a[len(a)-1].Value.(sql.Out).Dest.(*int) = 10
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
				if !tx.committed || (tc.changeType && writes != 1) || (!tc.changeType && writes != 2) {
					t.Fatal("write not committed")
				}
			} else if tx.committed || !tx.rolledBack {
				t.Fatal("failed transaction committed")
			}
		})
	}
}

func TestMaintenanceUpdateAtomicFunds(t *testing.T) {
	for _, tc := range []struct {
		name                  string
		lease, version        int64
		hours                 int
		development, failFund bool
		want                  error
	}{
		{name: "update", lease: 1, version: 2, hours: 100, development: true},
		{name: "expired lease", version: 2, hours: 100, development: true, want: ErrMaintenanceLease},
		{name: "stale version", lease: 1, version: 3, hours: 100, development: true, want: ErrMaintenanceConflict},
		{name: "empty development fund", lease: 1, version: 2, development: true, want: ErrMaintenanceInput},
		{name: "hours without development", lease: 1, version: 2, hours: 100, want: ErrMaintenanceInput},
		{name: "rollback funds", lease: 1, version: 2, hours: 100, development: true, failFund: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			metadata, funds, released := false, false, false
			store, tx := editStore(t, func(q string, a []driver.NamedValue) (driver.Rows, error) {
				checkBinds(t, q, a)
				switch {
				case strings.Contains(q, "FROM TED.UGO_EVID WHERE"):
					if !strings.Contains(q, "FOR UPDATE NOWAIT") {
						t.Fatal("missing parent lock")
					}
					return editRows(int64(1), int64(1), int64(3), int64(101)), nil
				case strings.Contains(q, "TO_CHAR(s.pocetak_ug"):
					return editRows("2026-01-01", "2026-12-31"), nil
				case strings.Contains(q, "FROM TED.SAP_UGOVORI"):
					return editRows("X"), nil
				case strings.Contains(q, "FROM TED.UGO_TIP_UGOVORA"):
					return editRows("ODRZAVANJE"), nil
				case strings.Contains(q, "FROM TED.UGO_TIP_DOC"), strings.Contains(q, "FROM TED.UGO_RAZ_VRSTA_RADA"):
					return editRows(int64(1)), nil
				case strings.Contains(q, "FROM TED.UGO_EDIT_LOCK"):
					if !strings.Contains(q, "expires_at > SYSTIMESTAMP") || !strings.Contains(q, "user_id = :user_id") || !strings.Contains(q, "lock_token = :token") {
						t.Fatal("lease ownership/expiry missing")
					}
					return editRows(tc.lease), nil
				case strings.Contains(q, "FROM TED.UGO_ODRZAVANJE"):
					return editRows(int64(10), tc.version), nil
				}
				t.Fatal(q)
				return nil, nil
			}, func(q string, a []driver.NamedValue) (driver.Result, error) {
				checkBinds(t, q, a)
				switch {
				case strings.Contains(q, "UPDATE TED.UGO_ODRZAVANJE"):
					metadata = true
					if !strings.Contains(q, "version=version+1") || strings.Contains(q, "datum_potpisa=") {
						t.Fatal("version or legacy date protection missing")
					}
				case strings.Contains(q, "UPDATE TED.UGO_RAZ_FOND"):
				case strings.Contains(q, "MERGE INTO TED.UGO_RAZ_FOND"):
					funds = true
					if tc.failFund {
						return nil, errors.New("fund write failed")
					}
				case strings.Contains(q, "DELETE FROM TED.UGO_EDIT_LOCK"):
					released = true
				default:
					t.Fatal(q)
				}
				return driver.RowsAffected(1), nil
			})
			err := store.UpdateMaintenance(context.Background(), 9, models.Maintenance{Version: 2, Token: strings.Repeat("a", 64), DocumentType: 5, Development: tc.development, Funds: []models.DevelopmentFund{{TypeID: 1, Hours: tc.hours}}}, 3, 7)
			if tc.failFund {
				if err == nil || !metadata || !funds || released || tx.committed || !tx.rolledBack {
					t.Fatal("partial write not rolled back")
				}
				return
			}
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v want %v", err, tc.want)
			}
			if err == nil {
				if !metadata || !funds || !released || !tx.committed {
					t.Fatal("incomplete update")
				}
			} else if metadata || tx.committed {
				t.Fatal("conflict wrote data")
			}
		})
	}
}
func TestMaintenanceLeaseOwnership(t *testing.T) {
	for _, tc := range []struct {
		name, action string
		count        int64
		want         error
	}{
		{"acquire", "acquire", 0, nil}, {"busy", "acquire", 1, ErrMaintenanceLocked},
		{"renew", "renew", 1, nil}, {"expired or foreign token", "renew", 0, ErrMaintenanceLease}, {"release", "release", 0, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			writes := 0
			store, tx := editStore(t, func(q string, a []driver.NamedValue) (driver.Rows, error) {
				checkBinds(t, q, a)
				switch {
				case strings.Contains(q, "SELECT id_ugo_org FROM"):
					return editRows(int64(3)), nil
				case strings.Contains(q, "FROM TED.UGO_EVID"):
					return editRows(int64(1), int64(1), int64(3), int64(101)), nil
				case strings.Contains(q, "FROM TED.SAP_UGOVORI"):
					return editRows("X"), nil
				case strings.Contains(q, "FROM TED.UGO_ODRZAVANJE"):
					return editRows(int64(10)), nil
				case strings.Contains(q, "SELECT expires_at"):
					return editRows(time.Now().Add(time.Minute)), nil
				case strings.Contains(q, "COUNT(*) FROM TED.UGO_EDIT_LOCK"):
					return editRows(tc.count), nil
				}
				t.Fatal(q)
				return nil, nil
			}, func(q string, a []driver.NamedValue) (driver.Result, error) {
				checkBinds(t, q, a)
				writes++
				return driver.RowsAffected(1), nil
			})
			lease, err := store.MaintenanceLock(context.Background(), 9, 3, 7, tc.action, strings.Repeat("a", 64))
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v want %v", err, tc.want)
			}
			if err == nil {
				if !tx.committed || writes == 0 {
					t.Fatal("not persisted")
				}
				if tc.action != "release" && len(lease.Token) != 64 {
					t.Fatal("token missing")
				}
			} else if tx.committed {
				t.Fatal("conflict committed")
			}
		})
	}
}
func TestMaintenanceReadPreservesLegacyDatesAndFunds(t *testing.T) {
	store := testSchemaStore(t, &schemaConn{query: func(q string, a []driver.NamedValue) (driver.Rows, error) {
		checkBinds(t, q, a)
		switch {
		case strings.Contains(q, "JOIN TED.SAP_UGOVORI"):
			return editRows(int64(1), int64(2), "2026-01-01", "2026-12-31"), nil
		case strings.Contains(q, "UGO_RAZ_VRSTA_RADA"):
			return editRows(int64(1), "INZENJER", "Inzenjer"), nil
		case strings.Contains(q, "UGO_ODRZAVANJE"):
			return editRows(int64(10), "2025-12-20", "2026-12-30", int64(1), nil, int64(5), int64(7), int64(3), nil, int64(2)), nil
		case strings.Contains(q, "UGO_RAZ_FOND"):
			return editRows(int64(1), int64(100)), nil
		}
		t.Fatal(q)
		return nil, nil
	}})
	m, err := store.GetContractSettings(context.Background(), 9)
	if err != nil {
		t.Fatal(err)
	}
	if m.SAPStart != "2026-01-01" || m.Maintenance.Start != "" || m.Maintenance.Signed != "2025-12-20" || m.Maintenance.Funds[0].Hours != 100 || m.Maintenance.Months != nil {
		t.Fatalf("incorrect read %+v", m)
	}
}
