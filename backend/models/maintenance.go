package models

type ContractType struct {
	ID   int    `json:"id"`
	Code string `json:"sifra"`
	Name string `json:"naziv"`
}
type WorkType struct {
	ID   int    `json:"id"`
	Code string `json:"sifra"`
	Name string `json:"naziv"`
}
type DevelopmentFund struct {
	TypeID int `json:"id_vrsta_rada" binding:"required,min=1"`
	Hours  int `json:"ugovoreno_casova" binding:"min=0,max=2147483647"`
}
type Maintenance struct {
	Start               string            `json:"pocetak_odrzavanja" binding:"omitempty,datetime=2006-01-02"`
	Version             int64             `json:"version,string"`
	Token               string            `json:"lock_token,omitempty"`
	Funds               []DevelopmentFund `json:"fondovi" binding:"max=3,dive"`
	ID                  int               `json:"id"`
	Signed              string            `json:"datum_potpisa"`
	Ends                string            `json:"datum_kraja"`
	Development         bool              `json:"ima_razvojnu_podrsku"`
	Months              *int              `json:"teh_period_meseci" binding:"omitempty,min=1,max=120"`
	DocumentType        int               `json:"id_teh_tip_doc" binding:"required,min=1"`
	TechnicalReminder   int               `json:"teh_podsetnik_dana" binding:"min=0,max=365"`
	DevelopmentReminder int               `json:"raz_podsetnik_dana" binding:"min=0,max=365"`
}
type ContractSettings struct {
	SAPStart    string       `json:"sap_pocetak"`
	SAPEnd      string       `json:"sap_kraj"`
	WorkTypes   []WorkType   `json:"vrste_rada"`
	TypeID      int          `json:"id_tip_ugovora"`
	Version     int64        `json:"version,string"`
	Maintenance *Maintenance `json:"odrzavanje"`
}
