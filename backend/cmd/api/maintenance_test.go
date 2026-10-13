package api

import (
	"context"
	"github.com/tijanadmi/ugo_evid/models"
	"github.com/tijanadmi/ugo_evid/repository"
	"github.com/tijanadmi/ugo_evid/util"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type maintenanceTestStore struct {
	documentTestStore
	failure error
}

func (s *maintenanceTestStore) CreateMaintenance(_ context.Context, id int, m models.Maintenance, org, user int) error {
	s.writes++
	if id != 9 || org != 3 || user != 7 {
		panic("wrong identity")
	}
	return s.failure
}
func TestCreateMaintenanceAPI(t *testing.T) {
	valid := `{"datum_potpisa":"2026-01-01","datum_kraja":"2026-12-31","ima_razvojnu_podrsku":true,"teh_period_meseci":3,"id_teh_tip_doc":5,"teh_podsetnik_dana":7,"raz_podsetnik_dana":3}`
	for _, tc := range []struct {
		name, body      string
		foreign, closed bool
		failure         error
		status, writes  int
	}{
		{name: "fractional hours", body: strings.Replace(valid, `"id_teh_tip_doc":5`, `"id_teh_tip_doc":5,"fondovi":[{"id_vrsta_rada":1,"ugovoreno_casova":1.5}]`, 1), status: 400},
		{name: "valid", body: valid, status: 201, writes: 1},
		{name: "custom periods", body: strings.Replace(valid, `"teh_period_meseci":3`, `"teh_period_meseci":null`, 1), status: 201, writes: 1},
		{name: "invalid override", body: strings.Replace(valid, `"datum_potpisa":"2026-01-01"`, `"pocetak_odrzavanja":"bad"`, 1), status: 400},
		{name: "fractional months", body: strings.Replace(valid, `"teh_period_meseci":3`, `"teh_period_meseci":1.5`, 1), status: 400},
		{name: "negative reminder", body: strings.Replace(valid, `"teh_podsetnik_dana":7`, `"teh_podsetnik_dana":-1`, 1), status: 400},
		{name: "foreign", body: valid, foreign: true, status: 403},
		{name: "closed", body: valid, closed: true, status: 409},
		{name: "duplicate", body: valid, failure: repository.ErrMaintenanceConflict, status: 409, writes: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := &maintenanceTestStore{documentTestStore: documentTestStore{foreign: tc.foreign, closed: tc.closed}, failure: tc.failure}
			server, err := NewServer(util.Config{TokenSymmetricKey: strings.Repeat("k", 32), ActiveUserStatus: "A"}, store)
			if err != nil {
				t.Fatal(err)
			}
			access, _, _ := server.tokenMaker.CreateToken("test.user", "user", time.Minute)
			req := httptest.NewRequest("POST", "/ugo_evid/9/odrzavanje", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+access)
			rec := httptest.NewRecorder()
			server.router.ServeHTTP(rec, req)
			if rec.Code != tc.status || store.writes != tc.writes {
				t.Fatalf("status=%d writes=%d body=%s", rec.Code, store.writes, rec.Body.String())
			}
		})
	}
}

func (s *maintenanceTestStore) SetContractType(_ context.Context, id, typeID int, version int64, org, user int) error {
	s.writes++
	if id != 9 || typeID != 1 || version != 2 || org != 3 || user != 7 {
		panic("wrong type request")
	}
	return s.failure
}
func TestSetContractTypeAPI(t *testing.T) {
	for _, tc := range []struct {
		name, body      string
		foreign, closed bool
		failure         error
		status, writes  int
	}{
		{name: "valid", body: `{"id_tip_ugovora":1,"version":"2"}`, status: 200, writes: 1},
		{name: "missing version", body: `{"id_tip_ugovora":1}`, status: 400},
		{name: "invalid type", body: `{"id_tip_ugovora":0,"version":"2"}`, status: 400},
		{name: "foreign", body: `{"id_tip_ugovora":1,"version":"2"}`, foreign: true, status: 403},
		{name: "closed", body: `{"id_tip_ugovora":1,"version":"2"}`, closed: true, status: 409},
		{name: "stale", body: `{"id_tip_ugovora":1,"version":"2"}`, failure: repository.ErrMaintenanceConflict, status: 409, writes: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := &maintenanceTestStore{documentTestStore: documentTestStore{foreign: tc.foreign, closed: tc.closed}, failure: tc.failure}
			server, err := NewServer(util.Config{TokenSymmetricKey: strings.Repeat("k", 32), ActiveUserStatus: "A"}, store)
			if err != nil {
				t.Fatal(err)
			}
			access, _, _ := server.tokenMaker.CreateToken("test.user", "user", time.Minute)
			req := httptest.NewRequest("PUT", "/ugo_evid/9/tip", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+access)
			rec := httptest.NewRecorder()
			server.router.ServeHTTP(rec, req)
			if rec.Code != tc.status || store.writes != tc.writes {
				t.Fatalf("status=%d writes=%d body=%s", rec.Code, store.writes, rec.Body.String())
			}
		})
	}
}

func (s *maintenanceTestStore) UpdateMaintenance(_ context.Context, id int, m models.Maintenance, org, user int) error {
	s.writes++
	if id != 9 || org != 3 || user != 7 || m.Version != 2 || m.Token != strings.Repeat("a", 64) {
		panic("incorrect update ownership")
	}
	return s.failure
}
func (s *maintenanceTestStore) MaintenanceLock(_ context.Context, id, org, user int, action, token string) (*models.EditLock, error) {
	s.writes++
	if id != 9 || org != 3 || user != 7 {
		panic("incorrect lease identity")
	}
	return &models.EditLock{Token: strings.Repeat("a", 64)}, s.failure
}
func TestMaintenanceEditEndpoints(t *testing.T) {
	body := `{"version":"2","lock_token":"` + strings.Repeat("a", 64) + `","id_teh_tip_doc":5,"teh_period_meseci":3}`
	for _, tc := range []struct {
		name, method, path, body string
		foreign, closed          bool
		failure                  error
		status, writes           int
	}{
		{name: "update", method: "PUT", path: "/odrzavanje", body: body, status: 200, writes: 1},
		{name: "missing lease", method: "PUT", path: "/odrzavanje", body: `{"id_teh_tip_doc":5,"version":"2"}`, status: 400},
		{name: "wrong owner", method: "PUT", path: "/odrzavanje", body: body, foreign: true, status: 403},
		{name: "lost lease", method: "PUT", path: "/odrzavanje", body: body, failure: repository.ErrMaintenanceLease, status: 409, writes: 1},
		{name: "acquire", method: "POST", path: "/odrzavanje/lock", status: 200, writes: 1},
		{name: "locked", method: "POST", path: "/odrzavanje/lock", failure: repository.ErrMaintenanceLocked, status: 423, writes: 1},
		{name: "closed acquire", method: "POST", path: "/odrzavanje/lock", closed: true, status: 409},
		{name: "release closed", method: "DELETE", path: "/odrzavanje/lock", body: body, closed: true, status: 200, writes: 1},
		{name: "renew", method: "PUT", path: "/odrzavanje/lock", body: body, status: 200, writes: 1},
		{name: "bad release token", method: "DELETE", path: "/odrzavanje/lock", body: `{"lock_token":"bad"}`, status: 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := &maintenanceTestStore{documentTestStore: documentTestStore{foreign: tc.foreign, closed: tc.closed}, failure: tc.failure}
			server, err := NewServer(util.Config{TokenSymmetricKey: strings.Repeat("k", 32), ActiveUserStatus: "A"}, store)
			if err != nil {
				t.Fatal(err)
			}
			access, _, _ := server.tokenMaker.CreateToken("test.user", "user", time.Minute)
			req := httptest.NewRequest(tc.method, "/ugo_evid/9"+tc.path, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+access)
			rec := httptest.NewRecorder()
			server.router.ServeHTTP(rec, req)
			if rec.Code != tc.status || store.writes != tc.writes {
				t.Fatalf("status=%d calls=%d body=%s", rec.Code, store.writes, rec.Body.String())
			}
		})
	}
}
