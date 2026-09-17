# UGO evidencija: Oracle i AD

Backend koristi `database/sql` i `github.com/sijms/go-ora/v2`, kao primer
`ddn_rdc`. Nije potreban Oracle Instant Client. SQL upiti zahtevaju Oracle 12c+
(paginacija `OFFSET ... FETCH NEXT`).

## Podesavanje i pokretanje

U `app.env` zamenite probne vrednosti ili postavite istoimene promenljive
okruzenja (one imaju prednost):

```dotenv
DB_DRIVER=oracle
DB_SOURCE=oracle://APP_USER:URL_ENCODED_PASSWORD@ORACLE_HOST:1521/SERVICE_NAME
LDAP_SERVERS=dc1.example.local,dc2.example.local
LDAP_DOMAIN=example.local
LDAP_PORT=389
LDAP_TIMEOUT=5s
ACTIVE_USER_STATUS=A
TOKEN_SYMMETRIC_KEY=<nasumicni kljuc od tacno 32 bajta>
ACCESS_TOKEN_DURATION=15m
REFRESH_TOKEN_DURATION=24h
HTTP_SERVER_ADDRESS=0.0.0.0:8080
```

`DB_SOURCE` sadrzi Oracle service name. Specijalne znakove u korisnickom imenu
i lozinci URL-kodirati. Upiti koriste eksplicitnu semu `TED`. DB nalogu treba
SELECT nad `TED.SAP_UGOVORI`, `TED.SAP_DOBAVLJACI` i `TED.SAP_ODGLICA`.
Nad UGO tabelama potrebne su privilegije za operacije koje aplikacija koristi.
Backend nema INSERT/UPDATE/DELETE operacije nad SAP tabelama.
Lozinka u ovom URL-u pripada tehnickom Oracle nalogu, a AD lozinka se salje
samo pri prijavi korisnika.

LDAP prijava koristi identican `util/ldap.go` kao projekat `ddn_rdc`: `ldap://`,
port 389, Bind sa `username@LDAP_DOMAIN` i pokusaj sledeceg servera ako prethodni
nije dostupan. LDAP_SERVERS je lista imena ili IP adresa bez protokola i porta.
Kopirati LDAP_SERVERS i LDAP_DOMAIN iz konfiguracije koja radi u `ddn_rdc`.

Iz foldera `backend`:

```powershell
go mod download
go test ./...
go run .
```

Pri pokretanju se izvrsava Oracle Ping sa vremenskim ogranicenjem. Aplikacija
ne kreira tabele i ne uvozi podatke automatski.

## AD korisnici

Korisnik mora unapred postojati u `ugo_kor` i imati `status='A'` (ili vrednost
iz ACTIVE_USER_STATUS). `ad_sifra` je jedini korisnicki identitet u bazi.
Prijava prihvata tacnu `ad_sifra`,
bez razlikovanja velikih i malih slova. Ako postoji vise odgovarajucih zapisa,
prijava se odbija. U `AD_SIFRA` upisati kratko AD korisnicko ime bez domena;
LDAP kod mu dodaje `@LDAP_DOMAIN`, identicno projektu `ddn_rdc`.

Primer provisioniranja u postojecoj semi sa automatskim ID-em:

```sql
INSERT INTO TED.UGO_KOR (ad_sifra, ime, status, datpri, datizm)
VALUES ('ime.prezime', 'Ime Prezime', 'A', SYSDATE, SYSDATE);
COMMIT;
```

`TED.UGO_KOR` nema kolone `SIFRA` ni `LOZINKA`. AD lozinka se ne cuva u bazi.
JSON polje `username` i identitet u tokenima sadrze `AD_SIFRA`.
Javno kreiranje naloga (`POST /users`) je uklonjeno; pristup
aplikaciji dodeljuje administrator kroz bazu.

```http
POST /users/login
Content-Type: application/json

{"username":"ime.prezime","password":"AD lozinka"}
```

Odgovor zadrzava access_token, refresh_token i podatke korisnika. Poslovne rute
zahtevaju `Authorization: Bearer <access_token>`. Obnova tokena proverava da je
korisnik i dalje aktivan u aplikacionoj bazi. Vec izdat access token vazi do
isteka. Prijava je provera AD korisnickog imena i lozinke, a ne Windows SSO.

`ugo_kor_role` predstavlja veze korisnika sa organizacijama; ne sadrzi nazive
bezbednosnih uloga iz ddn_rdc. Postojeci API zadrzava zajednicku ulogu `user`;
ogranicavanje podataka po organizacijama nije uvedeno ovom konverzijom.

## Sema i postojeci podaci

Merodavan model je dostavljeni Oracle DDL iz seme `TED`, a ne stare PostgreSQL
migracije. `oracle/schema.sql` je citljiva referenca devet tabela, ogranicenja
i trigera iz tog modela, bez fizickih storage podesavanja. Nije migracija za
izvrsavanje nad bazom: definicije i pocetne vrednosti sekvenci nisu dostavljene.
Postojeci trigeri koriste `TED.<TABELA>_SEQ` i popunjavaju ID; INSERT upiti
izostavljaju ID i preuzimaju ga kroz `RETURNING ... INTO`.
View `sap_ugovori_v` nije potreban endpointima niti je definisan u dostavljenom DDL-u.

`UGO_DOB_LICA` nema `ID_UGO_ORG`: uklonjeni su spoj sa organizacijom,
filtriranje po organizaciji i JSON polje `ugo_org` iz modela tog lica.
POST/PUT za lice vise ne zahtevaju `id_ugo_org`. Organizacija ostaje deo
`UGO_EVID` i `UGO_KOR_ROLE`, gde postoji u bazi.
Datumi su nullable Oracle `DATE`; `DATZM` se koristi kod lica i njihovih rola,
a `DATIZM` kod ostalih UGO tabela.

PUT rute za lice, rolu lica i evidenciju koriste ID u putanji:
`/ugo_dob_lica/:id`, `/ugo_dob_lica_rola/:id`, `/ugo_evid/:id`.
SAP ugovori se citaju preko `GET /sapugovori`; dobavljaci i odgovorna lica
citaju se u povezanim SELECT upitima. SAP podaci se ne menjaju kroz API.

Originalni fajlovi u `../migrations` i `../tdi_evid_seed_data` su PostgreSQL
skripte sa starim modelom i nisu uskladjene sa stvarnom TED semom. Ako podaci vec postoje na
Oracle-u, nema potrebe za ponovnim uvozom. Kod novog uvoza sacuvati ID-eve i
strane kljuceve, prilagoditi INSERT/date sintaksu i nakon uvoza uskladiti
identity/sequence sa najvecim postojecim ID-em.

Opcioni strani kljucevi sa ID=0 upisuju se kao NULL. SQL NULL vrednosti pri
citanju mapiraju se na postojece Go nulte vrednosti (prazan tekst, 0 ili nulti
datum), radi kompatibilnosti JSON modela.

## Provera

### Pregled otvorenih i zatvorenih ugovora

Oba endpointa citaju postojeci `TED.UGO_EVID_PROSIRENI_V` i zahtevaju
`Authorization: Bearer <access_token>`:

```http
GET /ugo_evid/otvoreni?page_id=1&page_size=20
GET /ugo_evid/zatvoreni?page_id=1&page_size=20
GET /ugo_evid/otvoreni?id_ugo_org=3&page_id=1&page_size=20
GET /ugo_evid/zatvoreni?id_ugo_org=3&page_id=1&page_size=20
```

- Otvoreni: `OTVOREN_UG = 'X'`.
- Zatvoreni: `OTVOREN_UG IS NULL`.
- `OTVOREN_UG` ima samo vrednosti `X` ili NULL; liste se ne preklapaju.
- Opcioni `id_ugo_org`: pozitivan ceo broj filtrira po organizacionoj jedinici.
  Izostavljen parametar ili `id_ugo_org=0` prikazuje sve jedinice. Negativne
  vrednosti i tekst (ukljucujuci `%`) vracaju HTTP 400.
  Isti filter se primenjuje na `items` i `total`.
- `page_id` je najmanje 1 (podrazumevano 1), `page_size` je 1–100 (podrazumevano 20).
- Redosled: `ID_UGO_EVID DESC`. Odgovor je `{"total":123,"items":[...]}`.
- Svaka stavka sadrzi sve 42 kolone pogleda, sa malim slovima u JSON nazivima.
  `naziv` je naziv dobavljaca, a `sluzba` sifra organizacije.
- NULL tekst je prazan string; nullable datumi, dobavljac ID i iznosi ostaju JSON `null`.
  Prazna stranica vraca `items: []`, uz ukupan broj odgovarajucih zapisa.
- Bez filtera pregled obuhvata evidencije iz pogleda za sve organizacije. Jedan ugovor moze
  imati vise evidencija; `total` broji evidencije, ne razlicite SAP ugovore.

DB nalogu potreban je SELECT nad `TED.UGO_EVID_PROSIRENI_V`.
Backend ne kreira pogled automatski. Endpointi su samo za citanje.

Ako prijava vrati `AD servis trenutno nije dostupan`, proveriti iste servere,
port i domen koji se koriste u `ddn_rdc`. LDAP funkcija vraca istu genericku
gresku kao referentni projekat. Sa racunara na kome radi backend proveriti:

```powershell
Test-NetConnection dc1.example.local -Port 389
```

Zameniti primer adresom AD servera. Promenljive okruzenja imaju prednost nad
`app.env`. Nakon promene konfiguracije restartovati backend.

Unit testovi pokrivaju AD tok prijave bez mreze, zastitu ruta, ucitavanje
konfiguracije i citanje Oracle NULL vrednosti. Za proveru konkretne Oracle
seme, RETURNING parametara, stvarnih AD naloga potrebno je
pokrenuti aplikaciju u ciljnom okruzenju i proveriti listanje i CRUD.

Dokumentacija biblioteka:
- https://github.com/sijms/go-ora
- https://pkg.go.dev/github.com/go-ldap/ldap/v3
