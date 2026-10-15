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

func (s *maintenanceTestStore) SaveTracking(_ context.Context, id, org, user int, kind string, m models.TrackingChange) error {
	if id != 9 || org != 3 || user != 7 || kind != "zahtevi" {
		panic("incorrect identity")
	}
	s.writes++
	return s.failure
}
func TestTrackingAPI(t *testing.T) {
	body := `{"lock_token":"` + strings.Repeat("a", 64) + `","items":[{"broj_casova":20}]}`
	for _, tc := range []struct {
		name, body      string
		foreign, closed bool
		failure         error
		status, writes  int
	}{
		{name: "authorized", body: body, status: 200, writes: 1},
		{name: "fractional", body: strings.Replace(body, ":20", ":1.5", 1), status: 400},
		{name: "no lease", body: `{"items":[]}`, status: 400},
		{name: "foreign", body: body, foreign: true, status: 403},
		{name: "closed", body: body, closed: true, status: 409},
		{name: "expired", body: body, failure: repository.ErrMaintenanceLease, status: 409, writes: 1},
		{name: "fund exceeded", body: body, failure: repository.ErrTrackingFunds, status: 400, writes: 1},
		{name: "missing calendar", body: body, failure: repository.ErrTrackingCalendar, status: 400, writes: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := &maintenanceTestStore{documentTestStore: documentTestStore{foreign: tc.foreign, closed: tc.closed}, failure: tc.failure}
			server, e := NewServer(util.Config{TokenSymmetricKey: strings.Repeat("k", 32), ActiveUserStatus: "A"}, store)
			if e != nil {
				t.Fatal(e)
			}
			token, _, _ := server.tokenMaker.CreateToken("test.user", "user", time.Minute)
			req := httptest.NewRequest("POST", "/ugo_evid/9/pracenje/zahtevi", strings.NewReader(tc.body))
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			server.router.ServeHTTP(rec, req)
			if rec.Code != tc.status || store.writes != tc.writes {
				t.Fatalf("%d %s writes %d", rec.Code, rec.Body.String(), store.writes)
			}
		})
	}
}
