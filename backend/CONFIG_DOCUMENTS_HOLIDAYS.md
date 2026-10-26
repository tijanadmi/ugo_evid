# Dokumenti i servis praznika

U `app.env` postaviti:

```dotenv
DOCUMENT_ROOT=/srv/ugo_evid/documents
HOLIDAYS_API_URL=https://nagerholidays.com/api/v4/Holidays/RS/
```

`DOCUMENT_ROOT` je korenski folder dokumenata. Na Ubuntu serveru preporučena je
apsolutna putanja; korisnik koji pokreće backend mora imati pravo čitanja i pisanja.
Relativna putanja se računa od radnog direktorijuma procesa. Lokalna vrednost
`./doc` zadržava dosadašnju lokaciju. Promena putanje ne premešta postojeće fajlove:
pri prelasku na novu lokaciju preneti ceo sadržaj starog foldera, uz istu strukturu.
I alat `doc-reconcile` koristi ovu postavku.

`HOLIDAYS_API_URL` je osnovna adresa za Srbiju, bez godine. Backend dodaje godinu,
na primer `2026`. Završna kosa crta se dodaje automatski ako nedostaje. URL mora
biti HTTP/HTTPS, bez korisničkih podataka, query parametara i fragmenta.
Servis mora vratiti postojeći Nager Holidays v4 JSON format; promena adrese sama
po sebi ne omogućava drugi format odgovora.

Obe vrednosti su obavezne. Nakon izmene restartovati backend. Promenljive
okruženja procesa, ako su postavljene, imaju prednost nad `app.env`.
Tajne iz ostatka `app.env` ne objavljivati u repozitorijumu ili izveštajima.
