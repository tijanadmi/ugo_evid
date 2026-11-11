package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/tijanadmi/ugo_evid/models"
	"strings"
)

var ErrUserOrganization = errors.New("user must have one active organization")

func (r *OracleStore) CanManagePartner(ctx context.Context, orgID, supplierID int) (bool, error) {
	var count int
	err := r.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM DUAL WHERE EXISTS (
 SELECT 1 FROM TED.SAP_UGOVORI su JOIN TED.UGO_EVID e ON e.id_sap_ugovor = su.id
 WHERE su.id_sap_dobavljac = :supplier_id AND `+accessPredicate(ctx, "e.id_ugo_org", "e.id_ugo_org = :org_id")+`)`,
		sql.Named("supplier_id", supplierID), sql.Named("org_id", orgID)).Scan(&count)
	return count > 0, err
}

func (r *OracleStore) GetUserOrganization(ctx context.Context, username, activeStatus string) (int, error) {
	if a := models.AccessFrom(ctx); a != nil {
		if a.SelectedOrg > 0 {
			return a.SelectedOrg, nil
		}
		// List queries apply accessPredicate; the scalar is only a legacy bind value.
		if len(a.Organizations) > 0 {
			return a.Organizations[0].ID, nil
		}
		return 0, ErrUserOrganization
	}

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

func (r *OracleStore) GetPartnersPaged(ctx context.Context, orgID, offset, limit int, naziv string) ([]models.Partner, int, error) {
	if orgID < 1 || offset < 0 || limit < 1 || limit > 100 {
		return nil, 0, fmt.Errorf("invalid partner filter or pagination")
	}
	from := ` FROM TED.SAP_DOBAVLJACI d WHERE EXISTS (
 SELECT 1 FROM TED.SAP_UGOVORI su JOIN TED.UGO_EVID e ON e.id_sap_ugovor = su.id
 WHERE su.id_sap_dobavljac = d.id AND ` + accessPredicate(ctx, "e.id_ugo_org", "e.id_ugo_org = :org_id") + `)`
	args := []any{sql.Named("org_id", orgID)}
	if naziv = strings.TrimSpace(naziv); naziv != "" {
		from += " AND INSTR(LOWER(d.naziv), LOWER(:naziv)) > 0"
		args = append(args, sql.Named("naziv", naziv))
	}
	var total int
	if err := r.DB.QueryRowContext(ctx, "SELECT COUNT(*)"+from, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, sql.Named("page_offset", offset), sql.Named("page_limit", limit))
	writeScope := "1=1"
	if a := models.AccessFrom(ctx); a != nil {
		writeAccess := *a
		writeAccess.WriteRequest = true
		writeAccess.SelectedOrg = 0
		writeScope = accessPredicate(models.WithAccess(ctx, &writeAccess), "we.id_ugo_org", "1=0")
	}
	rows, err := r.DB.QueryContext(ctx, `SELECT d.id,d.sifra,d.naziv,d.adresa,d.grad,d.web_portal,
 CASE WHEN EXISTS (SELECT 1 FROM TED.UGO_EVID we JOIN TED.SAP_UGOVORI ws ON ws.id=we.id_sap_ugovor
 WHERE ws.id_sap_dobavljac=d.id AND `+writeScope+`) THEN 1 ELSE 0 END`+from+` ORDER BY d.naziv,d.id OFFSET :page_offset ROWS FETCH NEXT :page_limit ROWS ONLY`, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]models.Partner, 0)
	for rows.Next() {
		var item models.Partner
		var writable int
		if err := scanNullable(rows, &item.ID, &item.Sifra, &item.Naziv, &item.Adresa, &item.Grad, &item.WebPortal, &writable); err != nil {
			return nil, 0, err
		}
		item.CanWrite = writable == 1
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
