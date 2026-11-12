package api

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/tijanadmi/ugo_evid/models"
	"github.com/tijanadmi/ugo_evid/repository"
	"github.com/tijanadmi/ugo_evid/util"
)

type adminAPIStore struct {
	accessTestStore
	calls   int
	failure error
	actor   int
	inputID int
}

func (s *adminAPIStore) ListAdminUsers(context.Context, string, int, int) ([]models.AdminUser, int, error) {
	s.calls++
	return []models.AdminUser{{ID: 2, Username: "test.user"}}, 1, nil
}
func (s *adminAPIStore) GetAdminUser(context.Context, int) (*models.AdminUser, error) {
	s.calls++
	return &models.AdminUser{ID: 2, Username: "test.user", Version: 1, Grants: []models.RoleGrant{}}, nil
}
func (s *adminAPIStore) AdminCatalog(context.Context) (models.AdminCatalog, error) {
	s.calls++
	return models.AdminCatalog{}, nil
}
func (s *adminAPIStore) SaveAdminUser(_ context.Context, actor int, _ string, u *models.AdminUser) error {
	s.calls++
	s.actor = actor
	s.inputID = u.ID
	u.ID = 2
	return s.failure
}
func (s *adminAPIStore) ListAdminNotifications(context.Context, string, int, int) ([]models.AdminNotification, int, error) {
	s.calls++
	return []models.AdminNotification{}, 0, nil
}
func (s *adminAPIStore) ChangeAdminNotification(_ context.Context, actor int, _ string, id int, version int64, action string) error {
	s.calls++
	s.actor = actor
	return s.failure
}
func (s *adminAPIStore) ListAdminAudit(context.Context, int, int) ([]models.AdminAudit, int, error) {
	s.calls++
	return []models.AdminAudit{}, 0, nil
}

func TestAdminEndpointsRequireGlobalAdmin(t *testing.T) {
	for _, admin := range []bool{false, true} {
		for _, tc := range []struct {
			method, path, body string
			status             int
		}{
			{"GET", "/admin/korisnici", "", 200}, {"GET", "/admin/korisnici/2", "", 200},
			{"GET", "/admin/sifarnici", "", 200}, {"GET", "/admin/obavestenja", "", 200}, {"GET", "/admin/istorija", "", 200},
			{"POST", "/admin/korisnici", `{"id":999,"ad_sifra":"new.user","status":"A","roles":[]}`, 201},
			{"PUT", "/admin/korisnici/2", `{"id":999,"ad_sifra":"test.user","status":"A","version":"1","roles":[]}`, 200},
			{"PUT", "/admin/obavestenja/8", `{"version":"1","action":"PREUZMI"}`, 200},
		} {
			t.Run(tc.method+tc.path+map[bool]string{true: " admin", false: " non-admin"}[admin], func(t *testing.T) {
				s := &adminAPIStore{accessTestStore: accessTestStore{rights: models.Access{UserID: 7, Admin: admin, CalendarAdmin: true}}}
				server, err := NewServer(util.Config{TokenSymmetricKey: strings.Repeat("k", 32), ActiveUserStatus: "A"}, s)
				if err != nil {
					t.Fatal(err)
				}
				token, _, _ := server.tokenMaker.CreateToken("admin", "user", time.Minute)
				req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
				req.Header.Set("Authorization", "Bearer "+token)
				req.Header.Set("Content-Type", "application/json")
				r := httptest.NewRecorder()
				server.router.ServeHTTP(r, req)
				want := 403
				if admin {
					want = tc.status
				}
				if r.Code != want {
					t.Fatalf("%d %s", r.Code, r.Body.String())
				}
				if !admin && s.calls > 0 {
					t.Fatal("calendar admin reached global administration")
				}
				if admin && tc.method == "POST" && s.inputID != 0 {
					t.Fatal("client controlled user ID")
				}
				if admin && tc.method == "PUT" && strings.Contains(tc.path, "korisnici") && s.inputID != 2 {
					t.Fatal("body ID overrode URL")
				}
				if s.actor != 0 && s.actor != 7 {
					t.Fatal("wrong audit actor")
				}
			})
		}
	}
}

func TestLastAdminAPIConflict(t *testing.T) {
	s := &adminAPIStore{accessTestStore: accessTestStore{rights: models.Access{UserID: 7, Admin: true}}, failure: repository.ErrLastAdmin}
	server, err := NewServer(util.Config{TokenSymmetricKey: strings.Repeat("k", 32), ActiveUserStatus: "A"}, s)
	if err != nil {
		t.Fatal(err)
	}
	token, _, _ := server.tokenMaker.CreateToken("admin", "user", time.Minute)
	req := httptest.NewRequest("PUT", "/admin/korisnici/7", strings.NewReader(`{"ad_sifra":"admin","status":"N","version":"1","roles":[]}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	r := httptest.NewRecorder()
	server.router.ServeHTTP(r, req)
	if r.Code != 409 || !strings.Contains(r.Body.String(), "poslednjeg") {
		t.Fatalf("%d %s", r.Code, r.Body.String())
	}
}
