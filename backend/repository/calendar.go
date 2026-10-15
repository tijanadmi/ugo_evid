package repository

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/tijanadmi/ugo_evid/models"
	"strings"
	"time"
	"unicode/utf8"
)

var ErrCalendarConflict = errors.New("Kalendar je izmenjen u drugoj sesiji. Osvezite prikaz pre nastavka.")
var ErrCalendarInput = errors.New("Proverite godinu, opseg datuma i radni status. Pre potvrde generisite celu godinu.")

type calendarReader interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func calendarBounds(year int) (time.Time, time.Time, error) {
	if year < 1900 || year > 2200 {
		return time.Time{}, time.Time{}, ErrCalendarInput
	}
	start := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	return start, start.AddDate(1, 0, 0), nil
}
func readCalendar(ctx context.Context, db calendarReader, year int) (*models.CalendarYear, error) {
	start, end, e := calendarBounds(year)
	if e != nil {
		return nil, e
	}
	rows, e := db.QueryContext(ctx, `SELECT TO_CHAR(datum,'YYYY-MM-DD'),radni,potvrdjen,opis FROM TED.UGO_RADNI_KALENDAR WHERE datum>=TO_DATE(:start_date,'YYYY-MM-DD') AND datum<TO_DATE(:end_date,'YYYY-MM-DD') ORDER BY datum`, sql.Named("start_date", start.Format("2006-01-02")), sql.Named("end_date", end.Format("2006-01-02")))
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	result := &models.CalendarYear{Items: []models.CalendarDay{}}
	for rows.Next() {
		var day models.CalendarDay
		if e = scanNullable(rows, &day.Date, &day.Work, &day.Approved, &day.Description); e != nil {
			return nil, e
		}
		result.Items = append(result.Items, day)
	}
	if e = rows.Err(); e != nil {
		return nil, e
	}
	raw, _ := json.Marshal(result.Items)
	hash := sha256.Sum256(raw)
	result.Version = hex.EncodeToString(hash[:])
	return result, nil
}
func (r *OracleStore) GetCalendar(ctx context.Context, year int) (*models.CalendarYear, error) {
	return readCalendar(ctx, r.DB, year)
}
func (r *OracleStore) ChangeCalendar(ctx context.Context, year int, m models.CalendarChange) error {
	start, end, e := calendarBounds(year)
	if e != nil {
		return e
	}
	if m.Action != "generate" && m.Action != "change" && m.Action != "approve" {
		return ErrCalendarInput
	}
	if len(m.Version) != 64 || utf8.RuneCountInString(m.Description) > 200 {
		return ErrCalendarInput
	}
	if m.Action == "change" {
		a, ae := date(m.Start)
		b, be := date(m.End)
		if ae != nil || be != nil || a.Before(start) || !b.Before(end) || a.After(b) || (m.Work != "D" && m.Work != "N") {
			return ErrCalendarInput
		}
	}
	tx, e := r.DB.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	// Short database lock also covers first generation, when no day rows exist.
	if _, e = tx.ExecContext(ctx, `LOCK TABLE TED.UGO_RADNI_KALENDAR IN EXCLUSIVE MODE NOWAIT`); e != nil {
		if strings.Contains(e.Error(), "ORA-00054") {
			return ErrCalendarConflict
		}
		return e
	}
	current, e := readCalendar(ctx, tx, year)
	if e != nil {
		return e
	}
	if current.Version != m.Version {
		return ErrCalendarConflict
	}
	switch m.Action {
	case "generate":
		_, e = tx.ExecContext(ctx, `MERGE INTO TED.UGO_RADNI_KALENDAR k USING (SELECT TO_DATE(:start_date,'YYYY-MM-DD')+LEVEL-1 datum FROM dual CONNECT BY LEVEL<=:days) s ON (k.datum=s.datum) WHEN NOT MATCHED THEN INSERT(datum,radni,potvrdjen) VALUES(s.datum,CASE WHEN s.datum-TRUNC(s.datum,'IW') IN (5,6) THEN 'N' ELSE 'D' END,'N')`, sql.Named("start_date", start.Format("2006-01-02")), sql.Named("days", int(end.Sub(start).Hours()/24)))
		if e != nil {
			return e
		}
	case "change":
		a, _ := date(m.Start)
		b, _ := date(m.End)
		result, err := tx.ExecContext(ctx, `UPDATE TED.UGO_RADNI_KALENDAR SET radni=:work,opis=:description,potvrdjen='N' WHERE datum>=TO_DATE(:start_date,'YYYY-MM-DD') AND datum<=TO_DATE(:end_date,'YYYY-MM-DD')`, sql.Named("work", m.Work), sql.Named("description", m.Description), sql.Named("start_date", m.Start), sql.Named("end_date", m.End))
		if err != nil {
			return err
		}
		n, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if n != int64(b.Sub(a).Hours()/24)+1 {
			return ErrCalendarInput
		}
	case "approve":
		if len(current.Items) != int(end.Sub(start).Hours()/24) {
			return ErrCalendarInput
		}
		_, e = tx.ExecContext(ctx, `UPDATE TED.UGO_RADNI_KALENDAR SET potvrdjen='D' WHERE datum>=TO_DATE(:start_date,'YYYY-MM-DD') AND datum<TO_DATE(:end_date,'YYYY-MM-DD')`, sql.Named("start_date", start.Format("2006-01-02")), sql.Named("end_date", end.Format("2006-01-02")))
		if e != nil {
			return e
		}
	}
	return tx.Commit()
}
