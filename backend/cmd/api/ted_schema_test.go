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

type contactStore struct {
	repository.Store
	saved *models.UgoDobLice
}

func (s *contactStore) InsertUgoDobLice(_ context.Context, m *models.UgoDobLice) (*models.UgoDobLice, error) {
	s.saved = m
	m.ID = 12
	return m, nil
}
func (s *contactStore) UpdateUgoDobLice(_ context.Context, m *models.UgoDobLice, userID int, token string) (*models.UgoDobLice, error) {
	s.saved = m
	return m, nil
}

func TestContactAPIUsesServerOrganization(t *testing.T) {
	store := &contactStore{}
	server, err := NewServer(util.Config{TokenSymmetricKey: strings.Repeat("k", 32), ActiveUserStatus: "A"}, store)
	if err != nil {
		t.Fatal(err)
	}
	access, _, err := server.tokenMaker.CreateToken("ad.account", "user", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		method, path string
		wantID       int
	}{{"POST", "/ugo_dob_lica", 12}, {"PUT", "/ugo_dob_lica/34", 34}} {
		request := httptest.NewRequest(tc.method, tc.path, strings.NewReader(`{"id_sap_dobavljac":9,"ime":"Kontakt","status":"A","version":"1","lock_token":"`+strings.Repeat("a", 64)+`"}`))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Authorization", "Bearer "+access)
		response := httptest.NewRecorder()
		server.router.ServeHTTP(response, request)
		if response.Code != 200 {
			t.Fatalf("%s: %d %s", tc.path, response.Code, response.Body.String())
		}
		if store.saved == nil || store.saved.ID != tc.wantID || store.saved.SapDobavljac.ID != 9 {
			t.Fatalf("wrong contact: %+v", store.saved)
		}
		if strings.Contains(response.Body.String(), "ugo_org") {
			t.Fatal("nonexistent organization returned")
		}
	}
}

func TestSAPRoutesExposeOnlyGET(t *testing.T) {
	server, err := NewServer(util.Config{TokenSymmetricKey: strings.Repeat("k", 32)}, loginStore{})
	if err != nil {
		t.Fatal(err)
	}
	for _, route := range server.router.Routes() {
		if strings.HasPrefix(route.Path, "/sap") && route.Method != "GET" {
			t.Fatalf("SAP mutation route: %+v", route)
		}
	}
	for _, method := range []string{"POST", "PUT", "PATCH", "DELETE"} {
		rec := httptest.NewRecorder()
		server.router.ServeHTTP(rec, httptest.NewRequest(method, "/sapugovori", nil))
		if rec.Code != 404 {
			t.Fatalf("unexpected SAP write route: %s %d", method, rec.Code)
		}
	}
}

func (s *contactStore) GetUserByUsername(context.Context, string) (*models.User, error) {
	return &models.User{ID: 7, Status: "A"}, nil
}
func (s *contactStore) GetUserOrganization(context.Context, string, string) (int, error) {
	return 3, nil
}
func (s *contactStore) CanManagePartner(context.Context, int, int) (bool, error) { return true, nil }
func (s *contactStore) GetUgoDobLiceById(_ context.Context, id int) (*models.UgoDobLice, error) {
	return &models.UgoDobLice{ID: id, Version: 1, SapDobavljac: models.SapDobavljac{ID: 9}}, nil
}
