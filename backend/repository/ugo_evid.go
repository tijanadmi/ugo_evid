package repository

import (
	"context"
	"fmt"
	"time"

	"database/sql"
	"github.com/tijanadmi/ugo_evid/models"
)

func (r *OracleStore) GetUgoEvidById(ctx context.Context, id int) (*models.UgoEvid, error) {

	query := `
        SELECT
e.id AS c0,
e.ime AS c1,
e.telefon AS c2,
e.email AS c3,
e.status AS c4,
e.datpri AS c5,
e.datizm AS c6,
su.id AS c7,
su.godina AS c8,
su.br_ugovor AS c9,
su.jn AS c10,
su.predmet_ugovora AS c11,
o.id AS c12,
o.sifra AS c13,
o.sifra_cir AS c14,
o.naziv AS c15,
o.naziv_cir AS c16,
o.status AS c17,
o.datpri AS c18,
o.datizm AS c19,
dl.id AS c20,
dl.ime AS c21,
dl.radno_mesto AS c22,
dl.telefon AS c23,
dl.email AS c24,
dl.status AS c25
 FROM TED.UGO_EVID e
        JOIN TED.SAP_UGOVORI su ON su.id = e.id_sap_ugovor
        JOIN TED.UGO_ORG o      ON o.id = e.id_ugo_org
        LEFT JOIN TED.UGO_DOB_LICA dl ON dl.id = e.id_ugo_dob_lica
        WHERE e.id = :p1
    `

	var e models.UgoEvid

	row := r.DB.QueryRowContext(ctx, query, sql.Named("p1", id))
	err := scanNullable(row,
		&e.ID,
		&e.Ime,
		&e.Telefon,
		&e.Email,
		&e.Status,
		&e.DatPri,
		&e.DatIzm,

		&e.SapUgovor.ID,
		&e.SapUgovor.Godina,
		&e.SapUgovor.BrUgovor,
		&e.SapUgovor.JN,
		&e.SapUgovor.PredmetUgovora,

		&e.UgoOrg.ID,
		&e.UgoOrg.Sifra,
		&e.UgoOrg.SifraCir,
		&e.UgoOrg.Naziv,
		&e.UgoOrg.NazivCir,
		&e.UgoOrg.Status,
		&e.UgoOrg.DatPri,
		&e.UgoOrg.DatIzm,

		&e.UgoDobLice.ID,
		&e.UgoDobLice.Ime,
		&e.UgoDobLice.RadnoMesto,
		&e.UgoDobLice.Telefon,
		&e.UgoDobLice.Email,
		&e.UgoDobLice.Status,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &e, nil
}

func (r *OracleStore) GetUgoEvidPaged(ctx context.Context, offset, limit int, filter string) ([]*models.UgoEvid, int, error) {

	query := `
        SELECT
e.id AS c0,
e.ime AS c1,
e.telefon AS c2,
e.email AS c3,
e.status AS c4,
e.datpri AS c5,
e.datizm AS c6,
su.id AS c7,
su.godina AS c8,
su.br_ugovor AS c9,
su.jn AS c10,
su.predmet_ugovora AS c11,
o.id AS c12,
o.sifra AS c13,
o.sifra_cir AS c14,
o.naziv AS c15,
o.naziv_cir AS c16,
o.status AS c17,
o.datpri AS c18,
o.datizm AS c19,
dl.id AS c20,
dl.ime AS c21,
dl.radno_mesto AS c22,
dl.telefon AS c23,
dl.email AS c24,
dl.status AS c25,
COUNT(*) OVER() AS c26
 FROM TED.UGO_EVID e
        JOIN TED.SAP_UGOVORI su ON su.id = e.id_sap_ugovor
        JOIN TED.UGO_ORG o      ON o.id = e.id_ugo_org
        LEFT JOIN TED.UGO_DOB_LICA dl ON dl.id = e.id_ugo_dob_lica
        WHERE 1=1
    `

	args := []any{}
	pos := 1

	if filter != "" {
		query += fmt.Sprintf(`
            AND (
                LOWER(e.ime) LIKE LOWER('%%' || :p%d || '%%')
                OR LOWER(e.email) LIKE LOWER('%%' || :p%d || '%%')
                OR LOWER(e.telefon) LIKE LOWER('%%' || :p%d || '%%')
            )
        `, pos, pos, pos)
		args = append(args, filter)
		pos++
	}

	query += fmt.Sprintf(" ORDER BY e.id DESC OFFSET :p%d ROWS FETCH NEXT :p%d ROWS ONLY", pos, pos+1)
	args = append(args, offset, limit)

	rows, err := r.DB.QueryContext(ctx, query, namedArgs(args)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []*models.UgoEvid
	total := 0

	for rows.Next() {
		var e models.UgoEvid
		var rowTotal int

		err := scanNullable(rows,
			&e.ID,
			&e.Ime,
			&e.Telefon,
			&e.Email,
			&e.Status,
			&e.DatPri,
			&e.DatIzm,

			&e.SapUgovor.ID,
			&e.SapUgovor.Godina,
			&e.SapUgovor.BrUgovor,
			&e.SapUgovor.JN,
			&e.SapUgovor.PredmetUgovora,

			&e.UgoOrg.ID,
			&e.UgoOrg.Sifra,
			&e.UgoOrg.SifraCir,
			&e.UgoOrg.Naziv,
			&e.UgoOrg.NazivCir,
			&e.UgoOrg.Status,
			&e.UgoOrg.DatPri,
			&e.UgoOrg.DatIzm,

			&e.UgoDobLice.ID,
			&e.UgoDobLice.Ime,
			&e.UgoDobLice.RadnoMesto,
			&e.UgoDobLice.Telefon,
			&e.UgoDobLice.Email,
			&e.UgoDobLice.Status,

			&rowTotal,
		)
		if err != nil {
			return nil, 0, err
		}

		total = rowTotal
		list = append(list, &e)
	}

	return list, total, rows.Err()
}

func (r *OracleStore) InsertUgoEvid(ctx context.Context, e *models.UgoEvid) (*models.UgoEvid, error) {

	query := `
        INSERT INTO TED.UGO_EVID
            (id_sap_ugovor, id_ugo_org, ime, telefon, email, status, datpri, datizm, id_ugo_dob_lica)
        VALUES
            (:p1, :p2, :p3, :p4, :p5, :p6, :p7, :p8, :p9)
        RETURNING id, datpri, datizm INTO :out0, :out1, :out2
    `

	now := time.Now()
	if e.DatPri.IsZero() {
		e.DatPri = now
	}
	if e.DatIzm.IsZero() {
		e.DatIzm = now
	}

	result, err := r.DB.ExecContext(ctx, query,
		sql.Named("p1", e.SapUgovor.ID),
		sql.Named("p2", e.UgoOrg.ID),
		sql.Named("p3", e.Ime),
		sql.Named("p4", e.Telefon),
		sql.Named("p5", e.Email),
		sql.Named("p6", e.Status),
		sql.Named("p7", e.DatPri),
		sql.Named("p8", e.DatIzm),
		sql.Named("p9", optionalID(e.UgoDobLice.ID)),
		sql.Named("out0", sql.Out{Dest: &e.ID}),
		sql.Named("out1", sql.Out{Dest: &e.DatPri}),
		sql.Named("out2", sql.Out{Dest: &e.DatIzm}),
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

	return e, nil
}

func (r *OracleStore) UpdateUgoEvid(ctx context.Context, e *models.UgoEvid) (*models.UgoEvid, error) {

	query := `
        UPDATE TED.UGO_EVID
        SET
            id_sap_ugovor = :p1,
            id_ugo_org = :p2,
            ime = :p3,
            telefon = :p4,
            email = :p5,
            status = :p6,
            datizm = :p7,
            id_ugo_dob_lica = :p8
        WHERE id = :p9
        RETURNING datpri, datizm INTO :out0, :out1
    `

	if e.DatIzm.IsZero() {
		e.DatIzm = time.Now()
	}

	var createdAt sql.NullTime
	result, err := r.DB.ExecContext(ctx, query,
		sql.Named("p1", e.SapUgovor.ID),
		sql.Named("p2", e.UgoOrg.ID),
		sql.Named("p3", e.Ime),
		sql.Named("p4", e.Telefon),
		sql.Named("p5", e.Email),
		sql.Named("p6", e.Status),
		sql.Named("p7", e.DatIzm),
		sql.Named("p8", optionalID(e.UgoDobLice.ID)),
		sql.Named("p9", e.ID),
		sql.Named("out0", sql.Out{Dest: &createdAt}),
		sql.Named("out1", sql.Out{Dest: &e.DatIzm}),
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

	e.DatPri = createdAt.Time
	return e, nil
}

func (r *OracleStore) DeleteUgoEvidById(ctx context.Context, id int) error {
	query := `DELETE FROM TED.UGO_EVID WHERE id = :p1`
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
