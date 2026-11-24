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
type TechnicalReport struct {
	ID         int    `json:"id"`
	ContractID int    `json:"id_ugo_evid"`
	PeriodID   int    `json:"id_teh_period"`
	TemplateID int    `json:"id_template"`
	Quarter    int    `json:"broj_kvartala"`
	Start      string `json:"period_od"`
	End        string `json:"period_do"`
	Signed     string `json:"datum_potpisa"`
	DOCX       string `json:"-"`
	PDF        string `json:"-"`
	HasPDF     bool   `json:"ima_pdf"`
	Created    string `json:"kreiran"`
}
