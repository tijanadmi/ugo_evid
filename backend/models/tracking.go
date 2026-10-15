package models

// A common API envelope; persistence uses separate period and request tables.
type TrackingItem struct {
	ID              int    `json:"id"`
	Version         int64  `json:"version,string"`
	Start           string `json:"period_od"`
	End             string `json:"period_do"`
	DocumentType    int    `json:"id_tip_doc"`
	WorkType        int    `json:"id_vrsta_rada"`
	Number          string `json:"broj_zahteva"`
	Requested       string `json:"datum_zahteva"`
	Hours           int    `json:"broj_casova"`
	Days            int    `json:"rok_radnih_dana"`
	Deadline        string `json:"rok"`
	Signed          string `json:"datum_potpisa"`
	Document        int    `json:"id_ugo_doc"`
	RequestDocument int    `json:"id_zahtev_doc"`
	Status          string `json:"status"`
	Note            string `json:"napomena"`
}
type TrackingChange struct {
	Token string         `json:"lock_token"`
	Items []TrackingItem `json:"items"`
}
type PeriodPreview struct {
	Mode string `json:"mode"`
}
type FundBalance struct {
	WorkType   int    `json:"id_vrsta_rada"`
	Name       string `json:"naziv"`
	Contracted int    `json:"ugovoreno"`
	Used       int    `json:"zauzeto"`
	Remaining  int    `json:"preostalo"`
}
type TrackingData struct {
	Periods  []TrackingItem `json:"periodi"`
	Requests []TrackingItem `json:"zahtevi"`
	Funds    []FundBalance  `json:"fondovi"`
}
type Reminder struct {
	ContractID int    `json:"id_ugo_evid"`
	Kind       string `json:"vrsta"`
	ID         int    `json:"id"`
	Label      string `json:"opis"`
	Due        string `json:"rok"`
	Overdue    bool   `json:"kasni"`
}
