package repository

import (
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
	"github.com/tijanadmi/ugo_evid/models"
	"strings"
	"testing"
)

func TestPeriodGeneration(t *testing.T) {
	for _, tc := range []struct {
		name, start, end, mode string
		months                 int
		want                   []string
	}{
		{"calendar partial month", "2026-02-15", "2026-04-10", "calendar", 1, []string{"2026-02-15/2026-02-28", "2026-03-01/2026-03-31", "2026-04-01/2026-04-10"}},
		{"calendar quarter", "2026-02-15", "2026-08-01", "calendar", 3, []string{"2026-02-15/2026-03-31", "2026-04-01/2026-06-30", "2026-07-01/2026-08-01"}},
		{"anchor month end no drift", "2024-01-31", "2024-04-01", "anchor", 1, []string{"2024-01-31/2024-02-28", "2024-02-29/2024-03-30", "2024-03-31/2024-04-01"}},
		{"anchor quarter", "2026-02-15", "2026-07-01", "anchor", 3, []string{"2026-02-15/2026-05-14", "2026-05-15/2026-07-01"}},
		{"same day", "2026-02-15", "2026-02-15", "anchor", 12, []string{"2026-02-15/2026-02-15"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			items, e := GeneratePeriods(tc.start, tc.end, tc.months, 5, tc.mode)
			if e != nil || len(items) != len(tc.want) {
				t.Fatalf("%+v %v", items, e)
			}
			for i, p := range items {
				if p.Start+"/"+p.End != tc.want[i] {
					t.Fatalf("%+v", p)
				}
			}
		})
	}
	if _, e := GeneratePeriods("2026-02-30", "2026-12-31", 1, 5, "calendar"); e == nil {
		t.Fatal("invalid date accepted")
	}
}
func TestWorkingDeadline(t *testing.T) {
	calendar := map[string]bool{"2026-02-14": false, "2026-02-15": false, "2026-02-16": false, "2026-02-17": false, "2026-02-18": true, "2026-02-19": true}
	d, e := requestDeadline("2026-02-13", 2, calendar)
	if e != nil || d != "2026-02-19" {
		t.Fatal(d, e)
	}
	delete(calendar, "2026-02-16")
	if _, e = requestDeadline("2026-02-13", 2, calendar); !errors.Is(e, ErrTrackingCalendar) {
		t.Fatal(e)
	}
}

func TestTrackingTransactionGuards(t *testing.T) {
	for _, tc := range []struct {
		name, kind                                         string
		lease, overlap, used                               int64
		stale, foreign, closed, missingCalendar, failWrite bool
		status                                             string
		want                                               error
	}{
		{name: "period insert", kind: "periodi", lease: 1, status: "PLANIRAN"},
		{name: "overlap", kind: "periodi", lease: 1, overlap: 1, status: "PLANIRAN", want: ErrTrackingInput},
		{name: "foreign org", kind: "periodi", lease: 1, foreign: true, status: "PLANIRAN", want: ErrUserOrganization},
		{name: "closed", kind: "periodi", lease: 1, closed: true, status: "PLANIRAN", want: ErrDocumentClosed},
		{name: "expired or foreign lease", kind: "periodi", status: "PLANIRAN", want: ErrMaintenanceLease},
		{name: "stale period", kind: "periodi", lease: 1, stale: true, status: "PLANIRAN", want: ErrMaintenanceConflict},
		{name: "request reserve", kind: "zahtevi", lease: 1, used: 80, status: "PODNET"},
		{name: "fund exhausted", kind: "zahtevi", lease: 1, used: 81, status: "PODNET", want: ErrTrackingFunds},
		{name: "draft does not reserve", kind: "zahtevi", lease: 1, used: 100, status: "NACRT"},
		{name: "cancel frees hours", kind: "zahtevi", lease: 1, used: 100, status: "OTKAZAN"},
		{name: "missing calendar", kind: "zahtevi", lease: 1, missingCalendar: true, status: "PODNET", want: ErrTrackingCalendar},
	} {
		t.Run(tc.name, func(t *testing.T) {
			writes, releases := 0, 0
			parentLocked, fundLocked := false, false
			store, tx := editStore(t, func(q string, a []driver.NamedValue) (driver.Rows, error) {
				switch {
				case strings.Contains(q, "FROM TED.UGO_EVID WHERE"):
					parentLocked = strings.Contains(q, "FOR UPDATE NOWAIT")
					org := int64(3)
					if tc.foreign {
						org = 4
					}
					return editRows(int64(1), int64(1), org, int64(101)), nil
				case strings.Contains(q, "FROM TED.SAP_UGOVORI WHERE"):
					if tc.closed {
						return editRows("-"), nil
					}
					return editRows("X"), nil
				case strings.Contains(q, "FROM TED.UGO_EDIT_LOCK"):
					if !strings.Contains(q, "lock_token = :token") || !strings.Contains(q, "expires_at > SYSTIMESTAMP") {
						return nil, fmt.Errorf("unguarded lease")
					}
					return editRows(tc.lease), nil
				case strings.Contains(q, "FROM TED.UGO_ODRZAVANJE o"):
					return editRows(int64(10), int64(1), "2026-01-01", "2026-12-31"), nil
				case strings.Contains(q, "FROM TED.UGO_TIP_DOC"):
					return editRows(int64(1)), nil
				case strings.Contains(q, "FROM TED.UGO_TEH_PERIOD"):
					return editRows(tc.overlap), nil
				case strings.Contains(q, "FROM TED.UGO_RAZ_FOND WHERE"):
					fundLocked = strings.Contains(q, "FOR UPDATE NOWAIT")
					return editRows(int64(20), int64(100)), nil
				case strings.Contains(q, "SUM(broj_casova)"):
					if !strings.Contains(q, "'NACRT','OTKAZAN'") {
						return nil, fmt.Errorf("wrong reservation statuses")
					}
					return editRows(tc.used), nil
				case strings.Contains(q, "FROM TED.UGO_RADNI_KALENDAR"):
					if tc.missingCalendar {
						return &schemaRows{width: 2}, nil
					}
					return editRows("2026-01-06", "D"), nil
				}
				return nil, fmt.Errorf("unexpected query %s", q)
			}, func(q string, a []driver.NamedValue) (driver.Result, error) {
				if !parentLocked {
					return nil, fmt.Errorf("parent not locked")
				}
				if strings.Contains(q, "DELETE FROM TED.UGO_EDIT_LOCK") {
					releases++
					return driver.RowsAffected(1), nil
				}
				if tc.kind == "zahtevi" && !fundLocked {
					return nil, fmt.Errorf("fund not locked")
				}
				writes++
				if tc.stale {
					return driver.RowsAffected(0), nil
				}
				return driver.RowsAffected(1), nil
			})
			p := models.TrackingItem{Start: "2026-01-01", End: "2026-01-31", DocumentType: 5, Status: tc.status, WorkType: 1, Number: "Z-1", Requested: "2026-01-05", Days: 1, Hours: 20}
			if tc.stale {
				p.ID = 2
				p.Version = 1
			}
			e := store.SaveTracking(context.Background(), 9, 3, 7, tc.kind, models.TrackingChange{Token: strings.Repeat("a", 64), Items: []models.TrackingItem{p}})
			if !errors.Is(e, tc.want) {
				t.Fatalf("got %v want %v", e, tc.want)
			}
			if e == nil {
				if !tx.committed || writes != 1 || releases != 1 {
					t.Fatal("incomplete commit")
				}
			} else if tx.committed || !tx.rolledBack || releases > 0 {
				t.Fatal("failed operation committed")
			}
		})
	}
}

func TestMaintenanceCannotReduceConsumedFund(t *testing.T) {
	store, txState := editStore(t, func(q string, a []driver.NamedValue) (driver.Rows, error) {
		if strings.Contains(q, "UGO_TEH_PERIOD") {
			return editRows(int64(0)), nil
		}
		return editRows(int64(1), int64(80), int64(1), int64(0)), nil
	}, nil)
	tx, e := store.DB.BeginTx(context.Background(), nil)
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback()
	e = validateTrackingSettings(context.Background(), tx, 10, "2026-01-01", "2026-12-31", models.Maintenance{Development: true, Funds: []models.DevelopmentFund{{TypeID: 1, Hours: 79}}})
	if !errors.Is(e, ErrTrackingDependencies) || txState.committed {
		t.Fatal(e)
	}
}

func TestGeneratedBatchRollsBackOnOverlap(t *testing.T) {
	checks, writes, releases := 0, 0, 0
	store, state := editStore(t, func(q string, a []driver.NamedValue) (driver.Rows, error) {
		switch {
		case strings.Contains(q, "FROM TED.UGO_EVID WHERE"):
			return editRows(int64(1), int64(1), int64(3), int64(101)), nil
		case strings.Contains(q, "FROM TED.SAP_UGOVORI WHERE"):
			return editRows("X"), nil
		case strings.Contains(q, "FROM TED.UGO_EDIT_LOCK"):
			return editRows(int64(1)), nil
		case strings.Contains(q, "FROM TED.UGO_ODRZAVANJE o"):
			return editRows(int64(10), int64(1), "2026-01-01", "2026-12-31"), nil
		case strings.Contains(q, "FROM TED.UGO_TIP_DOC"):
			return editRows(int64(1)), nil
		case strings.Contains(q, "FROM TED.UGO_TEH_PERIOD"):
			checks++
			if checks == 2 {
				return editRows(int64(1)), nil
			}
			return editRows(int64(0)), nil
		}
		return nil, fmt.Errorf("unexpected query %s", q)
	}, func(q string, a []driver.NamedValue) (driver.Result, error) {
		if strings.Contains(q, "DELETE") {
			releases++
		} else {
			writes++
		}
		return driver.RowsAffected(1), nil
	})
	items, _ := GeneratePeriods("2026-01-01", "2026-02-28", 1, 5, "calendar")
	err := store.SaveTracking(context.Background(), 9, 3, 7, "periodi", models.TrackingChange{Token: strings.Repeat("a", 64), Items: items})
	if !errors.Is(err, ErrTrackingInput) || writes != 1 || releases != 0 || state.committed || !state.rolledBack {
		t.Fatalf("batch was not rolled back: %v %+v", err, state)
	}
}

func TestRemindersAreOrganizationScoped(t *testing.T) {
	store := testSchemaStore(t, &schemaConn{query: func(q string, a []driver.NamedValue) (driver.Rows, error) {
		checkBinds(t, q, a)
		if len(a) != 2 || a[0].Value != 3 || a[1].Value != 3 || strings.Count(q, "e.id_ugo_org=") != 2 || !strings.Contains(q, "'NACRT','ZAVRSEN','OTKAZAN'") {
			return nil, fmt.Errorf("reminders are not scoped")
		}
		return editRows(int64(9), "RAZ", int64(1), "Z-1", "2026-01-10", int64(1)), nil
	}})
	items, e := store.ListReminders(context.Background(), 3)
	if e != nil || len(items) != 1 || !items[0].Overdue {
		t.Fatalf("%+v %v", items, e)
	}
}
