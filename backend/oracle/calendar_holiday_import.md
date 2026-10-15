# Mesečni kalendar i uvoz praznika Srbije

## Instalacija

Zaustaviti aplikaciju i kao TED **jednom** pokrenuti `calendar_holiday_import.sql`,
nakon prethodnog `maintenance_tracking.sql`. Zatim instalirati backend i frontend zajedno.
Oracle DDL implicitno potvrđuje promene; skriptu ne pokretati ponovo ako je kolona već kreirana.
Skripta nije izvršena nad poslovnom bazom u razvojnom okruženju.

Nova kolona `UGO_RADNI_KALENDAR.IZVOR` razlikuje `GENERISAN`, `RUCNO` i `PRAZNIK`.
Stari potvrđeni datumi, opisi i odstupanja od podrazumevanog vikenda zaštićeni su kao ručni.
Ranija evidencija ne može pouzdano da razlikuje svaku ručnu izmenu; zato su potvrđeni dani zaštićeni konzervativno.
U pregledu uvoza zaštićeni dan može izričito da se označi za zamenu.

## Korišćenje

- Prikazuje se jedan mesec; strelice prelaze i granicu godine. Godina i mesec su u URL-u.
- Zeleno znači radni dan, ljubičasto neradni. Vikendi su podrazumevano neradni,
  ali klik može da ih pretvori u radne dane. Promena se odmah čuva kao ručna i postaje nepotvrđena.
- Klik na radni praznik/neradni dan ponovo menja status. Pri grešci se vraća prethodna boja.
- `Opis / korekcija` uređuje izabrani datum, a `Izmeni dane` podržava opseg.
- `Preuzmi praznike za Srbiju` je dostupno nakon generisanja godine. Prikazuje pregled;
  samo dugme za uvoz menja bazu. Ručni datumi nisu unapred označeni.
- Uvoz menja samo izabrane datume u neradne. Ne uklanja ranije praznike niti automatski potvrđuje godinu.
  Identičan ponovni uvoz ne poništava postojeću potvrdu dana.
- Nakon provere koristi se `Potvrdi godinu`. Sačuvani rokovi razvojnih zahteva se ne preračunavaju.

## Servis i dostupnost

Backend koristi dokumentovani Nager.Date Community v4 endpoint:
`https://nagerholidays.com/api/v4/Holidays/RS/{year}`.
Dokumentacija: https://nagerholidays.com/api

Ubuntu server mora imati izlazni HTTPS pristup `nagerholidays.com:443`.
HTTP klijent koristi standardnu Go proxy konfiguraciju iz okruženja (`HTTPS_PROXY`, `NO_PROXY`),
timeout 12 sekundi, ograničenje odgovora 1 MiB i ne prati preusmerenja.
Klijent ne prosleđuje korisničke tokene niti podatke o ugovorima spoljnom servisu — samo godinu i oznaku RS.
Nema API ključa u ovoj integraciji. Ako servis ne radi, postojeći kalendar i ručne izmene ostaju dostupni.

Preuzimaju se samo nacionalni praznici kategorije `Public`. Ne uvoze se školski,
bankarski, regionalni ili opciono obeleženi datumi. Neispravan/nepotpun odgovor se odbija.
Iz naziva vaskršnjeg datuma u odgovoru dopunjava se period od petka do ponedeljka;
za državne praznike koji padnu u nedelju dodaje se prvi sledeći radni dan.
Verski praznici se ne prenose tim pravilom. Dopune su označene u pregledu.

Prilikom razvoja proveren je stvarni odgovor za 2026. godinu: servis nije sadržao 17. februar,
pa je uvedena dopuna prenetih dana. Provera zvaničnog spiska i budućih zakonskih izmena i dalje je potrebna:
- NBS: https://www.nbs.rs/sr/o_nbs/radno-vreme-i-neradni-dani/
- MSP, protokolarni priručnik (pravilo prenosa): https://www.msp.gov.rs/sites/default/files/inline-files/PROTOKOLARNI%20PRIRUCNIK%20Lat.pdf

## API i konkurentne izmene

- `GET /radni_kalendar/:year/praznici`: autorizovani pregled iz spoljnog API-ja, bez upisa u bazu.
- `POST /radni_kalendar/:year` sa `action: "import"`, `version` iz pregleda godine i
  `praznici: [{datum, naziv, zameni_rucno: false}]` čuva odabrane datume.
- Zamena ručnog datuma zahteva `zameni_rucno: true` baš za taj datum.
- Import proverava godinu, broj, jedinstvenost datuma i dužinu naziva.
  Koristi postojeći hash verzije godine i kratko zaključavanje tabele, pa stale pregled vraća 409.
  Svi izabrani praznici čuvaju se u jednoj transakciji.

Pristup kalendaru ostaje isti: aktivni prijavljeni korisnici sa važećom organizacionom jedinicom.
