# Trajno brisanje dokumenata

Pre postavljanja koda pokrenuti `document_deletion.sql` jednom, kao TED, u režimu izvršavanja skripte (SQL Developer F5). Skripta kreira `UGO_DOC_BRISANJE`, bez promene postojećih dokumenata. Ako postoje stari strani ključevi ka UGO_DOC, skripta staje pre izmena; prvo završiti prethodnu migraciju `cleanup_legacy_maintenance.sql` prema njenom uputstvu. Ne uklanjati strane ključeve naslepo.

Zatim postaviti novi backend i frontend i restartovati backend. Nema novih env promenljivih; koristi se postojeći `DOCUMENT_ROOT`. Sve backend instance koje rade sa istom bazom moraju imati pristup istom skladištu dokumenata. Stari backend treba zaustaviti pre postavljanja nove verzije.

## Postupak

1. Korisnik potvrđuje trajno brisanje kroz postojeći meni dokumenta. Važe prava IUD za OJ, otvoren ugovor, verzija i važeće zaključavanje.
2. Jedna transakcija postavlja `UGO_DOC.STATUS='D'`, upisuje zahtev za brisanje/audit i oslobađa zaključavanje.
3. Backend briše fajl iz `DOCUMENT_ROOT`. Ne briše folder ugovora, druge fajlove ni rekurzivno sadržaj foldera. Već nepostojeći fajl smatra se obrisanim.
4. Nakon uspešnog uklanjanja fajla, druga transakcija briše UGO_DOC slog i popunjava `UGO_DOC_BRISANJE.ZAVRSENO`.

Audit čuva ID dokumenta i ugovora, originalni naziv, korisnika, vreme zahteva i završetka. Nema strani ključ ka dokumentu, jer ostaje posle njegovog brisanja. Datumi, statusi, sati i ostali podaci razvojnog zahteva/tehničkog perioda ne menjaju se.

Dokumenti D nisu u aktivnim listama i API odbija nove preglede/preuzimanja. Već preuzet fajl ili ranije započeto preuzimanje ne mogu se povući.

## Oporavak

Ako drugi deo ne uspe, API vraća HTTP 202 i `status: pending_deletion`. Frontend zatvara potvrdu, osvežava listu i prikazuje da brisanje nije završeno. Uspešna poruka se prikazuje tek nakon uklanjanja fajla i sloga.

Backend pri pokretanju i zatim svakog minuta obrađuje nedovršene zahteve. Greške zapisuje u serverski log sa ID dokumenta i pokušava ponovo. Za više servera i istovremeni API poziv koristi Oracle zaključavanje audit sloga. Ako je fajl obrisan, a potvrda baze nije uspela, ponovni pokušaj dovršava brisanje sloga. Nije potreban cron ili poseban Go proces. Rok od minut je interval između završenih prolaza; trajna greška skladišta/baze zahteva administratorsku intervenciju.

Pregled zahteva koji čekaju:

```sql
SELECT ID_DOC, ID_UGO_EVID, ORIGINALNI_NAZIV, ID_KORISNIKA, TRAZENO
FROM TED.UGO_DOC_BRISANJE
WHERE ZAVRSENO IS NULL
ORDER BY TRAZENO;
```

**Raniji logički obrisani dokumenti sa D ne brišu se automatski.** Za njih ne postoji novi zahtev u audit tabeli. Njihovo eventualno trajno uklanjanje treba obraditi posebno, nakon pregleda; migracija ih ne uključuje u red za brisanje.

Postojeća komanda `doc-reconcile` ostaje namenjena oporavku upload-a sa statusom P. Ne služi za brisanje.

Testovi koriste privremene fajlove i simuliranu bazu/API. SQL i konkurentni rad na Oracle serveru treba proveriti na testnoj bazi pre produkcije.
