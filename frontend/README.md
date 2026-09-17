# Evidencija ugovora — frontend

React/Vite aplikacija u JavaScript/JSX-u, po uzoru na `C:\react_workspace\tis_frontend`:
portal sa karticama, tamnoplavi bočni meni i zajedničko zaglavlje. Za sada je jedna
aplikacija, pa portal i pregled koriste isti projekat i jednu prijavu.

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
detalje svih 42 polja pogleda, uključujući šest odgovornih lica. Broj rezultata
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
