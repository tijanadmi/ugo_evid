package models

import (
	"time"
)

type UgoDobLice struct {
	ID             int            `json:"id"`
	SapDobavljac   SapDobavljac   `json:"sap_dobavljac"`
	Ime            string         `json:"ime"`
	RadnoMesto     string         `json:"radno_mesto,omitempty"`
	Telefon        string         `json:"telefon,omitempty"`
	Email          string         `json:"email,omitempty"`
	UgoDobLicaRola UgoDobLicaRola `json:"ugo_dob_lica_rola,omitempty"`
	Status         string         `json:"status,omitempty"`
	DatPri         time.Time      `json:"datpri,omitempty"`
	DatZm          time.Time      `json:"datzm,omitempty"`
}
