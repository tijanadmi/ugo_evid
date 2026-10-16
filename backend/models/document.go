package models

import "time"

type DocumentType struct {
	ID   int    `json:"id"`
	Code string `json:"sifra"`
	Name string `json:"naziv"`
}
type DocumentMetadata struct {
	RequestID int    `json:"id_raz_zahtev" binding:"min=0"`
	TypeID    int    `json:"id_tip_doc" binding:"required,min=1"`
	Name      string `json:"naziv" binding:"required,max=200"`
	Number    string `json:"broj_dokumenta" binding:"max=100"`
	Date      string `json:"datum_dokumenta" binding:"omitempty,datetime=2006-01-02"`
	Note      string `json:"napomena" binding:"max=2000"`
}
type Document struct {
	DocumentMetadata
	ID           int        `json:"id"`
	ContractID   int        `json:"id_ugo_evid"`
	TypeName     string     `json:"tip_naziv"`
	OriginalName string     `json:"originalni_naziv"`
	Path         string     `json:"-"`
	MIME         string     `json:"mime_tip"`
	Size         int64      `json:"velicina_bajt"`
	SHA256       string     `json:"sha256"`
	Status       string     `json:"status"`
	UserID       int        `json:"id_korisnika_unosa"`
	Created      time.Time  `json:"datpri"`
	Updated      *time.Time `json:"datizm,omitempty"`
	Version      int64      `json:"version,string"`
}
