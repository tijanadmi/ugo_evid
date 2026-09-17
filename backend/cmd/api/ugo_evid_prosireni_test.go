package api

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/tijanadmi/ugo_evid/models"
	"github.com/tijanadmi/ugo_evid/repository"
	"github.com/tijanadmi/ugo_evid/util"
)

type prosireniStore struct {
	repository.Store
	called        bool
	open          bool
	orgID         int
	offset, limit int
	err           error
}

func (s *prosireniStore) GetUgoEvidProsireniPaged(_ context.Context, open bool, offset, limit, orgID int) ([]models.UgoEvidProsireni, int, error) {
	s.called = true
	s.open = open
	s.orgID = orgID
	s.offset = offset
	s.limit = limit
	return nil, 0, s.err
}

func TestProsireniEndpoints(t *testing.T) {
	for _, tc := range []struct {
		name, path    string
		auth          bool
		status        int
		open          bool
		offset, limit int
		dbErr         error
	}{
		{"open defaults", "/ugo_evid/otvoreni", true, 200, true, 0, 20, nil},
		{"closed page", "/ugo_evid/zatvoreni?page_id=3&page_size=5", true, 200, false, 10, 5, nil},
		{"open unauthorized", "/ugo_evid/otvoreni", false, 401, false, 0, 0, nil},
		{"closed unauthorized", "/ugo_evid/zatvoreni", false, 401, false, 0, 0, nil},
		{"invalid page", "/ugo_evid/otvoreni?page_id=0", true, 400, false, 0, 0, nil},
		{"invalid size", "/ugo_evid/zatvoreni?page_size=101", true, 400, false, 0, 0, nil},
		{"database failure", "/ugo_evid/otvoreni", true, 500, true, 0, 20, errors.New("internal database details")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := &prosireniStore{err: tc.dbErr}
			server, err := NewServer(util.Config{TokenSymmetricKey: strings.Repeat("k", 32)}, store)
			if err != nil {
				t.Fatal(err)
			}
			req := httptest.NewRequest("GET", tc.path, nil)
			if tc.auth {
				access, _, err := server.tokenMaker.CreateToken("ad.account", "user", time.Minute)
				if err != nil {
					t.Fatal(err)
				}
				req.Header.Set("Authorization", "Bearer "+access)
			}
			rec := httptest.NewRecorder()
			server.router.ServeHTTP(rec, req)
			if rec.Code != tc.status {
				t.Fatalf("got %d: %s", rec.Code, rec.Body.String())
			}
			if tc.status == 200 || tc.status == 500 {
				if !store.called || store.open != tc.open || store.offset != tc.offset || store.limit != tc.limit {
					t.Fatalf("wrong repository call: %+v", store)
				}
			} else if store.called {
				t.Fatal("repository called for rejected request")
			}
			if tc.status == 200 && strings.TrimSpace(rec.Body.String()) != `{"total":0,"items":[]}` {
				t.Fatalf("unexpected response: %s", rec.Body.String())
			}
			if strings.Contains(rec.Body.String(), "internal database details") {
				t.Fatal("internal error leaked")
			}
		})
	}
}

func TestProsireniOrganizationParameter(t *testing.T) {
	for _, path := range []string{"/ugo_evid/otvoreni", "/ugo_evid/zatvoreni"} {
		for _, tc := range []struct {
			query         string
			status, orgID int
		}{
			{"", 200, 0}, {"?id_ugo_org=0", 200, 0}, {"?id_ugo_org=7", 200, 7},
			{"?id_ugo_org=-1", 400, 0}, {"?id_ugo_org=abc", 400, 0},
			{"?id_ugo_org=%25", 400, 0}, {"?id_ugo_org=1.5", 400, 0},
		} {
			store := &prosireniStore{}
			server, err := NewServer(util.Config{TokenSymmetricKey: strings.Repeat("k", 32)}, store)
			if err != nil {
				t.Fatal(err)
			}
			access, _, err := server.tokenMaker.CreateToken("ad.account", "user", time.Minute)
			if err != nil {
				t.Fatal(err)
			}
			req := httptest.NewRequest("GET", path+tc.query, nil)
			req.Header.Set("Authorization", "Bearer "+access)
			rec := httptest.NewRecorder()
			server.router.ServeHTTP(rec, req)
			if rec.Code != tc.status {
				t.Fatalf("%s%s: %d %s", path, tc.query, rec.Code, rec.Body.String())
			}
			if tc.status == 200 && (!store.called || store.orgID != tc.orgID || store.open != (path == "/ugo_evid/otvoreni")) {
				t.Fatalf("wrong filter: %+v", store)
			}
			if tc.status == 400 && store.called {
				t.Fatal("invalid parameter reached repository")
			}
		}
	}
}
