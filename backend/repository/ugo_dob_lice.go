package repository

import (
	"context"
	"fmt"

	"database/sql"
	"github.com/tijanadmi/ugo_evid/models"
)

func (r *OracleStore) GetUgoDobLiceById(ctx context.Context, id int) (*models.UgoDobLice, error) {
	query := `
        SELECT
l.id AS c0,
l.ime AS c1,
l.radno_mesto AS c2,
l.telefon AS c3,
l.email AS c4,
l.status AS c5,
l.datpri AS contact_created,
l.datzm AS contact_updated,
s.id AS c6,
s.sifra AS c7,
s.naziv AS c8,
s.web_portal AS c9,
COALESCE(r.id, 0) AS c16,
COALESCE(r.naziv, '') AS c17,
COALESCE(r.status, '') AS c18
 FROM TED.UGO_DOB_LICA l
        JOIN TED.SAP_DOBAVLJACI s ON s.id = l.id_sap_dobavljac

        LEFT JOIN TED.UGO_DOB_LICA_ROLE r ON r.id = l.id_ugo_dob_lica_rola
        WHERE l.id = :p1
    `

	var m models.UgoDobLice

	row := r.DB.QueryRowContext(ctx, query, sql.Named("p1", id))
	err := scanNullable(row,
		&m.ID,
		&m.Ime,
		&m.RadnoMesto,
		&m.Telefon,
		&m.Email,
		&m.Status,
		&m.DatPri,
		&m.DatZm,

		&m.SapDobavljac.ID,
		&m.SapDobavljac.Sifra,
		&m.SapDobavljac.Naziv,
		&m.SapDobavljac.WebPortal,

		&m.UgoDobLicaRola.ID,
		&m.UgoDobLicaRola.Naziv,
		&m.UgoDobLicaRola.Status,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &m, nil
}

func (r *OracleStore) GetUgoDobLicePaged(ctx context.Context, offset, limit int, filter string) ([]*models.UgoDobLice, int, error) {
	query := `
        SELECT
l.id AS c0,
l.ime AS c1,
l.radno_mesto AS c2,
l.telefon AS c3,
l.email AS c4,
l.status AS c5,
l.datpri AS contact_created,
l.datzm AS contact_updated,
s.id AS c6,
s.sifra AS c7,
s.naziv AS c8,
s.web_portal AS c9,
COALESCE(r2.id, 0) AS c16,
COALESCE(r2.naziv, '') AS c17,
COALESCE(r2.status, '') AS c18,
COUNT(*) OVER() AS c19
 FROM TED.UGO_DOB_LICA l
        JOIN TED.SAP_DOBAVLJACI s ON s.id = l.id_sap_dobavljac

        LEFT JOIN TED.UGO_DOB_LICA_ROLE r2 ON r2.id = l.id_ugo_dob_lica_rola
        WHERE 1=1
    `

	args := []any{}
	pos := 1

	// FILTER: ime, email, radno mesto, telefon, dobavljač, rola
	if filter != "" {
		query += fmt.Sprintf(`
            AND (
                LOWER(l.ime) LIKE LOWER('%%' || :p%d || '%%')
                OR LOWER(l.email) LIKE LOWER('%%' || :p%d || '%%')
                OR LOWER(l.radno_mesto) LIKE LOWER('%%' || :p%d || '%%')
                OR LOWER(l.telefon) LIKE LOWER('%%' || :p%d || '%%')
                OR LOWER(s.naziv) LIKE LOWER('%%' || :p%d || '%%')

                OR LOWER(r2.naziv) LIKE LOWER('%%' || :p%d || '%%')
            )
        `, pos, pos, pos, pos, pos, pos)

		args = append(args, filter)
		pos++
	}

	// PAGINACIJA
	query += fmt.Sprintf(" ORDER BY l.id DESC OFFSET :p%d ROWS FETCH NEXT :p%d ROWS ONLY", pos, pos+1)
	args = append(args, offset, limit)

	rows, err := r.DB.QueryContext(ctx, query, namedArgs(args)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []*models.UgoDobLice
	total := 0

	for rows.Next() {
		var m models.UgoDobLice
		var rowTotal int

		err := scanNullable(rows,
			&m.ID,
			&m.Ime,
			&m.RadnoMesto,
			&m.Telefon,
			&m.Email,
			&m.Status,
			&m.DatPri,
			&m.DatZm,

			&m.SapDobavljac.ID,
			&m.SapDobavljac.Sifra,
			&m.SapDobavljac.Naziv,
			&m.SapDobavljac.WebPortal,

			&m.UgoDobLicaRola.ID,
			&m.UgoDobLicaRola.Naziv,
			&m.UgoDobLicaRola.Status,

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

func (r *OracleStore) InsertUgoDobLice(ctx context.Context, m *models.UgoDobLice) (*models.UgoDobLice, error) {
	query := `
        INSERT INTO TED.UGO_DOB_LICA
            (id_sap_dobavljac, ime, radno_mesto, telefon, email, id_ugo_dob_lica_rola, status, datpri, datzm)
        VALUES 
            (:p1, :p2, :p3, :p4, :p5, :p6, :p7, SYSDATE, SYSDATE)
        RETURNING id INTO :out0
    `

	result, err := r.DB.ExecContext(ctx, query,
		sql.Named("p1", m.SapDobavljac.ID),

		sql.Named("p2", m.Ime),
		sql.Named("p3", m.RadnoMesto),
		sql.Named("p4", m.Telefon),
		sql.Named("p5", m.Email),
		sql.Named("p6", optionalID(m.UgoDobLicaRola.ID)),
		sql.Named("p7", m.Status),
		sql.Named("out0", sql.Out{Dest: &m.ID}),
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

func (r *OracleStore) UpdateUgoDobLice(ctx context.Context, m *models.UgoDobLice) (*models.UgoDobLice, error) {
	query := `
        UPDATE TED.UGO_DOB_LICA
        SET 
            id_sap_dobavljac = :p1,

            ime = :p2,
            radno_mesto = :p3,
            telefon = :p4,
            email = :p5,
            id_ugo_dob_lica_rola = :p6,
            status = :p7,
            datzm = SYSDATE
        WHERE id = :p8
        RETURNING id INTO :out0
    `

	var id int
	result, err := r.DB.ExecContext(ctx, query,
		sql.Named("p1", m.SapDobavljac.ID),

		sql.Named("p2", m.Ime),
		sql.Named("p3", m.RadnoMesto),
		sql.Named("p4", m.Telefon),
		sql.Named("p5", m.Email),
		sql.Named("p6", optionalID(m.UgoDobLicaRola.ID)),
		sql.Named("p7", m.Status),
		sql.Named("p8", m.ID),
		sql.Named("out0", sql.Out{Dest: &id}),
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

	return m, nil
}

func (r *OracleStore) DeleteUgoDobLiceById(ctx context.Context, id int) error {
	query := `DELETE FROM TED.UGO_DOB_LICA WHERE id = :p1`

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
