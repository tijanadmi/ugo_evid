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

func TestCalendarChanges(t *testing.T) {
	for _, tc := range []struct {
		name, action               string
		year                       int
		full, stale, busy, partial bool
		want                       error
	}{
		{name: "generate leap year preserves holiday", action: "generate", year: 2028},
		{name: "approve complete year", action: "approve", year: 2026, full: true},
		{name: "incomplete approval", action: "approve", year: 2026, want: ErrCalendarInput},
		{name: "stale approval", action: "approve", year: 2026, full: true, stale: true, want: ErrCalendarConflict},
		{name: "concurrent generator", action: "generate", year: 2026, busy: true, want: ErrCalendarConflict},
		{name: "edit days", action: "change", year: 2026, full: true},
		{name: "missing range rolls back", action: "change", year: 2026, partial: true, want: ErrCalendarInput},
	} {
		t.Run(tc.name, func(t *testing.T) {
			start, end, _ := calendarBounds(tc.year)
			values := [][]driver.Value{}
			for d := start; d.Before(end); d = d.AddDate(0, 0, 1) {
				values = append(values, []driver.Value{d.Format("2006-01-02"), "N", "D", "Praznik"})
				if !tc.full {
					break
				}
			}
			locked := false
			inserts, updates := 0, 0
			store, tx := editStore(t, func(q string, a []driver.NamedValue) (driver.Rows, error) {
				return &schemaRows{width: 4, values: values}, nil
			}, func(q string, a []driver.NamedValue) (driver.Result, error) {
				if strings.HasPrefix(q, "LOCK TABLE") {
					if tc.busy {
						return nil, errors.New("ORA-00054")
					}
					locked = true
					return driver.RowsAffected(0), nil
				}
				if !locked {
					return nil, fmt.Errorf("unlocked calendar write")
				}
				if strings.HasPrefix(q, "MERGE") {
					inserts++
					if strings.Contains(q, "WHEN MATCHED") || !strings.Contains(q, "WHEN NOT MATCHED THEN INSERT") || !strings.Contains(q, "TRUNC(s.datum,'IW') IN (5,6)") || a[1].Value != 366 {
						return nil, fmt.Errorf("unsafe calendar generation")
					}
					return driver.RowsAffected(365), nil
				}
				updates++
				if tc.action == "change" && !strings.Contains(q, "potvrdjen='N'") {
					return nil, fmt.Errorf("edit did not invalidate approval")
				}
				if tc.partial {
					return driver.RowsAffected(1), nil
				}
				return driver.RowsAffected(2), nil
			})
			snapshot, e := store.GetCalendar(context.Background(), tc.year)
			if e != nil {
				t.Fatal(e)
			}
			version := snapshot.Version
			if tc.stale {
				version = strings.Repeat("a", 64)
			}
			e = store.ChangeCalendar(context.Background(), tc.year, models.CalendarChange{Action: tc.action, Version: version, Start: "2026-01-01", End: "2026-01-02", Work: "N", Description: "Praznik"})
			if !errors.Is(e, tc.want) {
				t.Fatalf("got %v want %v", e, tc.want)
			}
			if e != nil {
				if tx.committed || !tx.rolledBack {
					t.Fatal("failure committed")
				}
				return
			}
			if !tx.committed {
				t.Fatal("missing commit")
			}
			if tc.action == "generate" && inserts != 1 {
				t.Fatalf("leap year inserts %d", inserts)
			}
			if tc.action != "generate" && updates != 1 {
				t.Fatal("unexpected writes")
			}
		})
	}
}
func TestCalendarValidation(t *testing.T) {
	store := &OracleStore{}
	for _, m := range []models.CalendarChange{
		{Action: "unknown"},
		{Action: "change", Version: strings.Repeat("a", 64), Start: "2025-12-31", End: "2026-01-02", Work: "N"},
		{Action: "change", Version: strings.Repeat("a", 64), Start: "2026-02-30", End: "2026-03-01", Work: "N"},
	} {
		if e := store.ChangeCalendar(context.Background(), 2026, m); !errors.Is(e, ErrCalendarInput) {
			t.Fatal(e)
		}
	}
}
