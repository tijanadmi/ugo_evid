# Izmena odrzavanja i razvojni fondovi

## Instalacija

Prethodna skripta maintenance.sql mora vec biti izvrsena. Ne pokretati je ponovo.
Zaustaviti aplikaciju, napraviti backup i pokrenuti maintenance_edit.sql jednom kao TED preko SQLcl/SQL*Plus ili SQL Developer Run Script/F5. Oracle DDL se implicitno potvrdjuje: skripta nije idempotentna niti se sva izvrsena DDL naredjenja ponistavaju rollback-om.

Ako UGO_EDIT_LOCK ima rucno dodatu CHECK listu dozvoljenih ENTITY_TYPE vrednosti, prosiriti je sa UGO_ODRZAVANJE, uz ocuvanje svih postojecih vrednosti. Skripta ispisuje definicije postojecih CHECK ogranicenja za proveru; nije bezbedno automatski menjati nepoznata poslovna ogranicenja. Ako takve liste nema, nije potrebna dopuna.

Zatim postaviti novi backend i frontend zajedno. API vise ne koristi poslate datum_potpisa/datum_kraja kao izvor datuma: datume ugovora cita iz SAP-a.

## Postojeci podaci

DATUM_POTPISA i DATUM_KRAJA ostaju sacuvani i vidljivi u pregledu kao ranije uneti datumi za proveru. Ne kopiraju se automatski u novi POCETAK_ODRZAVANJA. Prazan POCETAK_ODRZAVANJA znaci da se koristi aktuelni SAP_UGOVORI.POCETAK_UG; kraj se uvek cita iz SAP_UGOVORI.KRAJ_UG.

Pregledati vec uneta odrzavanja i, gde je potrebno, izabrati drugi datum pocetka. Ako SAP pocetak nedostaje, potreban je drugi datum. Ako SAP kraj nedostaje, prvo dopuniti SAP podatak. Pocetak odrzavanja ne sme biti posle kraja ugovora.

Postojeca razvojna podrska nema izmisljene fondove: prikazuje nule dok korisnik ne unese stvarno ugovorene casove. Prilikom cuvanja ukljucene razvojne podrske bar jedan fond mora biti pozitivan.

## Koriscenje

Detalji ugovora > Unesi odrzavanje / Izmeni odrzavanje.
- Period: mesecno, kvartalno, polugodisnje, godisnje, drugi broj meseci ili posebno ugovoreni periodi.
- Opcioni drugi pocetak odrzavanja; prelazak nazad na SAP uklanja samo opcioni datum.
- Razvojna podrska: celi, nenegativni casovi za inzenjera/programera, tehnicara i konsultanta. Iskljucivanje razvojne podrske postavlja fondove na nulu prilikom cuvanja.
- Brava se stice PRE otvaranja izmene, obnavlja na 30 sekundi i istice posle 2 minuta bez obnavljanja. Zatvaranje uz potvrdu oslobadja je; gubitak brave cuva unos, ali onemogucava snimanje.
- VERSION odrzavanja stiti ceo skup: osnovne podatke i fondove. Sve se cuva u jednoj transakciji. ENTITY_ID brave UGO_ODRZAVANJE predstavlja ID ugovora iz UGO_EVID.

Ova faza ne uvodi razvojne zahteve, obracun utroska, raspored tehnickih perioda ni dostavu podsetnika. Kada se uvedu zahtevi, dodati proveru da se fond ne smanjuje ispod rezervisanih/utrosenih casova. ID fonda ostaje stabilan pri izmeni radi buducih veza.

Proveriti migraciju na Oracle test bazi: lokalni SQL testovi koriste simulirani drajver, a browser testovi simulirani API.
