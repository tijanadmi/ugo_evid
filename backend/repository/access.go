package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/tijanadmi/ugo_evid/models"
)

type AccessStore interface {
	LoadAccess(context.Context, string, string) (*models.Access, error)
}

func (r *OracleStore) LoadAccess(ctx context.Context, username, status string) (*models.Access, error) {
	a := &models.Access{Organizations: []models.OrganizationAccess{}}
	var calendar, admin int
	err := r.DB.QueryRowContext(ctx, `SELECT k.id,
 CASE WHEN EXISTS (SELECT 1 FROM TED.UGO_KOR_ROLE kr JOIN TED.UGO_ROLA r ON r.id=kr.id_ugo_rola
 WHERE kr.id_ugo_kor=k.id AND kr.status='A' AND r.sifra IN ('ADM_KALENDAR','ADMIN') AND kr.opseg='GLOBALNO') THEN 1 ELSE 0 END,
 CASE WHEN EXISTS (SELECT 1 FROM TED.UGO_KOR_ROLE kr JOIN TED.UGO_ROLA r ON r.id=kr.id_ugo_rola
 WHERE kr.id_ugo_kor=k.id AND kr.status='A' AND r.sifra='ADMIN' AND kr.opseg='GLOBALNO') THEN 1 ELSE 0 END
 FROM TED.UGO_KOR k WHERE LOWER(k.ad_sifra)=LOWER(:username) AND k.status=:status`,
		sql.Named("username", username), sql.Named("status", status)).Scan(&a.UserID, &calendar, &admin)
	if err != nil {
		return nil, err
	}
	a.CalendarAdmin = calendar == 1
	a.Admin = admin == 1
	rows, err := r.DB.QueryContext(ctx, `SELECT o.id,o.sifra,o.naziv,
 MAX(CASE WHEN r.sifra='IUD' THEN 1 ELSE 0 END)
 FROM TED.UGO_ORG o JOIN TED.UGO_KOR_ROLE kr ON
 ((kr.opseg='OJ' AND kr.id_ugo_org=o.id) OR kr.opseg='SVE_OJ')
 JOIN TED.UGO_ROLA r ON r.id=kr.id_ugo_rola
 WHERE kr.id_ugo_kor=:user_id AND kr.status='A' AND r.sifra IN ('GLE','IUD')
 GROUP BY o.id,o.sifra,o.naziv ORDER BY o.naziv,o.id`, sql.Named("user_id", a.UserID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var o models.OrganizationAccess
		var write int
		if err := rows.Scan(&o.ID, &o.Code, &o.Name, &write); err != nil {
			return nil, err
		}
		o.Write = write == 1
		a.Organizations = append(a.Organizations, o)
	}
	return a, rows.Err()
}

// column and fallback are server-owned SQL fragments, never request strings.
// UserID and SelectedOrg are validated integers. EXISTS keeps pagination and
// counts correct even when multiple assignments grant access to the same OJ.
func accessPredicate(ctx context.Context, column, fallback string) string {
	a := models.AccessFrom(ctx)
	if a == nil {
		return fallback
	}
	roles := "'GLE','IUD'"
	if a.WriteRequest {
		roles = "'IUD'"
	}
	p := fmt.Sprintf(`EXISTS (SELECT 1 FROM TED.UGO_KOR_ROLE ak JOIN TED.UGO_ROLA ar ON ar.id=ak.id_ugo_rola
 WHERE ak.id_ugo_kor=%d AND ak.status='A' AND ar.sifra IN (%s)
 AND ((ak.opseg='OJ' AND ak.id_ugo_org=%s) OR ak.opseg='SVE_OJ'))`, a.UserID, roles, column)
	if a.SelectedOrg > 0 {
		p += fmt.Sprintf(" AND %s=%d", column, a.SelectedOrg)
	}
	if _, bind, ok := strings.Cut(fallback, ":"); ok {
		p += " AND :" + strings.TrimSpace(bind) + ">0"
	}
	return "(" + p + ")"
}
