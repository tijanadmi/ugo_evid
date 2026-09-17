package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/tijanadmi/ugo_evid/models"
)

// GetUgoEvidProsireniPaged reads the existing view; it never modifies SAP or UGO data.
func (r *OracleStore) GetUgoEvidProsireniPaged(ctx context.Context, open bool, offset, limit, orgID int) ([]models.UgoEvidProsireni, int, error) {
	if offset < 0 || limit < 1 || limit > 100 {
		return nil, 0, fmt.Errorf("invalid pagination")
	}
	if orgID < 0 {
		return nil, 0, fmt.Errorf("invalid organization ID")
	}
	predicate := "v.otvoren_ug = 'X'"
	if !open {
		predicate = "v.otvoren_ug IS NULL"
	}
	from := " FROM TED.UGO_EVID_PROSIRENI_V v WHERE " + predicate
	args := []any{}
	if orgID > 0 {
		from += " AND v.id_ugo_org = :org_id"
		args = append(args, sql.Named("org_id", orgID))
	}
	// Separate count also returns the correct total for an empty/out-of-range page.
	var total int
	if err := r.DB.QueryRowContext(ctx, "SELECT COUNT(*)"+from, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, sql.Named("page_offset", offset), sql.Named("page_limit", limit))
	rows, err := r.DB.QueryContext(ctx, `SELECT
 v.id_ugo_evid, v.id_ugo_org, v.id_sap_ugovor, v.id_sap_dobavljac,
 v.godina, v.br_ugovor, v.ugovor_dms, v.jn, v.br_poz_plana,
 v.predmet_ugovora, v.naziv, v.otvoren_ug, v.zzn,
 v.pocetak_ug, v.kraj_ug, v.vrednost_ug, v.valuta_ug, v.kurs_ug,
 v.kom_grupa, v.m_br_komerc, v.naz_komerc, v.naziv_gr_plan, v.vrs_pred, v.sluzba,
 v.odg_zap_1, v.naziv_odg_zap_1, v.odg_zap_2, v.naziv_odg_zap_2,
 v.odg_zap_3, v.naziv_odg_zap_3, v.odg_zap_4, v.naziv_odg_zap_4,
 v.odg_zap_5, v.naziv_odg_zap_5, v.odg_zap_6, v.naziv_odg_zap_6,
 v.ime, v.telefon, v.email, v.status, v.datpri, v.datizm`+from+
		" ORDER BY v.id_ugo_evid DESC OFFSET :page_offset ROWS FETCH NEXT :page_limit ROWS ONLY",
		args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]models.UgoEvidProsireni, 0)
	for rows.Next() {
		var m models.UgoEvidProsireni
		if err = scanNullable(rows,
			&m.IDUgoEvid, &m.IDUgoOrg, &m.IDSapUgovor, &m.IDSapDobavljac,
			&m.Godina, &m.BrUgovor, &m.UgovorDMS, &m.JN, &m.BrPozPlana,
			&m.PredmetUgovora, &m.Naziv, &m.OtvorenUG, &m.ZZN,
			&m.PocetakUG, &m.KrajUG, &m.VrednostUG, &m.ValutaUG, &m.KursUG,
			&m.KomGrupa, &m.MBrKomerc, &m.NazKomerc, &m.NazivGrPlan, &m.VrsPred, &m.Sluzba,
			&m.OdgZap1, &m.NazivOdgZap1, &m.OdgZap2, &m.NazivOdgZap2,
			&m.OdgZap3, &m.NazivOdgZap3, &m.OdgZap4, &m.NazivOdgZap4,
			&m.OdgZap5, &m.NazivOdgZap5, &m.OdgZap6, &m.NazivOdgZap6,
			&m.Ime, &m.Telefon, &m.Email, &m.Status, &m.DatPri, &m.DatIzm); err != nil {
			return nil, 0, err
		}
		items = append(items, m)
	}
	return items, total, rows.Err()
}
