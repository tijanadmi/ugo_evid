package api

import (
	"context"
	"errors"
	"github.com/tijanadmi/ugo_evid/models"
	"github.com/tijanadmi/ugo_evid/repository"
	"github.com/tijanadmi/ugo_evid/util"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type partnerStore struct {
	repository.Store
	username          string
	orgID             int
	orgErr, errorList error
}

func (s *partnerStore) GetUserOrganization(_ context.Context, username, status string) (int, error) {
	s.username = username
	return 3, s.orgErr
}
func (s *partnerStore) GetPartnersPaged(_ context.Context, orgID, offset, limit int) ([]models.Partner, int, error) {
	s.orgID = orgID
	return nil, 0, s.errorList
}
func TestMyPartnersScope(t *testing.T) {
	for _, tc := range []struct {
		path            string
		auth            bool
		status          int
		orgErr, listErr error
	}{
		{"/moji_partneri?id_ugo_org=99", true, 200, nil, nil},
		{"/moji_partneri", false, 401, nil, nil},
		{"/moji_partneri?page_size=101", true, 400, nil, nil},
		{"/moji_partneri", true, 403, repository.ErrUserOrganization, nil},
		{"/moji_partneri", true, 500, nil, errors.New("private database error")},
	} {
		store := &partnerStore{orgErr: tc.orgErr, errorList: tc.listErr}
		server, err := NewServer(util.Config{TokenSymmetricKey: strings.Repeat("k", 32), ActiveUserStatus: "A"}, store)
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest("GET", tc.path, nil)
		if tc.auth {
			access, _, err := server.tokenMaker.CreateToken("ad.user", "user", time.Minute)
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Authorization", "Bearer "+access)
		}
		rec := httptest.NewRecorder()
		server.router.ServeHTTP(rec, req)
		if rec.Code != tc.status {
			t.Fatalf("%s: %d %s", tc.path, rec.Code, rec.Body.String())
		}
		if tc.status == 200 && (store.username != "ad.user" || store.orgID != 3 || !strings.Contains(rec.Body.String(), `"items":[]`)) {
			t.Fatal("wrong scope or empty response")
		}
		if tc.status == 403 && store.orgID != 0 {
			t.Fatal("unassigned user reached partners query")
		}
		if strings.Contains(rec.Body.String(), "private database error") {
			t.Fatal("error leak")
		}
	}
}
