package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/tijanadmi/ugo_evid/models"
)

func (r *OracleStore) GetUgoEvidProsireniByID(ctx context.Context, id int) (models.UgoEvidProsireni, error) {
	var item models.UgoEvidProsireni
	if id < 1 {
		return item, fmt.Errorf("invalid evidence ID")
	}
	err := scanProsireni(r.DB.QueryRowContext(ctx, prosireniSelect+" FROM TED.UGO_EVID_PROSIRENI_V v WHERE v.id_ugo_evid = :id", sql.Named("id", id)), &item)
	if err != nil {
		return item, err
	}
	items := []models.UgoEvidProsireni{item}
	if err := r.loadProsireniContacts(ctx, items); err != nil {
		return item, err
	}
	return items[0], nil
}

// Fetch all supplier contacts once, after closing the paged view cursor.
// This avoids multiplying contract rows or holding a connection for nested queries.
func (r *OracleStore) loadProsireniContacts(ctx context.Context, items []models.UgoEvidProsireni) error {
	bySupplier := make(map[int][]models.UgoDobLiceKontakt)
	args := []any{}
	binds := []string{}
	for i := range items {
		items[i].LicaDobavljaca = []models.UgoDobLiceKontakt{}
		if items[i].IDSapDobavljac == nil {
			continue
		}
		id := *items[i].IDSapDobavljac
		if _, exists := bySupplier[id]; exists {
			continue
		}
		bySupplier[id] = []models.UgoDobLiceKontakt{}
		name := fmt.Sprintf("supplier_%d", len(args))
		binds = append(binds, ":"+name)
		args = append(args, sql.Named(name, id))
	}
	if len(args) == 0 {
		return nil
	}
	rows, err := r.DB.QueryContext(ctx, `SELECT l.id_sap_dobavljac, l.ime, l.radno_mesto, l.telefon, l.email, r.naziv, l.id
 FROM TED.UGO_DOB_LICA l
 LEFT JOIN TED.UGO_DOB_LICA_ROLE r ON l.id_ugo_dob_lica_rola = r.id
 WHERE l.id_sap_dobavljac IN (`+strings.Join(binds, ",")+`)
 ORDER BY r.id, l.id`, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		var contact models.UgoDobLiceKontakt
		if err := scanNullable(rows, &id, &contact.Ime, &contact.RadnoMesto, &contact.Telefon, &contact.Email, &contact.RolaLica, &contact.ID); err != nil {
			return err
		}
		bySupplier[id] = append(bySupplier[id], contact)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for i := range items {
		if items[i].IDSapDobavljac != nil {
			items[i].LicaDobavljaca = bySupplier[*items[i].IDSapDobavljac]
		}
	}
	return nil
}

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
	rows, err := r.DB.QueryContext(ctx, prosireniSelect+from+
		" ORDER BY v.id_ugo_evid DESC OFFSET :page_offset ROWS FETCH NEXT :page_limit ROWS ONLY",
		args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]models.UgoEvidProsireni, 0)
	for rows.Next() {
		var m models.UgoEvidProsireni
		if err = scanProsireni(rows, &m); err != nil {
			return nil, 0, err
		}
		items = append(items, m)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	if err := rows.Close(); err != nil {
		return nil, 0, err
	}
	if err := r.loadProsireniContacts(ctx, items); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

const prosireniSelect = `SELECT
 v.id_ugo_evid, v.id_ugo_org, v.id_sap_ugovor, v.id_sap_dobavljac,
 v.godina, v.br_ugovor, v.ugovor_dms, v.jn, v.br_poz_plana,
 v.predmet_ugovora, v.naziv, v.otvoren_ug, v.zzn,
 v.pocetak_ug, v.kraj_ug, v.vrednost_ug, v.valuta_ug, v.kurs_ug,
 v.kom_grupa, v.m_br_komerc, v.naz_komerc, v.naziv_gr_plan, v.vrs_pred, v.sluzba,
 v.odg_zap_1, v.naziv_odg_zap_1, v.odg_zap_2, v.naziv_odg_zap_2,
 v.odg_zap_3, v.naziv_odg_zap_3, v.odg_zap_4, v.naziv_odg_zap_4,
 v.odg_zap_5, v.naziv_odg_zap_5, v.odg_zap_6, v.naziv_odg_zap_6,
 v.ime, v.telefon, v.email, v.status, v.datpri, v.datizm,
 (SELECT e.id_ugo_dob_lica FROM TED.UGO_EVID e WHERE e.id = v.id_ugo_evid) AS id_ugo_dob_lica`

func scanProsireni(rows interface{ Scan(...any) error }, m *models.UgoEvidProsireni) error {
	return scanNullable(rows,
		&m.IDUgoEvid, &m.IDUgoOrg, &m.IDSapUgovor, &m.IDSapDobavljac,
		&m.Godina, &m.BrUgovor, &m.UgovorDMS, &m.JN, &m.BrPozPlana,
		&m.PredmetUgovora, &m.Naziv, &m.OtvorenUG, &m.ZZN,
		&m.PocetakUG, &m.KrajUG, &m.VrednostUG, &m.ValutaUG, &m.KursUG,
		&m.KomGrupa, &m.MBrKomerc, &m.NazKomerc, &m.NazivGrPlan, &m.VrsPred, &m.Sluzba,
		&m.OdgZap1, &m.NazivOdgZap1, &m.OdgZap2, &m.NazivOdgZap2,
		&m.OdgZap3, &m.NazivOdgZap3, &m.OdgZap4, &m.NazivOdgZap4,
		&m.OdgZap5, &m.NazivOdgZap5, &m.OdgZap6, &m.NazivOdgZap6,
		&m.Ime, &m.Telefon, &m.Email, &m.Status, &m.DatPri, &m.DatIzm, &m.IDUgoDobLica)
}
