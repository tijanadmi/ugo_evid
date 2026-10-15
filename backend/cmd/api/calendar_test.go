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

func (s *maintenanceTestStore) GetCalendar(_ context.Context, year int) (*models.CalendarYear, error) {
	return &models.CalendarYear{Items: []models.CalendarDay{}, Version: strings.Repeat("a", 64)}, nil
}
func (s *maintenanceTestStore) ChangeCalendar(_ context.Context, year int, m models.CalendarChange) error {
	s.writes++
	return s.failure
}
func TestCalendarAPI(t *testing.T) {
	for _, tc := range []struct {
		name, method, path, body string
		anonymous                bool
		failure                  error
		status, writes           int
	}{
		{name: "read", method: "GET", path: "2026", status: 200},
		{name: "generate", method: "POST", path: "2026", body: `{"action":"generate","version":"` + strings.Repeat("a", 64) + `"}`, status: 200, writes: 1},
		{name: "unauthenticated", method: "POST", path: "2026", body: `{}`, anonymous: true, status: 401},
		{name: "invalid year", method: "GET", path: "0", status: 400},
		{name: "bad JSON", method: "POST", path: "2026", body: `{`, status: 400},
		{name: "stale", method: "POST", path: "2026", body: `{}`, failure: repository.ErrCalendarConflict, status: 409, writes: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := &maintenanceTestStore{failure: tc.failure}
			server, e := NewServer(util.Config{TokenSymmetricKey: strings.Repeat("k", 32), ActiveUserStatus: "A"}, store)
			if e != nil {
				t.Fatal(e)
			}
			req := httptest.NewRequest(tc.method, "/radni_kalendar/"+tc.path, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			if !tc.anonymous {
				token, _, _ := server.tokenMaker.CreateToken("test.user", "user", time.Minute)
				req.Header.Set("Authorization", "Bearer "+token)
			}
			rec := httptest.NewRecorder()
			server.router.ServeHTTP(rec, req)
			if rec.Code != tc.status || store.writes != tc.writes {
				t.Fatalf("%d %s writes=%d", rec.Code, rec.Body.String(), store.writes)
			}
		})
	}
}
