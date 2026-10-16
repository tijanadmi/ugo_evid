# Razvojni zahtevi: realizacija i dokumenta

Pre pokretanja nove verzije aplikacije, jednom kao TED izvršiti
`development_request_documents.sql`, nakon ranijih migracija održavanja.
Aplikaciju za vreme migracije zaustaviti. Skripta nije predviđena za ponovno izvršavanje.

- `UGO_RAZ_ZAHTEV.DATUM_REALIZACIJE` čuva stvarni datum izvršenja usluge.
  `DATUM_POTPISA` ostaje datum potpisa završnog zapisnika.
- Stari zahtevi dobijaju NULL za datum realizacije; datum se ne pretpostavlja.
  Pri čuvanju završenog zahteva traže se oba datuma. Realizacija ne sme biti
  pre zahteva, posle potpisa, niti u budućnosti. Postojeća provera roka potpisa ostaje.
- `UGO_DOC.ID_RAZ_ZAHTEV` omogućava više dokumenata po zahtevu. Četiri šifre su
  `RAZ_ZAHTEV`, `RAZ_DETALJI`, `RAZ_PONUDA`, `ZAPISNIK_RAZ`.
  Postojeći dokumenti vezani preko `ID_ZAHTEV_DOC` / `ID_UGO_DOC` ostaju vidljivi.
- Brisanje je zabranjeno za završen zahtev. U ostalim statusima postavlja `OBRISAN`, povećava verziju i
  evidentira korisnika i vreme. Zahtev se isključuje iz aktivne liste, podsetnika
  i obračuna fonda. Dokumenti ostaju sačuvani i dostupni u Dokumentima ugovora.
  Fizičko brisanje fajlova ili zahteva se ne vrši.
- Sve izmene i brisanje koriste postojeće zaključavanje održavanja i verziju.
  Upload proverava pripadnost zahteva ugovoru, status, tip dokumenta i aktivno zaključavanje.

U redu zahteva meni sadrži Izmeni/Obriši. Upload otvara unos dokumenta vezanog
za taj zahtev; Prikaži otvara samo njegove dokumente, sa pregledom/preuzimanjem.
Više priloga tipa Detalji zahteva dodaje se pojedinačnim uploadima.
Datumi u detaljima ugovora i unosu održavanja prikazuju se kao dd.mm.yyyy;
API i baza i dalje koriste ISO datume.
