# Evidencija ugovora — frontend

React/Vite aplikacija u JavaScript/JSX-u, po uzoru na `C:\react_workspace\tis_frontend`:
portal sa karticama, tamnoplavi bočni meni i zajedničko zaglavlje. Za sada je jedna
aplikacija, pa portal i pregled koriste isti projekat i jednu prijavu.

## Struktura

Organizacija koda prati `pgi_frontend`: `pages` su ulazne stranice,
`features/contracts` sadrži tabelu, red, odgovorna lica, detalje i `useContracts`,
`features/organizations` sadrži `useOrganizations`, a `features/authentication`
formu prijave. Zajedničke komponente su u `ui`, HTTP pozivi u `services`,
sesija u `context`, stilovi u `styles` i formatiranje u `utils`.

`App.jsx` postavlja rute i React Query provider. React Query upravlja učitavanjem,
greškama, ponovnim učitavanjem i kešom ugovora i organizacija. Ključ ugovora
uključuje korisnika, status, organizaciju, stranicu i veličinu stranice.
Keš se briše pri odjavi/promeni korisnika.

Kolona **Odgovorna lica** prikazuje popunjene parove `odg_zap_1` / `naziv_odg_zap_1`
do `odg_zap_6` / `naziv_odg_zap_6`, po jedno lice u redu. Vrednost je u detaljima.

## Pokretanje

Potreban je Node.js 20 i npm. Iz foldera `frontend`:

```powershell
npm ci
Copy-Item .env.example .env
npm run dev
```

Otvori `http://localhost:5173`. U `.env` postavi adresu Go backenda:

```dotenv
VITE_BACKEND_URL=http://localhost:8080
VITE_API_BASE_URL=/api
```

Na poslovnoj mreži zameni `localhost:8080` stvarnom adresom. Vite prosleđuje
`/api` zahteve backendu i uklanja prefiks. Restartuj Vite nakon promene `.env`.
U frontend konfiguraciju ne unositi Oracle lozinke niti ključ za tokene.

## Ekrani i API

- `/ugovori/partneri` — **Moji partneri**, preko `GET /moji_partneri`.
  Tabela prikazuje naziv, adresu i grad jedinstvenih dobavljača korisnikove
  organizacije. Klik na naziv ili strelicu otvara detalje sa kontakt licima
  i istaknutim Service Level Manager-om. Filter po delu naziva primenjuje se
  na backendu; dugme Pretraži vraća na prvu stranicu, Poništi uklanja filter.
  Filter i paginacija čuvaju se u URL-u. Organizaciju određuje backend iz naloga.
  Kontakt iz pojedinačnog ugovora ostaje na stranici detalja tog ugovora.

- `/login` — AD prijava preko `POST /users/login`.
- `/` — portal sa aplikacijom **Evidencija ugovora**.
- `/ugovori` — preusmeravanje na otvorene ugovore.
- `/ugovori/otvoreni` — `GET /ugo_evid/otvoreni`.
- `/ugovori/zatvoreni` — `GET /ugo_evid/zatvoreni`.

Pregledi šalju `page_id`, `page_size` i `id_ugo_org`. Izbor **Sve organizacione
jedinice** šalje `id_ugo_org=0`; ostale vrednosti dolaze iz paginiranog
`GET /ugo_org`. Promena filtera ili veličine stranice vraća pregled na prvu
stranicu. Filter i stranica čuvaju se u URL-u.

Tabela prikazuje glavne podatke ugovora. Klik na broj ugovora ili strelicu otvara
stranicu `/ugovori/detalji/:id` sa kompaktnim pregledom ugovora.
Naslov sadrži SAP broj i predmet ugovora. Prikazuju se godina, broj nabavke,
pozicija plana, DMS broj, period važenja i zaduženi komercijalista.
Operativna lica prikazana su jedno ispod drugog, samo za popunjene pozicije.
Kolona Kontakt prikazuje ime, email i telefon. Detalji koriste
`GET /ugo_evid/:id/detalji`, gde je `id` ID evidencije (`id_ugo_evid`).
Sekcija partnera prikazuje naziv dobavljača i njegova lica: ime, radno mesto,
telefon, email i rolu. Service Level Manager je posebno istaknut.
Adresa i grad prikazani su ispod naziva dobavljača, u detaljima i u tabelama
otvorenih i zatvorenih ugovora. Prazna polja se izostavljaju.
Kontakt sačuvan u `UGO_EVID` prikazuje se prvi kao „Kontakt za ovaj ugovor“.
Aktuelni SLM se izostavlja samo kada je jedinstveno i pouzdano podudaranje:
ako postoje oba ID-a moraju biti jednaka, uz iste normalizovane kontakt podatke.
Bez oba ID-a potrebno je isto neprazno ime i bar isti email ili telefon;
sva ostala polja takođe moraju biti jednaka. Različita ili nedostajuća polja
čuvaju oba prikaza, kao i neodređena višestruka podudaranja.
Poređenje zanemaruje veličinu slova, suvišne razmake i formatiranje telefona,
ali ne pretpostavlja pozivni broj zemlje. Ostale role se ne uklanjaju.
Ako nema sačuvanog kontakta, prikazuju se samo lica dobavljača.
Detalji rade i pri direktnom otvaranju i osvežavanju; povratak iz pregleda čuva filter i stranicu.
Broj rezultata
predstavlja broj evidencija iz pogleda, ne broj različitih SAP ugovora.
Frontend u ovoj fazi omogućava samo pregled.

## Sesija

Backend ove aplikacije vraća Bearer tokene u JSON-u. Frontend ih šalje kroz
`Authorization` i obnavlja preko `POST /tokens/renew_access`. Sesija se čuva u
`sessionStorage` tekućeg taba; AD lozinka se ne čuva. Odjava briše lokalnu sesiju.
Backend nema endpoint za opoziv tokena, pa već izdati tokeni važe do isteka.
Zaštićene stranice preusmeravaju na prijavu kada sesija istekne.

## Produkcija

```powershell
npm run build
```

Rezultat je folder `dist`. Web server treba da vraća `index.html` za frontend
putanje (SPA fallback), a `/api/*` da prosleđuje Go backendu bez `/api` prefiksa.
Alternativno pre build-a postavi `VITE_API_BASE_URL=https://adresa-backenda`;
tada backend mora dozvoliti taj frontend origin kroz CORS. `VITE_*` vrednosti
ugrađuju se u build; promena zahteva novi build.

`npm run preview` služi za lokalnu proveru build-a, na portu 4173.

## Provera

```powershell
npm run test:e2e
```

Browser testovi koriste lokalne simulirane API odgovore; ne pristupaju AD-u ni
Oracle bazi. Pokrivaju prijavu, portal, filter, paginaciju, detalje, zatvorene
ugovore, obnovu tokena, odjavu, greške servera i mobilni meni.
Na Windows-u koriste instalirani Microsoft Edge. Za drugi browser postavi
`PLAYWRIGHT_CHANNEL` ili prilagodi `playwright.config.js`.
