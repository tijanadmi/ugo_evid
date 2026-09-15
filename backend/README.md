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
LDAP_PORT=636
LDAP_SECURITY=ldaps
LDAP_TIMEOUT=5s
LDAP_CA_CERT=
ACTIVE_USER_STATUS=A
TOKEN_SYMMETRIC_KEY=<nasumicni kljuc od tacno 32 bajta>
ACCESS_TOKEN_DURATION=15m
REFRESH_TOKEN_DURATION=24h
HTTP_SERVER_ADDRESS=0.0.0.0:8080
```

`DB_SOURCE` sadrzi Oracle service name. Specijalne znakove u korisnickom imenu
i lozinci URL-kodirati. DB nalog treba da vidi postojece tabele u svojoj semi
ili kroz sinonime i ima odgovarajuce SELECT/INSERT/UPDATE/DELETE privilegije.
Lozinka u ovom URL-u pripada tehnickom Oracle nalogu, a AD lozinka se salje
samo pri prijavi korisnika.

LDAP_SERVERS je lista DNS imena bez protokola i porta. Za StartTLS postaviti
`LDAP_SECURITY=starttls` i `LDAP_PORT=389`. Podrazumevano se koriste sistemski
CA sertifikati; `LDAP_CA_CERT` moze biti putanja do dodatnog PEM CA lanca.
Provera sertifikata ostaje ukljucena. API objaviti preko HTTPS-a.

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
iz ACTIVE_USER_STATUS). `sifra` je aplikaciono korisnicko ime, a `ad_sifra`
identitet koji se proverava na AD-u. Prijava prihvata sifra ili tacnu ad_sifra,
bez razlikovanja velikih i malih slova. Ako postoji vise odgovarajucih zapisa,
prijava se odbija. Kratkoj ad_sifra dodaje se LDAP_DOMAIN; UPN
(`ime@domen`) i `DOMEN\ime` koriste se kako su upisani.

Primer provisioniranja u postojecoj semi sa automatskim ID-em:

```sql
INSERT INTO ugo_kor (ad_sifra, sifra, lozinka, ime, status, datpri, datizm)
VALUES ('ime.prezime', 'ime.prezime', RAWTOHEX(SYS_GUID()),
        'Ime Prezime', 'A', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
COMMIT;
```

Kolona `lozinka` iz originalne seme je NOT NULL UNIQUE, pa se upisuje jedinstvena
nasumicna vrednost. Ne upisivati AD lozinku. Kolona se ne cita pri prijavi i ne
vraca kroz JSON. Javno kreiranje naloga (`POST /users`) je uklonjeno; pristup
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

Nazivi tabela i kolona odgovaraju fajlovima u `../migrations`, ukljucujuci
razliku `datzm` / `datizm`. SELECT upiti zadrzavaju postojece spojeve tabela;
view `sap_ugovori_v` nije potreban tim endpointima.

`oracle/schema.sql` je opciona Oracle varijanta svih devet tabela i view-a,
iskljucivo za novu praznu semu. Postojecu bazu ne menjati tim fajlom.
ID kolone moraju imati identity ili postojeci sequence/trigger koji popunjava
ID pri INSERT-u; backend ne koristi `MAX(id)+1`.

Originalni fajlovi u `../migrations` i `../tdi_evid_seed_data` su PostgreSQL
skripte i nisu automatski izvrsivi na Oracle-u. Ako podaci vec postoje na
Oracle-u, nema potrebe za ponovnim uvozom. Kod novog uvoza sacuvati ID-eve i
strane kljuceve, prilagoditi INSERT/date sintaksu i nakon uvoza uskladiti
identity/sequence sa najvecim postojecim ID-em.

Opcioni strani kljucevi sa ID=0 upisuju se kao NULL. SQL NULL vrednosti pri
citanju mapiraju se na postojece Go nulte vrednosti (prazan tekst, 0 ili nulti
datum), radi kompatibilnosti JSON modela.

## Provera

Unit testovi pokrivaju AD tok prijave bez mreze, zastitu ruta, ucitavanje
konfiguracije i citanje Oracle NULL vrednosti. Za proveru konkretne Oracle
seme, RETURNING parametara, LDAP sertifikata i stvarnih AD naloga potrebno je
pokrenuti aplikaciju u ciljnom okruzenju i proveriti listanje i CRUD.

Dokumentacija biblioteka:
- https://github.com/sijms/go-ora
- https://pkg.go.dev/github.com/go-ldap/ldap/v3
