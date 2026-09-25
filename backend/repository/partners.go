package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/tijanadmi/ugo_evid/models"
)

var ErrUserOrganization = errors.New("user must have one active organization")

func (r *OracleStore) GetUserOrganization(ctx context.Context, username, activeStatus string) (int, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT DISTINCT kr.id_ugo_org
 FROM TED.UGO_KOR k JOIN TED.UGO_KOR_ROLE kr ON kr.id_ugo_kor = k.id
 WHERE LOWER(k.ad_sifra) = LOWER(:username) AND k.status = :active_status AND kr.status = 'A'`, sql.Named("username", username), sql.Named("active_status", activeStatus))
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	var orgID int
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return 0, err
		}
		return 0, ErrUserOrganization
	}
	if err := rows.Scan(&orgID); err != nil {
		return 0, err
	}
	if rows.Next() {
		return 0, ErrUserOrganization
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	if orgID < 1 {
		return 0, ErrUserOrganization
	}
	return orgID, nil
}

func (r *OracleStore) GetPartnersPaged(ctx context.Context, orgID, offset, limit int) ([]models.Partner, int, error) {
	if orgID < 1 || offset < 0 || limit < 1 || limit > 100 {
		return nil, 0, fmt.Errorf("invalid partner filter or pagination")
	}
	const from = ` FROM TED.SAP_DOBAVLJACI d WHERE EXISTS (
 SELECT 1 FROM TED.SAP_UGOVORI su JOIN TED.UGO_EVID e ON e.id_sap_ugovor = su.id
 WHERE su.id_sap_dobavljac = d.id AND e.id_ugo_org = :org_id)`
	var total int
	if err := r.DB.QueryRowContext(ctx, "SELECT COUNT(*)"+from, sql.Named("org_id", orgID)).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.DB.QueryContext(ctx, `SELECT d.id,d.sifra,d.naziv,d.adresa,d.grad,d.web_portal`+from+` ORDER BY d.naziv,d.id OFFSET :page_offset ROWS FETCH NEXT :page_limit ROWS ONLY`, sql.Named("org_id", orgID), sql.Named("page_offset", offset), sql.Named("page_limit", limit))
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]models.Partner, 0)
	for rows.Next() {
		var item models.Partner
		if err := scanNullable(rows, &item.ID, &item.Sifra, &item.Naziv, &item.Adresa, &item.Grad, &item.WebPortal); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	if err := rows.Close(); err != nil {
		return nil, 0, err
	}
	// Reuse the batched contact loader and its role ordering for the current page.
	suppliers := make([]models.UgoEvidProsireni, len(items))
	for i := range items {
		suppliers[i].IDSapDobavljac = &items[i].ID
	}
	if err := r.loadProsireniContacts(ctx, suppliers); err != nil {
		return nil, 0, err
	}
	for i := range items {
		items[i].LicaDobavljaca = suppliers[i].LicaDobavljaca
	}
	return items, total, nil
}
