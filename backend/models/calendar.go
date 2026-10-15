package models

type CalendarDay struct {
	Source      string `json:"izvor"`
	Date        string `json:"datum"`
	Work        string `json:"radni"`
	Approved    string `json:"potvrdjen"`
	Description string `json:"opis"`
}
type CalendarYear struct {
	Items   []CalendarDay `json:"items"`
	Version string        `json:"version"`
}
type CalendarChange struct {
	Holidays    []CalendarHoliday `json:"praznici"`
	Action      string            `json:"action"`
	Version     string            `json:"version"`
	Start       string            `json:"od"`
	End         string            `json:"do"`
	Work        string            `json:"radni"`
	Description string            `json:"opis"`
}

type CalendarHoliday struct {
	ReplaceManual bool   `json:"zameni_rucno,omitempty"`
	Date          string `json:"datum"`
	Name          string `json:"naziv"`
	Origin        string `json:"poreklo,omitempty"`
}
