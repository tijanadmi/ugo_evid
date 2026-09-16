package repository

import (
	"context"
	"fmt"
	"time"

	"database/sql"
	"github.com/tijanadmi/ugo_evid/models"
)

func (r *OracleStore) GetUgoOrgById(ctx context.Context, id int) (*models.UgoOrg, error) {
	query := `
        SELECT
id AS c0,
sifra AS c1,
sifra_cir AS c2,
naziv AS c3,
naziv_cir AS c4,
status AS c5,
datpri AS c6,
datizm AS c7
 FROM TED.UGO_ORG
        WHERE id = :p1
    `

	var org models.UgoOrg
	row := r.DB.QueryRowContext(ctx, query, sql.Named("p1", id))
	err := scanNullable(row,
		&org.ID,
		&org.Sifra,
		&org.SifraCir,
		&org.Naziv,
		&org.NazivCir,
		&org.Status,
		&org.DatPri,
		&org.DatIzm,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &org, nil
}

func (r *OracleStore) GetUgoOrgPaged(ctx context.Context, offset, limit int, filter string) ([]*models.UgoOrg, int, error) {

	query := `
        SELECT
id AS c0,
sifra AS c1,
sifra_cir AS c2,
naziv AS c3,
naziv_cir AS c4,
status AS c5,
datpri AS c6,
datizm AS c7,
COUNT(*) OVER() AS c8
 FROM TED.UGO_ORG
        WHERE 1=1
    `

	args := []any{}
	pos := 1

	// FILTER: po sifra ili nazivu
	if filter != "" {
		query += fmt.Sprintf(" AND (LOWER(sifra) LIKE LOWER('%%' || :p%d || '%%') OR LOWER(naziv) LIKE LOWER('%%' || :p%d || '%%'))", pos, pos)
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

	var orgs []*models.UgoOrg
	total := 0

	for rows.Next() {
		var org models.UgoOrg
		var rowTotal int

		err := scanNullable(rows,
			&org.ID,
			&org.Sifra,
			&org.SifraCir,
			&org.Naziv,
			&org.NazivCir,
			&org.Status,
			&org.DatPri,
			&org.DatIzm,
			&rowTotal,
		)
		if err != nil {
			return nil, 0, err
		}

		total = rowTotal
		orgs = append(orgs, &org)
	}

	return orgs, total, rows.Err()
}

func (r *OracleStore) InsertUgoOrg(ctx context.Context, org *models.UgoOrg) (*models.UgoOrg, error) {
	query := `
        INSERT INTO TED.UGO_ORG
            (sifra, sifra_cir, naziv, naziv_cir, status, datpri, datizm)
        VALUES
            (:p1, :p2, :p3, :p4, :p5, :p6, :p7)
        RETURNING id, datpri, datizm INTO :out0, :out1, :out2
    `

	now := time.Now()
	if org.DatPri.IsZero() {
		org.DatPri = now
	}
	if org.DatIzm.IsZero() {
		org.DatIzm = now
	}

	result, err := r.DB.ExecContext(ctx, query,
		sql.Named("p1", org.Sifra),
		sql.Named("p2", org.SifraCir),
		sql.Named("p3", org.Naziv),
		sql.Named("p4", org.NazivCir),
		sql.Named("p5", org.Status),
		sql.Named("p6", org.DatPri),
		sql.Named("p7", org.DatIzm),
		sql.Named("out0", sql.Out{Dest: &org.ID}),
		sql.Named("out1", sql.Out{Dest: &org.DatPri}),
		sql.Named("out2", sql.Out{Dest: &org.DatIzm}),
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

	return org, nil
}

func (r *OracleStore) UpdateUgoOrg(ctx context.Context, org *models.UgoOrg) (*models.UgoOrg, error) {
	query := `
        UPDATE TED.UGO_ORG
        SET 
            sifra_cir = :p1,
            naziv = :p2,
            naziv_cir = :p3,
            status = :p4,
            datizm = :p5
        WHERE sifra = :p6
        RETURNING id, datpri, datizm INTO :out0, :out1, :out2
    `

	if org.DatIzm.IsZero() {
		org.DatIzm = time.Now()
	}

	var createdAt sql.NullTime
	result, err := r.DB.ExecContext(ctx, query,
		sql.Named("p1", org.SifraCir),
		sql.Named("p2", org.Naziv),
		sql.Named("p3", org.NazivCir),
		sql.Named("p4", org.Status),
		sql.Named("p5", org.DatIzm),
		sql.Named("p6", org.Sifra),
		sql.Named("out0", sql.Out{Dest: &org.ID}),
		sql.Named("out1", sql.Out{Dest: &createdAt}),
		sql.Named("out2", sql.Out{Dest: &org.DatIzm}),
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

	org.DatPri = createdAt.Time
	return org, nil
}

func (r *OracleStore) DeleteUgoOrgById(ctx context.Context, id int) error {
	query := `DELETE FROM TED.UGO_ORG WHERE id = :p1`
	cmdTag, err := r.DB.ExecContext(ctx, query, sql.Named("p1", id))
	if err != nil {
		return err
	}

	affected, err := cmdTag.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return nil // ili sql.ErrNoRows ako želiš signalizaciju da ne postoji
	}

	return nil
}
