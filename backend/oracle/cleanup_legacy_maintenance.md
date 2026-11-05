# Uklanjanje starih kolona

Skripta: `cleanup_legacy_maintenance.sql`.

Ovo je promena modela, a ne samo promena redosleda kreiranja tabela.
`create_documents_tracking_clean.sql` je sada usklađena sa konačnim modelom
za novu bazu. Za postojeću bazu koristiti migraciju, ne CREATE skriptu.

| Tabela | Kolone koje se uklanjaju | Zamena |
| --- | --- | --- |
| UGO_RAZ_ZAHTEV | ID_ZAHTEV_DOC, ID_UGO_DOC | UGO_DOC.ID_RAZ_ZAHTEV; tip dokumenta određuje UGO_DOC.ID_TIP_DOC |
| UGO_TEH_PERIOD | ID_UGO_DOC | UGO_DOC.ID_TEH_PERIOD |
| UGO_ODRZAVANJE | DATUM_POTPISA, DATUM_KRAJA | Podaci ugovora iz SAP_UGOVORI; opcioni POCETAK_ODRZAVANJA za početak održavanja |

Zadržavaju se DATUM_POTPISA u UGO_RAZ_ZAHTEV i UGO_TEH_PERIOD: to su datumi
potpisivanja zapisnika/potvrde, a ne zastareli datum ugovora. Zadržavaju se
DATUM_REALIZACIJE, oba ID polja u UGO_DOC, ID_TIP_DOC i DATUM_DOKUMENTA.
DATUM_DOKUMENTA ostaje prema ranijem dogovoru, iako se ne unosi kroz formu.
ID_TIP_DOC u tehničkom periodu označava očekivanu vrstu dokumenta i nije višak.

## Izvršavanje

1. Sačuvati rezervnu kopiju baze. Zaustaviti sve instance aplikacije.
2. Pokrenuti skriptu kao TED, u režimu Run Script (F5 u SQL Developer-u).
3. Ako provera prijavi dokument sa više vlasnika, rešiti taj slučaj pre nastavka.
   Jedan dokument ne može istovremeno pripadati različitim zahtevima/periodima.
   Skripta ne bira vlasnika umesto korisnika i ne briše takve veze.
4. Skripta pravi tri rezervne tabele, prenosi veze i proverava prenos, pa uklanja kolone.
   Stari datumi ostaju u UGO_BAK_ODR_LEGACY radi pregleda; ne kopiraju se automatski
   u POCETAK_ODRZAVANJA jer datum potpisa nije nužno početak održavanja.
5. Pre ponovnog pokretanja aplikacije OBAVEZNO instalirati prilagođen backend.
   Backend i frontend su prilagođeni u istoj izmeni; postaviti ih zajedno sa migracijom.

Oracle DDL se automatski potvrđuje: greška ne vraća već izvršene CREATE/ALTER
naredbe. Ako izvršavanje delimično uspe, sačuvati izlaz i pregledati stanje;
ne brisati rezervne tabele radi slepog ponavljanja skripte.

Skripta ne rekonstruiše ranije izgubljene strane ključeve zbog DROP TABLE CASCADE.
Ako su ranije skripte delimično izvršene, potrebno je proveriti i te veze.
Nije izvršena na Oracle bazi tokom pripreme.
