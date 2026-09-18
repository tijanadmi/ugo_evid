package api

import (
	"context"
	"database/sql"
	"errors"
	"github.com/tijanadmi/ugo_evid/models"
	"github.com/tijanadmi/ugo_evid/repository"
	"github.com/tijanadmi/ugo_evid/util"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type detailStore struct {
	repository.Store
	id  int
	err error
}

func (s *detailStore) GetUgoEvidProsireniByID(_ context.Context, id int) (models.UgoEvidProsireni, error) {
	s.id = id
	return models.UgoEvidProsireni{IDUgoEvid: id, LicaDobavljaca: []models.UgoDobLiceKontakt{{Ime: "Kontakt", RolaLica: "SLM"}}}, s.err
}
func TestContractDetailEndpoint(t *testing.T) {
	for _, tc := range []struct {
		path   string
		auth   bool
		status int
		err    error
	}{
		{"/ugo_evid/7/detalji", true, 200, nil},
		{"/ugo_evid/7/detalji", false, 401, nil},
		{"/ugo_evid/0/detalji", true, 400, nil},
		{"/ugo_evid/abc/detalji", true, 400, nil},
		{"/ugo_evid/7/detalji", true, 404, sql.ErrNoRows},
		{"/ugo_evid/7/detalji", true, 500, errors.New("private database error")},
	} {
		store := &detailStore{err: tc.err}
		server, err := NewServer(util.Config{TokenSymmetricKey: strings.Repeat("k", 32)}, store)
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest("GET", tc.path, nil)
		if tc.auth {
			access, _, err := server.tokenMaker.CreateToken("test", "user", time.Minute)
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
		if (tc.status == 401 || tc.status == 400) && store.id != 0 {
			t.Fatal("invalid request reached database")
		}
		if tc.status == 200 && !strings.Contains(rec.Body.String(), `"rola_lica":"SLM"`) {
			t.Fatal("contacts missing")
		}
		if strings.Contains(rec.Body.String(), "private database error") {
			t.Fatal("database error leaked")
		}
	}
}
