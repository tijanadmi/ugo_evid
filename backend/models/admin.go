package models

type RoleGrant struct {
	RoleID int    `json:"id_ugo_rola"`
	Scope  string `json:"opseg"`
	OrgID  int    `json:"id_ugo_org"`
}
type AdminUser struct {
	ID       int         `json:"id"`
	Username string      `json:"ad_sifra"`
	Name     string      `json:"ime"`
	Status   string      `json:"status"`
	Version  int64       `json:"version,string"`
	Grants   []RoleGrant `json:"roles"`
}
type AdminRole struct {
	ID   int    `json:"id"`
	Code string `json:"sifra"`
	Name string `json:"naziv"`
}
type AdminCatalog struct {
	Roles         []AdminRole          `json:"roles"`
	Organizations []OrganizationAccess `json:"organizations"`
}
type AdminNotification struct {
	OriginalName string `json:"originalni_naziv"`
	ContractID   int    `json:"id_ugo_evid"`
	Attempts     int    `json:"broj_pokusaja"`
	ID           int    `json:"id"`
	Kind         string `json:"vrsta"`
	EntityID     int    `json:"id_sloga"`
	Title        string `json:"naslov"`
	Message      string `json:"poruka"`
	Status       string `json:"status"`
	AssignedID   int    `json:"id_preuzeo"`
	AssignedName string `json:"preuzeo"`
	First        string `json:"prvi_put"`
	Last         string `json:"poslednji_put"`
	Count        int    `json:"broj_ponavljanja"`
	Resolved     string `json:"reseno"`
	Version      int64  `json:"version,string"`
}
type AdminAudit struct {
	ID       int    `json:"id"`
	UserID   int    `json:"id_korisnika"`
	Username string `json:"korisnik"`
	Action   string `json:"akcija"`
	Kind     string `json:"vrsta"`
	EntityID int    `json:"id_sloga"`
	Before   string `json:"pre"`
	After    string `json:"posle"`
	Time     string `json:"vreme"`
}
