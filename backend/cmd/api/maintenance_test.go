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
		{name: "valid", body: valid, status: 201, writes: 1},
		{name: "custom periods", body: strings.Replace(valid, `"teh_period_meseci":3`, `"teh_period_meseci":null`, 1), status: 201, writes: 1},
		{name: "bad dates", body: strings.Replace(valid, "2026-12-31", "2025-12-31", 1), status: 400},
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
