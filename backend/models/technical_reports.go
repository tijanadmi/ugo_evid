package models

type ReportTemplate struct {
	UsesQuarter bool   `json:"koristi_kvartal"`
	ID          int    `json:"id"`
	ContractID  int    `json:"id_ugo_evid"`
	TypeID      int    `json:"id_tip_doc"`
	Version     int    `json:"verzija"`
	Name        string `json:"naziv"`
	Path        string `json:"-"`
}

// TechnicalReport contains only the current in-memory generation snapshot.
type TechnicalReport struct {
	ContractID int
	PeriodID   int
	TemplateID int
	Quarter    int
	Start      string
	End        string
	Signed     string
}
