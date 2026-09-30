package models

// ContractChoice is the small projection needed by the SAP contract picker.
type ContractChoice struct {
	ID         int    `json:"id"`
	SupplierID int    `json:"id_sap_dobavljac"`
	Number     string `json:"br_ugovor"`
	Year       string `json:"godina"`
	Subject    string `json:"predmet_ugovora"`
	Supplier   string `json:"dobavljac"`
	Open       string `json:"otvoren_ug"`
}

type ContractPreparation struct {
	Contract ContractChoice `json:"contract"`
	Contact  *UgoDobLice    `json:"contact"`
}

type NewContractContact struct {
	Ime        string `json:"ime"`
	RadnoMesto string `json:"radno_mesto"`
	Telefon    string `json:"telefon"`
	Email      string `json:"email"`
}

type RegisterContract struct {
	ContractID     int                 `json:"id_sap_ugovor" binding:"required,min=1"`
	ContactID      int                 `json:"id_ugo_dob_lica" binding:"min=0"`
	ContactVersion int64               `json:"contact_version,string"`
	Contact        *NewContractContact `json:"novo_lice"`
}
