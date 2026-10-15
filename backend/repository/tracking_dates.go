package repository

import (
	"errors"
	"fmt"
	"github.com/tijanadmi/ugo_evid/models"
	"time"
)

var ErrPeriodDelete = errors.New("Period nije obrisan: samo nepromenjen period sa statusom Planiran moze da se obrise. Osvezite podatke.")

var ErrTrackingInput = errors.New("Proverite datume, status, dokument i ugovorene casove. Periodi ne smeju da se preklapaju.")
var ErrTrackingCalendar = errors.New("Otvorite Radni kalendar, dopunite potrebne datume i potvrdite godinu.")
var ErrTrackingFunds = errors.New("Zahtev prekoracuje preostale casove ove vrste rada ili fond nije ugovoren.")
var ErrTrackingDependencies = errors.New("Postojeci periodi ili zahtevi nisu uskladjeni sa novim datumima ili fondom casova.")

func date(s string) (time.Time, error) { return time.Parse("2006-01-02", s) }

// Always add to the original anchor, avoiding Jan31 -> Feb28 -> Mar28 drift.
func anchoredMonth(start time.Time, months int) time.Time {
	first := time.Date(start.Year(), start.Month()+time.Month(months), 1, 0, 0, 0, 0, time.UTC)
	last := first.AddDate(0, 1, -1).Day()
	day := start.Day()
	if day > last {
		day = last
	}
	return first.AddDate(0, 0, day-1)
}
func GeneratePeriods(start, end string, months, doc int, mode string) ([]models.TrackingItem, error) {
	a, e := date(start)
	if e != nil {
		return nil, ErrTrackingInput
	}
	b, e := date(end)
	if e != nil || a.After(b) || months < 1 || months > 120 || (mode != "calendar" && mode != "anchor") {
		return nil, ErrTrackingInput
	}
	if mode == "calendar" && months != 1 && months != 3 && months != 6 && months != 12 {
		return nil, ErrTrackingInput
	}
	out := []models.TrackingItem{}
	cursor := a
	for n := 1; !cursor.After(b); n++ {
		if n > 1200 {
			return nil, ErrTrackingInput
		}
		next := anchoredMonth(a, n*months)
		if mode == "calendar" {
			m := ((int(cursor.Month())-1)/months+1)*months + 1
			next = time.Date(cursor.Year(), time.Month(m), 1, 0, 0, 0, 0, time.UTC)
		}
		last := next.AddDate(0, 0, -1)
		if last.After(b) {
			last = b
		}
		out = append(out, models.TrackingItem{Start: cursor.Format("2006-01-02"), End: last.Format("2006-01-02"), DocumentType: doc, Status: "PLANIRAN"})
		cursor = last.AddDate(0, 0, 1)
	}
	return out, nil
}

func requestDeadline(start string, days int, calendar map[string]bool) (string, error) {
	d, e := date(start)
	if e != nil || days < 1 || days > 3660 {
		return "", ErrTrackingInput
	}
	for n := 0; n < 20000; n++ {
		d = d.AddDate(0, 0, 1)
		key := d.Format("2006-01-02")
		work, ok := calendar[key]
		if !ok {
			return "", fmt.Errorf("Kalendar za %d. godinu nije pripremljen ili potvrdjen (datum %s). %w", d.Year(), key, ErrTrackingCalendar)
		}
		if work {
			days--
		}
		if days == 0 {
			return key, nil
		}
	}
	return "", ErrTrackingCalendar
}
