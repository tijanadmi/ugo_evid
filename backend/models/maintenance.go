package models

type ContractType struct {
	ID   int    `json:"id"`
	Code string `json:"sifra"`
	Name string `json:"naziv"`
}
type Maintenance struct {
	ID                  int    `json:"id"`
	Signed              string `json:"datum_potpisa" binding:"required,datetime=2006-01-02"`
	Ends                string `json:"datum_kraja" binding:"required,datetime=2006-01-02"`
	Development         bool   `json:"ima_razvojnu_podrsku"`
	Months              *int   `json:"teh_period_meseci" binding:"omitempty,min=1,max=120"`
	DocumentType        int    `json:"id_teh_tip_doc" binding:"required,min=1"`
	TechnicalReminder   int    `json:"teh_podsetnik_dana" binding:"min=0,max=365"`
	DevelopmentReminder int    `json:"raz_podsetnik_dana" binding:"min=0,max=365"`
}
type ContractSettings struct {
	TypeID      int          `json:"id_tip_ugovora"`
	Version     int64        `json:"version,string"`
	Maintenance *Maintenance `json:"odrzavanje"`
}
