package models

type ContractPerson struct {
	ID           int    `json:"id"`
	PersonID     int    `json:"id_ugo_dob_lica"`
	RoleID       int    `json:"id_ugo_dob_lica_rola"`
	Role         string `json:"rola"`
	Name         string `json:"ime"`
	Phone        string `json:"telefon"`
	Email        string `json:"email"`
	Status       string `json:"status"`
	PersonStatus string `json:"status_lica"`
	Version      int64  `json:"version,string"`
}
type ContractPersonChange struct {
	ID              int   `json:"id"`
	Version         int64 `json:"version,string"`
	PersonID        int   `json:"id_ugo_dob_lica"`
	RoleID          int   `json:"id_ugo_dob_lica_rola"`
	PreviousSLM     int   `json:"prethodni_slm"`
	PreviousVersion int64 `json:"prethodna_verzija,string"`
}
type ContactEngagement struct {
	ContractID int    `json:"id_ugo_evid"`
	Number     string `json:"br_ugovor"`
	Subject    string `json:"predmet_ugovora"`
	Role       string `json:"rola"`
}
