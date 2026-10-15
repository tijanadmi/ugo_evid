# Periodi, razvojni zahtevi i podsetnici

## Instalacija

1. Prethodno moraju biti primenjeni `documents.sql`, `maintenance.sql` i `maintenance_edit.sql`.
2. Zaustaviti aplikaciju, napraviti backup i jednom pokrenuti `maintenance_tracking.sql` kao TED.
   DDL u Oracle-u implicitno potvrđuje promene; neuspeh zahteva proveru već kreiranih objekata pre nastavka.
3. Po potrebi pokrenuti `maintenance_calendar.sql` za svaku potrebnu godinu. To je samo predlog radnih dana:
   proveriti i dopuniti praznike, prenete praznike i izuzetke po poslovnom kalendaru, pa eksplicitno potvrditi datume.
   Nepotvrđen ili nedostajući datum zaustavlja obračun roka razvojnog zahteva.
4. Instalirati novi backend i frontend zajedno, pa pokrenuti aplikaciju.

Nove tabele: `UGO_TEH_PERIOD`, `UGO_RAZ_ZAHTEV`, `UGO_RADNI_KALENDAR`.
Postojeći `UGO_RAZ_FOND` ostaje fond po vrsti rada; tabela stavki zahteva nije potrebna.
`UGO_OBAVESTENJE`, SMTP, email adrese i pozadinski proces nisu deo ove isporuke.
Podsetnici se računaju prilikom API poziva, bez posebnog job-a.

## Poslovna pravila

- Generisanje perioda: kalendarski mesec/kvartal/polugodište/godina ili period od datuma početka održavanja.
  Drugi pravilni intervali koriste datum početka. Ručni unos podržava proizvoljne periode.
- Kalendarski prvi period može biti skraćen. Poslednji period se skraćuje na kraj ugovora.
  Obračun od početka koristi originalni dan, uz ograničenje na poslednji dan kraćeg meseca, bez pomeranja sidra.
- UI generiše samo dok nema perioda; pregled se potvrđuje pre transakcionog unosa svih perioda.
  Unos ne prepisuje postojeće redove. Backend odbija preklapanje aktivnih perioda i duplirane datume.
  Rupe u pokrivenosti prikazane su kao upozorenje. Izmene su pojedinačne; planirani periodi mogu da se obrišu, a ostali otkazuju promenom statusa.
- Tip dokumenta perioda bira se između `POTVRDA_TEH` i `ZAPISNIK_TEH`.
- Tehnički podsetnik počinje `PERIOD_DO - TEH_PODSETNIK_DANA` i ostaje dok period nije završen/otkazan.
  Sam kraj perioda nije prekoračenje roka potpisa. Ako je upisan opcioni `ROK_DOKUMENTA`, posle njega se prikazuje kašnjenje.
- Zahtev ima jednu vrstu rada i ceo pozitivan broj časova. Broj radnih dana nije izveden iz časova.
  Prvi brojani dan je dan posle datuma podnošenja, ako je radan; neradni dani se preskaču.
  Sačuvani rok je snimak obračuna i menja se samo pri izmeni datuma zahteva ili broja dana.
  Potpis mora biti pre datuma roka (strogo `<`), prema dogovorenom pravilu.
  Primer: zahtev u ponedeljak + 2 radna dana daje sredu kao rok; potpis mora biti pre srede.
- Statusi `NACRT` i `OTKAZAN` ne zauzimaju fond. `PODNET`, `U_RADU`, `NA_POTPISU` i `ZAVRSEN` ulaze u zbir.
  Zahtev može biti nacrt i kada nema dovoljno preostalih časova; podnošenje se odbija ako bi premašilo fond.
- Završavanje zahteva ili perioda traži datum potpisa. Datumi u budućnosti nisu dozvoljeni.
- Dokumenti se prvo uploaduju postojećim ekranom Dokumenta ugovora, zatim se povezuju kroz formu perioda/zahteva.
  Povezani dokument mora biti aktivan i pripadati istom ugovoru. Fajl nije obavezan da bi istorijski podaci mogli da se evidentiraju.
- Svi upisi koriste postojeći lease `UGO_ODRZAVANJE`, sa `ENTITY_ID = UGO_EVID.ID`.
  Jedan ugovor ima jednu aktivnu sesiju izmene održavanja/perioda/zahteva. Drugi ugovori se uređuju nezavisno.
  Zaključavanje se preuzima pre otvaranja forme, obnavlja i oslobađa uz uspešnu transakciju ili odustajanje.
  Transakcija dodatno zaključava roditeljski slog i fond, proverava organizaciju, otvorenost i verziju sloga.
- Nije moguće smanjiti fond ispod zauzetih časova, isključiti razvoj uz postojeće neotkazane zahteve,
  ili izmeniti početak/kraj održavanja tako da postojeći neotkazani periodi/zahtevi budu izvan opsega.
- Nulti fondovi se ne prikazuju na detalju ugovora; sve vrste ostaju u formi podešavanja.
- Podsetnici na početnoj stranici obuhvataju dostupnu organizacionu jedinicu, a detalji samo taj ugovor.
  Lično filtriranje preko SAP odgovornih lica i email dostava ostaju za sledeću fazu.

## API

- `GET /ugo_evid/:id/pracenje`: periodi, zahtevi i stanje fondova.
- `POST /ugo_evid/:id/pracenje/preview`: `{ "mode": "calendar" }` ili `anchor`.
- `POST /ugo_evid/:id/pracenje/periodi`: `{ "lock_token": "...", "items": [...] }`.
- `POST /ugo_evid/:id/pracenje/zahtevi`: isti omotač, tačno jedan zahtev.
  Novi slog ima `id=0`; izmena šalje postojeći ID i `version` kao string.
- `GET /podsetnici`: podsetnici autorizovane organizacione jedinice.
- Lease koristi postojeće `POST/PUT/DELETE /ugo_evid/:id/odrzavanje/lock` rute.

SQL nije izvršen nad poslovnom Oracle bazom u razvojnom okruženju. Lokalni testovi koriste SQL driver/mokove,
pa na poslovnoj mreži treba proveriti instalaciju, kalendar i dva istovremena taba.

## Brisanje planiranog perioda

Meni perioda nudi izmenu i brisanje. DELETE /ugo_evid/:id/pracenje/periodi koristi isti omotac sa jednim slogom (id, version) i lock_token. Fizicko brisanje dozvoljeno je samo za status PLANIRAN, uz roditeljski lock, vazeci lease i verziju. Dokument se ne brise. Za ostale statuse ostaje izmena/otkazivanje. Nije potrebna SQL migracija.
