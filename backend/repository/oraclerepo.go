package repository

import (
	"database/sql"
	"fmt"
	"time"
)

type OracleStore struct{ DB *sql.DB }

func NewStore(db *sql.DB) Store { return &OracleStore{DB: db} }

func namedArgs(args []any) []any {
	result := make([]any, len(args))
	for i, arg := range args {
		result[i] = sql.Named(fmt.Sprintf("p%d", i+1), arg)
	}
	return result
}

func optionalID(id int) any {
	if id == 0 {
		return nil
	}
	return id
}

// Oracle treats empty strings as NULL. Preserve the existing JSON model's
// zero values for nullable columns, including missing LEFT JOIN records.
func scanNullable(row interface{ Scan(...any) error }, dest ...any) error {
	values := make([]any, len(dest))
	for i, d := range dest {
		switch d.(type) {
		case *string:
			values[i] = &sql.NullString{}
		case *int:
			values[i] = &sql.NullInt64{}
		case *float64:
			values[i] = &sql.NullFloat64{}
		case *time.Time:
			values[i] = &sql.NullTime{}
		default:
			values[i] = d
		}
	}
	if err := row.Scan(values...); err != nil {
		return err
	}
	for i, d := range dest {
		switch v := d.(type) {
		case *string:
			*v = values[i].(*sql.NullString).String
		case *int:
			*v = int(values[i].(*sql.NullInt64).Int64)
		case *float64:
			*v = values[i].(*sql.NullFloat64).Float64
		case *time.Time:
			*v = values[i].(*sql.NullTime).Time
		}
	}
	return nil
}
