package models

import "time"

type UgoDobLicaRola struct {
	ID     int       `json:"id"`
	Naziv  string    `json:"naziv"`
	Status string    `json:"status,omitempty"`
	DatPri time.Time `json:"datpri,omitempty"`
	DatZm  time.Time `json:"datzm,omitempty"`
}
