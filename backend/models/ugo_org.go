package models

import (
	"time"
)

type UgoOrg struct {
	ID       int       `json:"id"`
	Sifra    string    `json:"sifra"`
	SifraCir string    `json:"sifra_cir,omitempty"`
	Naziv    string    `json:"naziv"`
	NazivCir string    `json:"naziv_cir,omitempty"`
	Status   string    `json:"status,omitempty"`
	DatPri   time.Time `json:"datpri,omitempty"`
	DatIzm   time.Time `json:"datizm,omitempty"`
}
