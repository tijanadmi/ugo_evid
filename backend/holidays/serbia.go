// Package holidays imports a preview, never writes to the application's calendar.
package holidays

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/tijanadmi/ugo_evid/models"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

const Endpoint = "https://nagerholidays.com/api/v4/Holidays/RS/"

var ErrResponse = errors.New("holiday service returned invalid or incomplete data")
var client = &http.Client{Timeout: 12 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}

func Fetch(ctx context.Context, year int) ([]models.CalendarHoliday, error) {
	return fetch(ctx, year, client, Endpoint)
}
func fetch(ctx context.Context, year int, c *http.Client, endpoint string) ([]models.CalendarHoliday, error) {
	if year < 1900 || year > 2200 {
		return nil, ErrResponse
	}
	req, e := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s%d", endpoint, year), nil)
	if e != nil {
		return nil, e
	}
	req.Header.Set("Accept", "application/json")
	response, e := c.Do(req)
	if e != nil {
		return nil, e
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("holiday service HTTP %d", response.StatusCode)
	}
	data, e := io.ReadAll(io.LimitReader(response.Body, 1024*1024+1))
	if e != nil {
		return nil, e
	}
	if len(data) > 1024*1024 {
		return nil, ErrResponse
	}
	return parse(data, year)
}

type externalHoliday struct {
	Date     string   `json:"date"`
	Name     string   `json:"name"`
	Country  string   `json:"countryCode"`
	National bool     `json:"nationalHoliday"`
	Types    []string `json:"holidayTypes"`
}

func parse(data []byte, year int) ([]models.CalendarHoliday, error) {
	var source []externalHoliday
	if e := json.Unmarshal(data, &source); e != nil || len(source) == 0 || len(source) > 100 {
		return nil, ErrResponse
	}
	out := map[string]models.CalendarHoliday{}
	fixed := map[string]string{"01-01": "Nova godina", "01-02": "Nova godina", "01-07": "Božić", "02-15": "Sretenje — Dan državnosti", "02-16": "Sretenje — Dan državnosti", "05-01": "Praznik rada", "05-02": "Praznik rada", "11-11": "Dan primirja"}
	var easter time.Time
	for _, h := range source {
		d, e := time.Parse("2006-01-02", h.Date)
		if e != nil || d.Year() != year || h.Country != "RS" || strings.TrimSpace(h.Name) == "" || utf8.RuneCountInString(h.Name) > 200 {
			return nil, ErrResponse
		}
		public := false
		for _, typ := range h.Types {
			if typ == "Public" {
				public = true
			}
		}
		if !h.National || !public {
			continue
		}
		name := h.Name
		if translated, ok := fixed[d.Format("01-02")]; ok {
			name = translated
		}
		out[h.Date] = models.CalendarHoliday{Date: h.Date, Name: name, Origin: "Nager.Date"}
		if d.Weekday() == time.Sunday && strings.Contains(strings.ToLower(h.Name), "easter") {
			easter = d
		}
	}
	// Fail closed on incomplete data; do not make a partial import look complete.
	for suffix := range fixed {
		if _, ok := out[fmt.Sprintf("%04d-%s", year, suffix)]; !ok {
			return nil, ErrResponse
		}
	}
	if easter.IsZero() {
		return nil, ErrResponse
	}
	// Orthodox Easter holidays include Good Friday through Easter Monday.
	for offset := -2; offset <= 1; offset++ {
		key := easter.AddDate(0, 0, offset).Format("2006-01-02")
		origin := "Nager.Date"
		name := "Vaskršnji praznici"
		if existing, ok := out[key]; ok {
			if _, stateHoliday := fixed[key[5:]]; stateHoliday {
				name = existing.Name + " / " + name
			}
		}
		if _, ok := out[key]; !ok {
			origin = "Dopuna vaskršnjeg perioda"
		}
		out[key] = models.CalendarHoliday{Date: key, Name: name, Origin: origin}
	}
	// Serbian state holidays falling on Sunday transfer to the next working day.
	// Christmas and Easter are religious holidays, so this transfer does not apply.
	for _, suffix := range []string{"01-01", "01-02", "02-15", "02-16", "05-01", "05-02", "11-11"} {
		key := fmt.Sprintf("%04d-%s", year, suffix)
		d, _ := time.Parse("2006-01-02", key)
		if d.Weekday() != time.Sunday {
			continue
		}
		for d = d.AddDate(0, 0, 1); d.Weekday() == time.Saturday || d.Weekday() == time.Sunday; d = d.AddDate(0, 0, 1) {
		}
		// Skip the other statutory holiday date and Easter overlap, but do not move
		// again if the provider already supplied the observed date.
		for {
			suffix2 := d.Format("01-02")
			_, base := fixed[suffix2]
			isEaster := !d.Before(easter.AddDate(0, 0, -2)) && !d.After(easter.AddDate(0, 0, 1))
			if !base && !isEaster && d.Weekday() != time.Saturday && d.Weekday() != time.Sunday {
				break
			}
			d = d.AddDate(0, 0, 1)
		}
		observed := d.Format("2006-01-02")
		if _, ok := out[observed]; !ok {
			out[observed] = models.CalendarHoliday{Date: observed, Name: fixed[suffix] + " — preneti dan", Origin: "Pravilo prenosa državnog praznika"}
		}
	}
	result := make([]models.CalendarHoliday, 0, len(out))
	for _, h := range out {
		result = append(result, h)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Date < result[j].Date })
	return result, nil
}
