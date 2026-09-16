package repository

import (
	"context"
	"fmt"
	"time"

	"database/sql"
	"github.com/tijanadmi/ugo_evid/models"
)

func (r *OracleStore) GetUgoDobLicaRoleById(ctx context.Context, id int) (*models.UgoDobLicaRola, error) {
	query := `
        SELECT
id AS c0,
naziv AS c1,
status AS c2,
datpri AS c3,
datzm AS c4
 FROM TED.UGO_DOB_LICA_ROLE
        WHERE id = :p1
    `

	var m models.UgoDobLicaRola

	row := r.DB.QueryRowContext(ctx, query, sql.Named("p1", id))
	err := scanNullable(row,
		&m.ID,
		&m.Naziv,
		&m.Status,
		&m.DatPri,
		&m.DatZm,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &m, nil
}

func (r *OracleStore) GetUgoDobLicaRolePaged(ctx context.Context, offset, limit int, filter string) ([]*models.UgoDobLicaRola, int, error) {
	query := `
        SELECT
id AS c0,
naziv AS c1,
status AS c2,
datpri AS c3,
datzm AS c4,
COUNT(*) OVER() AS c5
 FROM TED.UGO_DOB_LICA_ROLE
        WHERE 1=1
    `

	args := []any{}
	pos := 1

	// FILTER po nazivu
	if filter != "" {
		query += fmt.Sprintf(" AND LOWER(naziv) LIKE LOWER('%%' || :p%d || '%%')", pos)
		args = append(args, filter)
		pos++
	}

	// PAGINACIJA
	query += fmt.Sprintf(" ORDER BY id DESC OFFSET :p%d ROWS FETCH NEXT :p%d ROWS ONLY", pos, pos+1)
	args = append(args, offset, limit)

	rows, err := r.DB.QueryContext(ctx, query, namedArgs(args)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []*models.UgoDobLicaRola
	total := 0

	for rows.Next() {
		var m models.UgoDobLicaRola
		var rowTotal int

		err := scanNullable(rows,
			&m.ID,
			&m.Naziv,
			&m.Status,
			&m.DatPri,
			&m.DatZm,
			&rowTotal,
		)
		if err != nil {
			return nil, 0, err
		}

		total = rowTotal
		list = append(list, &m)
	}

	return list, total, rows.Err()
}

func (r *OracleStore) InsertUgoDobLicaRole(ctx context.Context, m *models.UgoDobLicaRola) (*models.UgoDobLicaRola, error) {
	query := `
        INSERT INTO TED.UGO_DOB_LICA_ROLE
            (naziv, status, datpri, datzm)
        VALUES 
            (:p1, :p2, :p3, :p4)
        RETURNING id, datpri, datzm INTO :out0, :out1, :out2
    `

	now := time.Now()
	if m.DatPri.IsZero() {
		m.DatPri = now
	}
	if m.DatZm.IsZero() {
		m.DatZm = now
	}

	result, err := r.DB.ExecContext(ctx, query,
		sql.Named("p1", m.Naziv),
		sql.Named("p2", m.Status),
		sql.Named("p3", m.DatPri),
		sql.Named("p4", m.DatZm),
		sql.Named("out0", sql.Out{Dest: &m.ID}),
		sql.Named("out1", sql.Out{Dest: &m.DatPri}),
		sql.Named("out2", sql.Out{Dest: &m.DatZm}),
	)
	if err == nil {
		affected, rowsErr := result.RowsAffected()
		if rowsErr != nil {
			err = rowsErr
		} else if affected == 0 {
			err = sql.ErrNoRows
		}
	}

	if err != nil {
		return nil, err
	}

	return m, nil
}

func (r *OracleStore) UpdateUgoDobLicaRole(ctx context.Context, m *models.UgoDobLicaRola) (*models.UgoDobLicaRola, error) {
	query := `
        UPDATE TED.UGO_DOB_LICA_ROLE
        SET 
            naziv = :p1,
            status = :p2,
            datzm = :p3
        WHERE id = :p4
        RETURNING datpri, datzm INTO :out0, :out1
    `

	if m.DatZm.IsZero() {
		m.DatZm = time.Now()
	}

	var createdAt sql.NullTime
	result, err := r.DB.ExecContext(ctx, query,
		sql.Named("p1", m.Naziv),
		sql.Named("p2", m.Status),
		sql.Named("p3", m.DatZm),
		sql.Named("p4", m.ID),
		sql.Named("out0", sql.Out{Dest: &createdAt}),
		sql.Named("out1", sql.Out{Dest: &m.DatZm}),
	)
	if err == nil {
		affected, rowsErr := result.RowsAffected()
		if rowsErr != nil {
			err = rowsErr
		} else if affected == 0 {
			err = sql.ErrNoRows
		}
	}

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	m.DatPri = createdAt.Time
	return m, nil
}

func (r *OracleStore) DeleteUgoDobLicaRoleById(ctx context.Context, id int) error {
	query := `DELETE FROM TED.UGO_DOB_LICA_ROLE WHERE id = :p1`

	cmdTag, err := r.DB.ExecContext(ctx, query, sql.Named("p1", id))
	if err != nil {
		return err
	}

	affected, err := cmdTag.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return nil
	}

	return nil
}
