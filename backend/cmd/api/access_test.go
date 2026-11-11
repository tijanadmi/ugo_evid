package api

import (
	"context"
	"database/sql"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/tijanadmi/ugo_evid/models"
	"github.com/tijanadmi/ugo_evid/util"
)

type accessTestStore struct {
	documentTestStore
	rights    models.Access
	accessErr error
	usedOrg   int
}

func (s *accessTestStore) LoadAccess(context.Context, string, string) (*models.Access, error) {
	a := s.rights
	return &a, s.accessErr
}
func (s *accessTestStore) GetUgoEvidProsireniByID(_ context.Context, id int) (models.UgoEvidProsireni, error) {
	return models.UgoEvidProsireni{IDUgoEvid: id, IDUgoOrg: id, OtvorenUG: "X"}, nil
}
func (s *accessTestStore) SetContractType(_ context.Context, id, typ int, version int64, org, user int) error {
	s.writes++
	s.usedOrg = org
	return nil
}
func (s *accessTestStore) GetCalendar(context.Context, int) (*models.CalendarYear, error) {
	return &models.CalendarYear{Items: []models.CalendarDay{}}, nil
}
func (s *accessTestStore) ChangeCalendar(context.Context, int, models.CalendarChange) error {
	s.writes++
	return nil
}
func (s *accessTestStore) CanManagePartner(ctx context.Context, _ int, supplier int) (bool, error) {
	a := models.AccessFrom(ctx)
	return a.Allows(supplier, a.WriteRequest), nil
}
func (s *accessTestStore) GetUgoDobLiceById(_ context.Context, id int) (*models.UgoDobLice, error) {
	return &models.UgoDobLice{ID: id, SapDobavljac: models.SapDobavljac{ID: id}}, nil
}
func (s *accessTestStore) GetDocument(_ context.Context, id int) (*models.Document, error) {
	return &models.Document{ID: id, ContractID: id, Status: "A"}, nil
}
func (s *accessTestStore) GetUnregisteredContracts(_ context.Context, org, offset, limit int, filter string) ([]models.ContractChoice, int, error) {
	s.usedOrg = org
	return []models.ContractChoice{}, 0, nil
}

func TestMixedOrganizationPermissions(t *testing.T) {
	for _, tc := range []struct {
		name, method, path, body   string
		admin, onlyAdmin, inactive bool
		status, writes             int
	}{
		{name: "GLE detail", method: "GET", path: "/ugo_evid/1/detalji", status: 200},
		{name: "IUD includes read", method: "GET", path: "/ugo_evid/4/detalji", status: 200},
		{name: "foreign detail", method: "GET", path: "/ugo_evid/5/detalji", status: 403},
		{name: "GLE type write", method: "PUT", path: "/ugo_evid/1/tip", body: `{"id_tip_ugovora":1,"version":"1"}`, status: 403},
		{name: "IUD type write", method: "PUT", path: "/ugo_evid/4/tip", body: `{"id_tip_ugovora":1,"version":"1"}`, status: 200, writes: 1},
		{name: "foreign type write", method: "PUT", path: "/ugo_evid/5/tip", body: `{"id_tip_ugovora":1,"version":"1"}`, status: 403},
		{name: "GLE upload", method: "POST", path: "/ugo_evid/1/dokumenti", status: 403},
		{name: "GLE lock", method: "POST", path: "/ugo_evid/1/odrzavanje/lock", status: 403},
		{name: "GLE preview", method: "POST", path: "/ugo_evid/1/pracenje/preview", status: 403},
		{name: "GLE document metadata", method: "GET", path: "/ugo_doc/1", status: 200},
		{name: "foreign document metadata", method: "GET", path: "/ugo_doc/5", status: 403},
		{name: "foreign document bytes", method: "GET", path: "/ugo_doc/5/sadrzaj", status: 403},
		{name: "GLE document lease", method: "POST", path: "/ugo_doc/1/lock", status: 403},
		{name: "GLE document delete", method: "DELETE", path: "/ugo_doc/1", body: `{}`, status: 403},
		{name: "GLE contact read", method: "GET", path: "/ugo_dob_lica/1", status: 200},
		{name: "GLE contact lock", method: "POST", path: "/ugo_dob_lica/1/lock", status: 403},
		{name: "foreign contact read", method: "GET", path: "/ugo_dob_lica/5", status: 403},
		{name: "IUD is not calendar admin", method: "POST", path: "/radni_kalendar/2026", body: `{}`, status: 403},
		{name: "IUD cannot import holidays", method: "GET", path: "/radni_kalendar/2026/praznici", status: 403},
		{name: "calendar read", method: "GET", path: "/radni_kalendar/2026", status: 200},
		{name: "calendar-only admin", method: "POST", path: "/radni_kalendar/2026", body: `{}`, admin: true, onlyAdmin: true, status: 200, writes: 1},
		{name: "calendar grants no contracts", method: "GET", path: "/ugo_evid/1/detalji", admin: true, onlyAdmin: true, status: 403},
		{name: "dictionary not granted", method: "POST", path: "/ugo_org", body: `{}`, admin: true, status: 403},
		{name: "unscoped contacts disabled", method: "GET", path: "/ugo_dob_lica", status: 403},
		{name: "unscoped SAP disabled", method: "GET", path: "/sapugovori", status: 403},
		{name: "foreign export", method: "GET", path: "/ugo_evid/otvoreni/excel?id_ugo_org=5", status: 403},
		{name: "GLE registration denied", method: "GET", path: "/sapugovori/neevidentirani?id_ugo_org=1", status: 403},
		{name: "IUD registration selected", method: "GET", path: "/sapugovori/neevidentirani?id_ugo_org=4", status: 200},
		{name: "single writable OJ automatic", method: "GET", path: "/sapugovori/neevidentirani", status: 200},
		{name: "active token inactive user", method: "GET", path: "/users/prava", inactive: true, status: 403},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := &accessTestStore{rights: models.Access{UserID: 2, CalendarAdmin: tc.admin, Organizations: []models.OrganizationAccess{{ID: 1}, {ID: 2}, {ID: 3}, {ID: 4, Write: true}}}}
			if tc.onlyAdmin {
				s.rights.Organizations = nil
			}
			if tc.inactive {
				s.accessErr = sql.ErrNoRows
			}
			server, err := NewServer(util.Config{TokenSymmetricKey: strings.Repeat("k", 32), ActiveUserStatus: "A"}, s)
			if err != nil {
				t.Fatal(err)
			}
			token, _, _ := server.tokenMaker.CreateToken("user", "user", time.Minute)
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Content-Type", "application/json")
			r := httptest.NewRecorder()
			server.router.ServeHTTP(r, req)
			if r.Code != tc.status || s.writes != tc.writes {
				t.Fatalf("status=%d writes=%d body=%s", r.Code, s.writes, r.Body.String())
			}
			if s.usedOrg != 0 && s.usedOrg != 4 {
				t.Fatalf("wrong write org: %d", s.usedOrg)
			}
		})
	}
}

func TestRightsRevocationAndMultipleWritableOrganizations(t *testing.T) {
	s := &accessTestStore{rights: models.Access{UserID: 2, Organizations: []models.OrganizationAccess{{ID: 1, Write: true}, {ID: 4, Write: true}}}}
	server, err := NewServer(util.Config{TokenSymmetricKey: strings.Repeat("k", 32), ActiveUserStatus: "A"}, s)
	if err != nil {
		t.Fatal(err)
	}
	token, _, _ := server.tokenMaker.CreateToken("user", "user", time.Minute)
	call := func(method, path, body string, want int) {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		r := httptest.NewRecorder()
		server.router.ServeHTTP(r, req)
		if r.Code != want {
			t.Fatalf("%s %s: %d %s", method, path, r.Code, r.Body.String())
		}
	}
	call("GET", "/sapugovori/neevidentirani", "", 400)
	call("GET", "/sapugovori/neevidentirani?id_ugo_org=4", "", 200)
	call("PUT", "/ugo_evid/4/tip", `{"id_tip_ugovora":1,"version":"1"}`, 200)
	s.rights.Organizations[1].Write = false
	call("PUT", "/ugo_evid/4/tip", `{"id_tip_ugovora":1,"version":"1"}`, 403)
	call("GET", "/ugo_evid/4/detalji", "", 200)
	s.rights.Organizations = s.rights.Organizations[:1]
	call("GET", "/ugo_evid/4/detalji", "", 403)
	if s.writes != 1 {
		t.Fatalf("writes after revoke: %d", s.writes)
	}
}
