package models

import "time"

type UgoEvid struct {
	ID         int        `json:"id"`
	SapUgovor  SapUgovor  `json:"sap_ugovor"`
	UgoOrg     UgoOrg     `json:"ugo_org"`
	Ime        string     `json:"ime"`
	Telefon    string     `json:"telefon,omitempty"`
	Email      string     `json:"email,omitempty"`
	Status     string     `json:"status,omitempty"`
	DatPri     time.Time  `json:"datpri,omitempty"`
	DatIzm     time.Time  `json:"datizm,omitempty"`
	UgoDobLice UgoDobLice `json:"ugo_dob_lice,omitempty"`
}
