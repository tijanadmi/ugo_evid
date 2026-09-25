package models

type Partner struct {
	ID             int                 `json:"id"`
	Sifra          string              `json:"sifra"`
	Naziv          string              `json:"naziv"`
	Adresa         string              `json:"adresa"`
	Grad           string              `json:"grad"`
	WebPortal      string              `json:"web_portal"`
	LicaDobavljaca []UgoDobLiceKontakt `json:"lica_dobavljaca"`
}
