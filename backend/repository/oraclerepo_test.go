package repository

import (
	"database/sql"
	"errors"
	"testing"
	"time"
)

type testRow struct {
	values []any
	err    error
}

func (r testRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	for i, d := range dest {
		if err := d.(sql.Scanner).Scan(r.values[i]); err != nil {
			return err
		}
	}
	return nil
}
func TestScanOracleNullableValues(t *testing.T) {
	var s string
	var i int
	var f float64
	var date time.Time
	if err := scanNullable(testRow{values: []any{nil, nil, nil, nil}}, &s, &i, &f, &date); err != nil {
		t.Fatal(err)
	}
	if s != "" || i != 0 || f != 0 || !date.IsZero() {
		t.Fatal("NULL mapping failed")
	}
	now := time.Now()
	if err := scanNullable(testRow{values: []any{[]byte("tekst"), int64(42), "12.5", now}}, &s, &i, &f, &date); err != nil {
		t.Fatal(err)
	}
	if s != "tekst" || i != 42 || f != 12.5 || !date.Equal(now) {
		t.Fatal("value conversion failed")
	}
	if err := scanNullable(testRow{err: sql.ErrNoRows}, &s); !errors.Is(err, sql.ErrNoRows) {
		t.Fatal("scan error lost")
	}
}
