package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/tijanadmi/ugo_evid/models"
)

func (r *OracleStore) GetOdgLicaForUgovor(ctx context.Context, ugovorID int) ([]models.SapOdglica, error) {

	query := `
		SELECT
id AS c0,
id_sap_ugovori AS c1,
odg_zap AS c2,
naziv_odg_zap AS c3,
r_br AS c4
 FROM sap_odglica
		WHERE id_sap_ugovori = :p1
		ORDER BY r_br
	`

	rows, err := r.DB.QueryContext(ctx, query, sql.Named("p1", ugovorID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []models.SapOdglica

	for rows.Next() {
		var o models.SapOdglica
		err := scanNullable(rows,
			&o.ID,
			&o.IdSapUgovori,
			&o.OdgZap,
			&o.NazivOdgZap,
			&o.RBr,
		)
		if err != nil {
			return nil, err
		}

		lista = append(lista, o)
	}

	return lista, rows.Err()
}

func (r *OracleStore) GetSapUgovoriPaged(
	ctx context.Context,
	offset, limit int,
	dobavljacNaziv string, // filtriranje po imenu
) ([]*models.SapUgovor, int, error) {

	query := `
        SELECT
u.id AS c0,
u.godina AS c1,
u.br_ugovor AS c2,
u.ugovor_dms AS c3,
u.jn AS c4,
u.br_poz_plana AS c5,
u.predmet_ugovora AS c6,
u.id_sap_dobavljac AS c7,
u.otvoren_ug AS c8,
u.zzn AS c9,
u.pocetak_ug AS c10,
u.kraj_ug AS c11,
u.vrednost_ug AS c12,
u.valuta_ug AS c13,
u.kurs_ug AS c14,
u.kom_grupa AS c15,
u.m_br_komerc AS c16,
u.naz_komerc AS c17,
u.gr_nab AS c18,
u.naziv_gr_plan AS c19,
u.vrs_pred AS c20,
d.id AS c21,
d.sifra AS c22,
d.naziv AS c23,
d.web_portal AS c24,
COUNT(*) OVER() AS c25
 FROM sap_ugovori u
        LEFT JOIN sap_dobavljaci d ON d.id = u.id_sap_dobavljac
        WHERE 1=1
    `

	args := []any{}
	pos := 1

	// ---------------------
	// FILTER: naziv dobavljaca (LIKE)
	// ---------------------
	if dobavljacNaziv != "" {
		query += fmt.Sprintf(" AND LOWER(d.naziv) LIKE LOWER('%%' || :p%d || '%%')", pos)
		args = append(args, dobavljacNaziv)
		pos++
	}

	// ---------------------
	// PAGINACIJA
	// ---------------------
	query += fmt.Sprintf(" ORDER BY u.id DESC OFFSET :p%d ROWS FETCH NEXT :p%d ROWS ONLY", pos, pos+1)
	args = append(args, offset, limit)

	rows, err := r.DB.QueryContext(ctx, query, namedArgs(args)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var ugovori []*models.SapUgovor
	var total int = 0

	for rows.Next() {
		var u models.SapUgovor
		var d models.SapDobavljac
		var rowTotal int

		err := scanNullable(rows,
			&u.ID,
			&u.Godina,
			&u.BrUgovor,
			&u.UgovorDMS,
			&u.JN,
			&u.BrPozPlana,
			&u.PredmetUgovora,
			&u.IdSapDobavljac,
			&u.OtvorenUG,
			&u.ZZN,
			&u.PocetakUG,
			&u.KrajUG,
			&u.VrednostUG,
			&u.ValutaUG,
			&u.KursUG,
			&u.KomGrupa,
			&u.MBrKomerc,
			&u.NazKomerc,
			&u.GrNab,
			&u.NazivGrPlan,
			&u.VrsPred,

			&d.ID,
			&d.Sifra,
			&d.Naziv,
			&d.WebPortal,

			&rowTotal,
		)
		if err != nil {
			return nil, 0, err
		}

		total = rowTotal
		u.Dobavljac = d

		ugovori = append(ugovori, &u)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	if err := rows.Close(); err != nil {
		return nil, 0, err
	}
	for _, u := range ugovori {
		odg, err := r.GetOdgLicaForUgovor(ctx, u.ID)
		if err != nil {
			return nil, 0, err
		}
		u.OdgLica = odg

	}
	return ugovori, total, nil
}
