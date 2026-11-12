package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"regexp"
	"sort"
	"strings"

	"github.com/tijanadmi/ugo_evid/models"
)

var (
	ErrAdminForbidden = errors.New("Nemate administratorska prava.")
	ErrAdminInput     = errors.New("Proverite korisnika, status i dodeljena prava.")
	ErrAdminConflict  = errors.New("Podaci su promenjeni. Ponovo učitajte aktuelne podatke.")
	ErrLastAdmin      = errors.New("Nije dozvoljeno uklanjanje poslednjeg aktivnog administratora.")
	ErrAdminDuplicate = errors.New("Korisnik sa tim AD nalogom već postoji.")
)

type AdminStore interface {
	ListAdminUsers(context.Context, string, int, int) ([]models.AdminUser, int, error)
	GetAdminUser(context.Context, int) (*models.AdminUser, error)
	AdminCatalog(context.Context) (models.AdminCatalog, error)
	SaveAdminUser(context.Context, int, string, *models.AdminUser) error
	ListAdminNotifications(context.Context, string, int, int) ([]models.AdminNotification, int, error)
	ChangeAdminNotification(context.Context, int, string, int, int64, string) error
	ListAdminAudit(context.Context, int, int) ([]models.AdminAudit, int, error)
}

func (r *OracleStore) ListAdminUsers(ctx context.Context, filter string, offset, limit int) ([]models.AdminUser, int, error) {
	where := ` FROM TED.UGO_KOR WHERE INSTR(LOWER(ad_sifra||' '||ime),LOWER(:filter))>0 OR :empty_filter=1`
	args := []any{sql.Named("filter", strings.TrimSpace(filter)), sql.Named("empty_filter", boolInt(strings.TrimSpace(filter) == ""))}
	var total int
	if err := r.DB.QueryRowContext(ctx, "SELECT COUNT(*)"+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, sql.Named("off", offset), sql.Named("lim", limit))
	rows, err := r.DB.QueryContext(ctx, `SELECT id,ad_sifra,ime,status,version`+where+` ORDER BY LOWER(ad_sifra),id OFFSET :off ROWS FETCH NEXT :lim ROWS ONLY`, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := []models.AdminUser{}
	for rows.Next() {
		var u models.AdminUser
		if err := scanNullable(rows, &u.ID, &u.Username, &u.Name, &u.Status, &u.Version); err != nil {
			return nil, 0, err
		}
		items = append(items, u)
	}
	return items, total, rows.Err()
}
func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

type adminQuerier interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func readAdminUser(ctx context.Context, q adminQuerier, id int) (*models.AdminUser, error) {
	// One statement gives a consistent user/version/role snapshot.
	rows, err := q.QueryContext(ctx, `SELECT k.id,k.ad_sifra,k.ime,k.status,k.version,kr.id_ugo_rola,kr.opseg,kr.id_ugo_org
 FROM TED.UGO_KOR k LEFT JOIN TED.UGO_KOR_ROLE kr ON kr.id_ugo_kor=k.id AND kr.status='A'
 WHERE k.id=:id ORDER BY kr.id_ugo_rola,kr.opseg,kr.id_ugo_org`, sql.Named("id", id))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	u := &models.AdminUser{Grants: []models.RoleGrant{}}
	for rows.Next() {
		var g models.RoleGrant
		if err := scanNullable(rows, &u.ID, &u.Username, &u.Name, &u.Status, &u.Version, &g.RoleID, &g.Scope, &g.OrgID); err != nil {
			return nil, err
		}
		if g.RoleID > 0 {
			u.Grants = append(u.Grants, g)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if u.ID == 0 {
		return nil, sql.ErrNoRows
	}
	return u, nil
}
func (r *OracleStore) GetAdminUser(ctx context.Context, id int) (*models.AdminUser, error) {
	return readAdminUser(ctx, r.DB, id)
}

func (r *OracleStore) AdminCatalog(ctx context.Context) (models.AdminCatalog, error) {
	result := models.AdminCatalog{Roles: []models.AdminRole{}, Organizations: []models.OrganizationAccess{}}
	rows, err := r.DB.QueryContext(ctx, `SELECT id,sifra,naziv FROM TED.UGO_ROLA ORDER BY id`)
	if err != nil {
		return result, err
	}
	for rows.Next() {
		var v models.AdminRole
		if err := rows.Scan(&v.ID, &v.Code, &v.Name); err != nil {
			rows.Close()
			return result, err
		}
		result.Roles = append(result.Roles, v)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return result, err
	}
	rows, err = r.DB.QueryContext(ctx, `SELECT id,sifra,naziv FROM TED.UGO_ORG ORDER BY naziv,id`)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	for rows.Next() {
		var v models.OrganizationAccess
		if err := rows.Scan(&v.ID, &v.Code, &v.Name); err != nil {
			return result, err
		}
		result.Organizations = append(result.Organizations, v)
	}
	return result, rows.Err()
}

var adUsername = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,199}$`)

func validateAdminUser(u *models.AdminUser, active string) error {
	u.Username = strings.ToLower(strings.TrimSpace(u.Username))
	u.Name = strings.TrimSpace(u.Name)
	if !adUsername.MatchString(u.Username) || len(u.Name) > 100 || (u.Status != active && u.Status != "N") || len(u.Grants) > 50 || u.ID < 0 {
		return ErrAdminInput
	}
	seen := map[models.RoleGrant]bool{}
	for _, g := range u.Grants {
		valid := (g.RoleID == 1 || g.RoleID == 2) && ((g.Scope == "OJ" && g.OrgID > 0) || (g.Scope == "SVE_OJ" && g.OrgID == 0)) || (g.RoleID == 3 || g.RoleID == 4) && g.Scope == "GLOBALNO" && g.OrgID == 0
		if !valid || seen[g] {
			return ErrAdminInput
		}
		seen[g] = true
	}
	sort.Slice(u.Grants, func(i, j int) bool {
		a, b := u.Grants[i], u.Grants[j]
		if a.RoleID != b.RoleID {
			return a.RoleID < b.RoleID
		}
		if a.Scope != b.Scope {
			return a.Scope < b.Scope
		}
		return a.OrgID < b.OrgID
	})
	return nil
}

// All administrative writers take the same lock BEFORE reading/checking the
// actor and last-admin invariant. This prevents concurrent self-removals.
func lockAdmin(ctx context.Context, tx *sql.Tx, actor int, active string) error {
	var lock int
	if err := tx.QueryRowContext(ctx, `SELECT id FROM TED.PORTAL_ADMIN_LOCK WHERE id=1 FOR UPDATE WAIT 5`).Scan(&lock); err != nil {
		return err
	}
	var count int
	err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM TED.UGO_KOR k WHERE k.id=:actor AND k.status=:active AND EXISTS
 (SELECT 1 FROM TED.UGO_KOR_ROLE kr WHERE kr.id_ugo_kor=k.id AND kr.status='A' AND kr.id_ugo_rola=4 AND kr.opseg='GLOBALNO')`, sql.Named("actor", actor), sql.Named("active", active)).Scan(&count)
	if err != nil {
		return err
	}
	if count != 1 {
		return ErrAdminForbidden
	}
	return nil
}
func adminAudit(ctx context.Context, tx *sql.Tx, actor, id int, action, kind string, before, after any) error {
	b, err := json.Marshal(before)
	if err != nil {
		return err
	}
	a, err := json.Marshal(after)
	if err != nil {
		return err
	}
	if len(b) > 4000 || len(a) > 4000 {
		return ErrAdminInput
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO TED.PORTAL_ADMIN_AUDIT (id_korisnika,akcija,vrsta,id_sloga,pre,posle) VALUES (:actor,:action,:kind,:id,:before,:after)`, sql.Named("actor", actor), sql.Named("action", action), sql.Named("kind", kind), sql.Named("id", id), sql.Named("before", string(b)), sql.Named("after", string(a)))
	return err
}

func (r *OracleStore) SaveAdminUser(ctx context.Context, actor int, active string, u *models.AdminUser) error {
	if err := validateAdminUser(u, active); err != nil {
		return err
	}
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := lockAdmin(ctx, tx, actor, active); err != nil {
		return err
	}
	var before *models.AdminUser
	action := "KREIRANJE_KORISNIKA"
	if u.ID > 0 {
		before, err = readAdminUser(ctx, tx, u.ID)
		if err != nil {
			return err
		}
		if u.Version != before.Version {
			return ErrAdminConflict
		}
		if !strings.EqualFold(strings.TrimSpace(before.Username), u.Username) {
			return ErrAdminInput
		}
		var result sql.Result
		result, err = tx.ExecContext(ctx, `UPDATE TED.UGO_KOR SET ime=:name,status=:status,datizm=SYSDATE,version=version+1 WHERE id=:id AND version=:version`, sql.Named("name", u.Name), sql.Named("status", u.Status), sql.Named("id", u.ID), sql.Named("version", u.Version))
		if err == nil {
			count, countErr := result.RowsAffected()
			if countErr != nil {
				return countErr
			}
			if count != 1 {
				return ErrAdminConflict
			}
		}
		action = "IZMENA_KORISNIKA"
		u.Version++
	} else {
		_, err = tx.ExecContext(ctx, `INSERT INTO TED.UGO_KOR (ad_sifra,ime,status,datpri) VALUES (:username,:name,:status,SYSDATE) RETURNING id INTO :id`, sql.Named("username", u.Username), sql.Named("name", u.Name), sql.Named("status", u.Status), sql.Named("id", sql.Out{Dest: &u.ID}))
		u.Version = 1
	}
	if err != nil {
		if strings.Contains(err.Error(), "ORA-00001") {
			return ErrAdminDuplicate
		}
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE TED.UGO_KOR_ROLE SET status='N',datizm=SYSDATE WHERE id_ugo_kor=:id AND status='A'`, sql.Named("id", u.ID)); err != nil {
		return err
	}
	for _, g := range u.Grants {
		_, err = tx.ExecContext(ctx, `INSERT INTO TED.UGO_KOR_ROLE (id_ugo_kor,id_ugo_rola,opseg,id_ugo_org,status,datpri) VALUES (:user_id,:role_id,:scope,NULLIF(:org_id,0),'A',SYSDATE)`, sql.Named("user_id", u.ID), sql.Named("role_id", g.RoleID), sql.Named("scope", g.Scope), sql.Named("org_id", g.OrgID))
		if err != nil {
			if strings.Contains(err.Error(), "ORA-02291") {
				return ErrAdminInput
			}
			return err
		}
	}
	var admins int
	err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM TED.UGO_KOR k WHERE k.status=:active AND EXISTS
 (SELECT 1 FROM TED.UGO_KOR_ROLE kr WHERE kr.id_ugo_kor=k.id AND kr.status='A' AND kr.id_ugo_rola=4 AND kr.opseg='GLOBALNO')`, sql.Named("active", active)).Scan(&admins)
	if err != nil {
		return err
	}
	if admins < 1 {
		return ErrLastAdmin
	}
	if err := adminAudit(ctx, tx, actor, u.ID, action, "KORISNIK", before, u); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *OracleStore) ListAdminAudit(ctx context.Context, offset, limit int) ([]models.AdminAudit, int, error) {
	var total int
	if err := r.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM TED.PORTAL_ADMIN_AUDIT`).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.DB.QueryContext(ctx, `SELECT a.id,a.id_korisnika,k.ad_sifra,a.akcija,a.vrsta,a.id_sloga,a.pre,a.posle,TO_CHAR(a.vreme,'YYYY-MM-DD"T"HH24:MI:SS') FROM TED.PORTAL_ADMIN_AUDIT a LEFT JOIN TED.UGO_KOR k ON k.id=a.id_korisnika ORDER BY a.id DESC OFFSET :off ROWS FETCH NEXT :lim ROWS ONLY`, sql.Named("off", offset), sql.Named("lim", limit))
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []models.AdminAudit{}
	for rows.Next() {
		var a models.AdminAudit
		if err := scanNullable(rows, &a.ID, &a.UserID, &a.Username, &a.Action, &a.Kind, &a.EntityID, &a.Before, &a.After, &a.Time); err != nil {
			return nil, 0, err
		}
		out = append(out, a)
	}
	return out, total, rows.Err()
}
