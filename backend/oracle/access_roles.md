# Prava pristupa za evidenciju ugovora

## Instalacija

1. Napraviti rezervnu kopiju `TED.UGO_KOR_ROLE` i sačuvati prethodnu verziju aplikacije.
2. Zaustaviti backend tokom migracije. Pokrenuti **access_roles.sql jednom, kao TED**, u režimu izvršavanja skripte (SQL Developer F5 / SQLcl / SQL*Plus).
3. Pregledati dodele prava. Svi postojeći slogovi dobijaju `IUD` za postojeću OJ; njihov status se čuva. Eventualni dupli aktivni slogovi ostaju u tabeli, ali višak postaje neaktivan.
4. Izabranom korisniku zasebno dodeliti `ADM_KALENDAR` (primer ispod), pa COMMIT.
5. Buildovati i postaviti novi backend i frontend zajedno; restartovati backend. Stari backend nije kompatibilan sa novim višestrukim/globalnim dodelama.

Oracle DDL automatski potvrđuje promene. Skripta zaustavlja rad na prvoj grešci; ne pokretati je ponovo naslepo posle delimičnog izvršenja. Skripta ne sadrži produkcione korisničke podatke, lozinke ili konfiguraciju servera.

## Model i primeri

`UGO_ROLA` ima fiksne identifikatore: 1 = GLE, 2 = IUD, 3 = ADM_KALENDAR.
`IUD` uključuje pregled. Dodele su aditivne. `SVE_OJ` obuhvata i buduće OJ.
`ADM_KALENDAR` ne daje nikakva prava nad ugovorima ili administracijom šifarnika.

Primer za korisnika 2: najpre pregledati njegove postojeće dodele. Sledeći blok **zamenjuje njegove aktivne dodele** primerom GLE za OJ 1, 2, 3 i IUD za OJ 4; izvršiti samo ako je to zaista željena dodela i sve četiri OJ postoje.

```sql
UPDATE TED.UGO_KOR_ROLE SET STATUS='N', DATIZM=SYSDATE
WHERE ID_UGO_KOR=2 AND STATUS='A';

INSERT INTO TED.UGO_KOR_ROLE (ID_UGO_KOR,ID_UGO_ROLA,OPSEG,ID_UGO_ORG,STATUS,DATPRI)
VALUES (2,1,'OJ',1,'A',SYSDATE);
INSERT INTO TED.UGO_KOR_ROLE (ID_UGO_KOR,ID_UGO_ROLA,OPSEG,ID_UGO_ORG,STATUS,DATPRI)
VALUES (2,1,'OJ',2,'A',SYSDATE);
INSERT INTO TED.UGO_KOR_ROLE (ID_UGO_KOR,ID_UGO_ROLA,OPSEG,ID_UGO_ORG,STATUS,DATPRI)
VALUES (2,1,'OJ',3,'A',SYSDATE);
INSERT INTO TED.UGO_KOR_ROLE (ID_UGO_KOR,ID_UGO_ROLA,OPSEG,ID_UGO_ORG,STATUS,DATPRI)
VALUES (2,2,'OJ',4,'A',SYSDATE);
COMMIT;
```

Posebna dodela administratora kalendara (zameniti 2 stvarnim ID korisnika):

```sql
INSERT INTO TED.UGO_KOR_ROLE (ID_UGO_KOR,ID_UGO_ROLA,OPSEG,STATUS,DATPRI)
VALUES (2,3,'GLOBALNO','A',SYSDATE);
COMMIT;
```

Pregled svih OJ:

```sql
INSERT INTO TED.UGO_KOR_ROLE (ID_UGO_KOR,ID_UGO_ROLA,OPSEG,STATUS,DATPRI)
VALUES (2,1,'SVE_OJ','A',SYSDATE);
COMMIT;
```

Postojeći trigger i sekvenca `UGO_KOR_ROLE` popunjavaju ID kao i ranije. Novi šifarnik nema potrebu za sekvencom jer sadrži tri fiksne role.

## Ponašanje aplikacije

- Backend učitava aktivnog korisnika i prava iz baze pri svakom zahtevu. Promena prava ne zahteva novu prijavu; frontend ih osvežava pri povratku u prozor i svakih 30 sekundi.
- Otvoreni/zatvoreni ugovori, podsetnici, partneri i Excel primenjuju OJ opseg u SQL-u pre brojanja i paginacije. Nedozvoljeni filter ili pristup konkretnom ugovoru vraća 403.
- Jedna dozvoljena OJ za unos bira se automatski. Ako ih ima više, korisnik je bira pre SAP ugovora. OJ ide u query parametru `id_ugo_org` i proverava se na serveru u svim koracima unosa.
- Svaka izmena ugovora, održavanja, perioda, zahteva, dokumenata i pribavljanje zaključavanja zahteva IUD za OJ tog ugovora. Postojeća poslovna pravila i zaključavanje ostaju na snazi.
- Kontakt dobavljača je zajednički podatak: IUD nad bar jednom OJ sa ugovorom tog dobavljača daje pravo izmene kontakta. Zato izmena može biti vidljiva i drugim OJ. U listi partnera server vraća `can_write`; u detalju ugovora akcije su dodatno ograničene pravom izmene tog ugovora.
- Kalendar mogu pregledati korisnici sa pristupom ugovorima i administrator kalendara. Generisanje godine, ručne izmene, potvrda dana i preuzimanje praznika zahtevaju ADM_KALENDAR.
- Stari neograničeni list servisi `/sapugovori` i `/ugo_dob_lica` vraćaju 403. Portal koristi ograničeni izbor ugovora i kontakte kroz partnere/detalje. Nijedna nova rola ne daje pravo izmene šifarnika OJ ili rola kontakt lica; ti stari CRUD servisi takođe odbijaju izmene.
- Administratorski ekran za dodelu rola nije deo ove izmene; prava se za sada dodeljuju kroz bazu.

## Provera na poslovnoj mreži

Proveriti korisnika sa mešovitim pravima, GLE za sve OJ, korisnika sa više IUD OJ i administratora kalendara bez OJ. Posebno proveriti Excel filter, upload/preuzimanje dokumenata, nedozvoljene direktne API pozive i promenu prava tokom aktivne sesije. Automatski testovi koriste simuliranu bazu/API; Oracle migraciju treba proveriti na testnoj bazi pre produkcije.
