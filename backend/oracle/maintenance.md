> Naknadna dorada: pogledati maintenance_edit.md i izvrsiti maintenance_edit.sql posle ove pocetne migracije. Ona uvodi SAP datume, izmenu sa zakljucavanjem i fondove razvojnih casova.

# Tip ugovora i osnovni podaci odrzavanja

## Instalacija

1. Na poslovnoj bazi vec moraju postojati UGO_EVID, UGO_KOR, UGO_TIP_DOC i UGO_DOC (prethodna dorada dokumenata).
2. U terminu za instalaciju zaustaviti aplikaciju i napraviti backup seme. Pokrenuti `maintenance.sql` kao TED u SQLcl/SQL*Plus (ili SQL Developer Run Script/F5). Skripta se izvrsava jednom, nije namenjena ponovnom pokretanju. Oracle DDL implicitno radi COMMIT; neuspeh ne ponistava prethodno kreirane objekte.
3. Proveriti rezultat skripte i da je TRG_EVID_TIP_REQUIRED validan. Postaviti novi backend i frontend zajedno, pa pokrenuti aplikaciju. Stari backend vise ne moze da upisuje ugovore bez tipa.
4. Ne pokretati ponovo documents.sql. maintenance.sql dodaje samo nove tipove dokumenta, uz ocuvanje postojecih ID vrednosti.

## Koriscenje

- Otvoreni ugovori > Izaberi ugovor > izabrati ugovor > obavezno izabrati Tip ugovora > Evidentiraj ugovor.
- Za postojeci otvoreni ugovor: detalji > Tip ugovora i odrzavanje > Odredi tip ugovora.
- Za tip Odrzavanje softvera: Unesi odrzavanje. Datum potpisa unosi korisnik: SAP pocetak se ne tumaci automatski kao datum potpisa.
- Period meseci: 1 mesecno, 3 kvartalno, 12 godisnje ili drugi ceo broj 1-120. Prazno predstavlja posebno ugovorene periode.
- Podsetnici: 0-365 dana ranije; pocetne vrednosti 7 za tehnicku, 3 za razvojnu podrsku.
- Dokument: POTVRDA_TEH ili ZAPISNIK_TEH. Menjati NAZIV radi prikaza, ne SIFRA koju koristi aplikacija.
- Odrzavanje ima jedan slog po ugovoru. Posle unosa prikazuju se sacuvani podaci; izmena odrzavanja nije deo ove faze. Promena tipa nakon unosa odrzavanja je blokirana.

## Obim ove faze

Ova migracija uvodi UGO_TIP_UGOVORA, vezu na UGO_EVID i UGO_ODRZAVANJE. Fondovi celih casova, razvojni zahtevi, tehnicki periodi, kalendar, obrada i dostava podsetnika bice predmet narednih dorada. Broj dana za podsetnike se sada cuva, ali se obavestenja jos ne generisu.

API: GET /ugo_tip_ugovora; GET /ugo_evid/:id/podesavanja; PUT /ugo_evid/:id/tip; POST /ugo_evid/:id/odrzavanje. Izmena tipa koristi version iz podesavanja (TIP_VERSION u bazi). Kreiranje odrzavanja i promena tipa zakljucavaju roditeljski slog unutar transakcije i ponovo proveravaju organizaciju i otvoren status. Dva istovremena upisa ne prepisuju podatke; dobijaju konflikt/obavestenje o zauzetosti.

Nije potrebna izmena dozvoljenih entity_type vrednosti UGO_EDIT_LOCK za ovu fazu: koriste se kratke transakcijske brave, bez dugotrajne brave tokom popunjavanja forme.

Lokalni testovi koriste simulirani API i SQL drajver. Migracija i Oracle integracija moraju se proveriti na poslovnoj/test bazi.
