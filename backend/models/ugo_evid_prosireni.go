package models

import "time"

// UgoEvidProsireni maps the columns of TED.UGO_EVID_PROSIRENI_V.
type UgoEvidProsireni struct {
	IDUgoEvid      int        `json:"id_ugo_evid"`
	IDUgoOrg       int        `json:"id_ugo_org"`
	IDSapUgovor    int        `json:"id_sap_ugovor"`
	IDSapDobavljac *int       `json:"id_sap_dobavljac"`
	Godina         string     `json:"godina"`
	BrUgovor       string     `json:"br_ugovor"`
	UgovorDMS      string     `json:"ugovor_dms"`
	JN             string     `json:"jn"`
	BrPozPlana     string     `json:"br_poz_plana"`
	PredmetUgovora string     `json:"predmet_ugovora"`
	Naziv          string     `json:"naziv"`
	OtvorenUG      string     `json:"otvoren_ug"`
	ZZN            string     `json:"zzn"`
	PocetakUG      *time.Time `json:"pocetak_ug"`
	KrajUG         *time.Time `json:"kraj_ug"`
	VrednostUG     *float64   `json:"vrednost_ug"`
	ValutaUG       string     `json:"valuta_ug"`
	KursUG         *float64   `json:"kurs_ug"`
	KomGrupa       string     `json:"kom_grupa"`
	MBrKomerc      string     `json:"m_br_komerc"`
	NazKomerc      string     `json:"naz_komerc"`
	NazivGrPlan    string     `json:"naziv_gr_plan"`
	VrsPred        string     `json:"vrs_pred"`
	Sluzba         string     `json:"sluzba"`
	OdgZap1        string     `json:"odg_zap_1"`
	NazivOdgZap1   string     `json:"naziv_odg_zap_1"`
	OdgZap2        string     `json:"odg_zap_2"`
	NazivOdgZap2   string     `json:"naziv_odg_zap_2"`
	OdgZap3        string     `json:"odg_zap_3"`
	NazivOdgZap3   string     `json:"naziv_odg_zap_3"`
	OdgZap4        string     `json:"odg_zap_4"`
	NazivOdgZap4   string     `json:"naziv_odg_zap_4"`
	OdgZap5        string     `json:"odg_zap_5"`
	NazivOdgZap5   string     `json:"naziv_odg_zap_5"`
	OdgZap6        string     `json:"odg_zap_6"`
	NazivOdgZap6   string     `json:"naziv_odg_zap_6"`
	Ime            string     `json:"ime"`
	Telefon        string     `json:"telefon"`
	Email          string     `json:"email"`
	Status         string     `json:"status"`
	DatPri         *time.Time `json:"datpri"`
	DatIzm         *time.Time `json:"datizm"`
}
