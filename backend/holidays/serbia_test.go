package holidays

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func fixture(year int, easter string) []externalHoliday {
	out := []externalHoliday{}
	for _, suffix := range []string{"01-01", "01-02", "01-07", "02-15", "02-16", "05-01", "05-02", "11-11"} {
		out = append(out, externalHoliday{Date: fmt.Sprintf("%d-%s", year, suffix), Name: "Holiday", Country: "RS", National: true, Types: []string{"Public"}})
	}
	return append(out, externalHoliday{Date: easter, Name: "Easter Sunday", Country: "RS", National: true, Types: []string{"Public"}})
}
func TestSerbianObservedAndEasterDays(t *testing.T) {
	for _, tc := range []struct {
		year                              int
		easter, observed, christmasMonday string
	}{
		{2026, "2026-04-12", "2026-02-17", ""},
		{2022, "2022-04-24", "2022-05-03", ""},
		{2023, "2023-04-16", "2023-01-03", ""},
		{2024, "2024-05-05", "", "2024-01-08"},
		{2021, "2021-05-02", "2021-05-04", ""},
	} {
		t.Run(fmt.Sprint(tc.year), func(t *testing.T) {
			input := fixture(tc.year, tc.easter)
			input = append(input, externalHoliday{Date: fmt.Sprintf("%d-06-28", tc.year), Name: "Observance", Country: "RS", National: true, Types: []string{"Observance"}})
			raw, _ := json.Marshal(input)
			items, e := parse(raw, tc.year)
			if e != nil {
				t.Fatal(e)
			}
			found := map[string]bool{}
			for _, h := range items {
				found[h.Date] = true
			}
			if tc.observed != "" && !found[tc.observed] {
				t.Fatalf("missing observed %s", tc.observed)
			}
			if tc.christmasMonday != "" && found[tc.christmasMonday] {
				t.Fatal("religious holiday incorrectly transferred")
			}
			easter, _ := time.Parse("2006-01-02", tc.easter)
			for i := -2; i <= 1; i++ {
				if !found[easter.AddDate(0, 0, i).Format("2006-01-02")] {
					t.Fatal("missing Easter day")
				}
			}
			if found[fmt.Sprintf("%d-06-28", tc.year)] {
				t.Fatal("observance imported as nonworking")
			}
		})
	}
}
func TestHolidayHTTPFailures(t *testing.T) {
	raw, _ := json.Marshal(fixture(2026, "2026-04-12"))
	for _, tc := range []struct {
		name      string
		status    int
		body      string
		wantError bool
	}{
		{"ok", 200, string(raw), false}, {"server error", 503, `[]`, true}, {"HTML", 200, "<html>error</html>", true},
		{"empty", 200, `[]`, true}, {"too big", 200, strings.Repeat("x", 1024*1024+1), true},
		{"wrong country", 200, strings.ReplaceAll(string(raw), `"RS"`, `"US"`), true},
		{"wrong year", 200, strings.ReplaceAll(string(raw), "2026-", "2025-"), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/2026" {
					t.Error(r.URL.Path)
				}
				w.WriteHeader(tc.status)
				w.Write([]byte(tc.body))
			}))
			defer server.Close()
			_, e := fetch(context.Background(), 2026, server.Client(), server.URL+"/")
			if (e != nil) != tc.wantError {
				t.Fatal(e)
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := fetch(ctx, 2026, client, "https://holidays.example.test/RS/"); e == nil {
		t.Fatal("cancelled request succeeded")
	}
}
