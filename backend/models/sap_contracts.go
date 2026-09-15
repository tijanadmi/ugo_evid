package models

import "time"

type SapDobavljac struct {
	ID        int    `json:"id"`
	Sifra     string `json:"sifra"`
	Naziv     string `json:"naziv"`
	WebPortal string `json:"web_portal"`
}
type SapUgovor struct {
	ID             int          `json:"id"`
	Godina         string       `json:"godina"`
	BrUgovor       string       `json:"br_ugovor"`
	UgovorDMS      string       `json:"ugovor_dms"`
	JN             string       `json:"jn"`
	BrPozPlana     string       `json:"br_poz_plana"`
	PredmetUgovora string       `json:"predmet_ugovora"`
	IdSapDobavljac int          `json:"id_sap_dobavljac"`
	OtvorenUG      string       `json:"otvoren_ug"`
	ZZN            string       `json:"zzn"`
	PocetakUG      time.Time    `json:"pocetak_ug"`
	KrajUG         time.Time    `json:"kraj_ug"`
	VrednostUG     float64      `json:"vrednost_ug"`
	ValutaUG       string       `json:"valuta_ug"`
	KursUG         float64      `json:"kurs_ug"`
	KomGrupa       string       `json:"kom_grupa"`
	MBrKomerc      string       `json:"m_br_komerc"`
	NazKomerc      string       `json:"naz_komerc"`
	GrNab          string       `json:"gr_nab"`
	NazivGrPlan    string       `json:"naziv_gr_plan"`
	VrsPred        string       `json:"vrs_pred"`
	Dobavljac      SapDobavljac `json:"dobavljac"`
	OdgLica        []SapOdglica `json:"odg_lica"`
}

type SapOdglica struct {
	ID           int    `json:"id"`
	IdSapUgovori int    `json:"id_sap_ugovori"`
	OdgZap       string `json:"odg_zap"`
	NazivOdgZap  string `json:"naziv_odg_zap"`
	RBr          int    `json:"r_br"`
}
