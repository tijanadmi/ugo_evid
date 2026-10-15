# Radni kalendar u aplikaciji

Nije potrebna nova SQL migracija. Koristi se postojeća `TED.UGO_RADNI_KALENDAR`
iz `maintenance_tracking.sql`, sa kolonama DATUM, RADNI, POTVRDJEN i OPIS.

1. U bočnom meniju otvoriti **Radni kalendar** i izabrati godinu.
2. **Generiši kalendar** dopunjava nedostajuće datume; vikendi su neradni,
   ostali dani radni. Praznici se ne pretpostavljaju. Postojeće vrednosti se ne menjaju.
3. **Izmeni dane** menja jedan datum ili raspon, uz radni status i opis.
   Opseg obuhvata sve datume, uključujući vikende. Za izuzetno radnu subotu izabrati Radni dan = Da.
4. Nakon pregleda svih praznika i izuzetaka, **Potvrdi godinu** potvrđuje celu godinu.
   Naknadna izmena označava izmenjene dane kao nepotvrđene; godinu treba ponovo potvrditi.

Kalendar je zajednički za sve organizacione jedinice i ugovore. Pristup imaju
prijavljeni aktivni korisnici sa važećom organizacionom jedinicom, prema postojećoj autorizaciji aplikacije.
Kratka transakcija zaključava tabelu pri upisu; SHA-256 snimka godine proverava
da korisnik menja/potvrđuje podatke koje je zaista video. Zastarela potvrda vraća 409.
Nije potrebno dodavati VERSION niti novu vrstu u UGO_EDIT_LOCK.

API: GET/POST `/radni_kalendar/:year`; POST prima `action` (`generate`, `change`, `approve`),
`version` iz GET odgovora, a za promenu i `od`, `do`, `radni` (`D`/`N`), `opis`.

Rokovi već sačuvanih razvojnih zahteva ostaju nepromenjeni. Za nove obračune,
datumi do izračunatog roka moraju postojati i biti potvrđeni, uključujući narednu godinu ako rok prelazi u nju.
Forma zahteva otvara kalendar u novom tabu da bi uneti zahtev ostao u formi.
