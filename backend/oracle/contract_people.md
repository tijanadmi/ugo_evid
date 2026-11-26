# Lica angažovana po ugovoru

## Primena

1. Napraviti rezervnu kopiju baze i zaustaviti stari backend.
2. Kao TED pokrenuti **contract_people.sql**, jednom. Skripta proverava postojeće
   eksplicitne SLM veze, kreira UGO_EVID_LICA, prenosi SLM-a svakog ugovora i uklanja
   poznati stari jedinstveni indeks UQ_UGO_DOB_LICA_SLM_A.
3. Pregledati završni upit: ugovorima bez starog ID_UGO_DOB_LICA potrebno je ručno
   izabrati SLM-a. Ime/telefon/email bez ID-a ne povezuju se automatski.
4. Objaviti novi backend i frontend zajedno, pa proveriti dva ugovora istog
   dobavljača sa različitim SLM-ovima i različitim OJ pravima.

Oracle DDL automatski potvrđuje izmene. Ako skripta stane, pregledati stanje pre
ponavljanja; nije predviđena za ponovno pokretanje nad već kreiranom tabelom.
Ako u bazi postoji dodatno ručno kreirano ograničenje ili trigger za jednog SLM-a
po dobavljaču pod drugim nazivom, potrebno ga je zasebno pregledati.
Skripta nije pokrenuta na poslovnoj bazi.

## Model

UGO_DOB_LICA ostaje imenik dobavljača. Njegova postojeća rola služi kao podrazumevana
klasifikacija (npr. lista SLM-ova pri unosu ugovora). UGO_EVID_LICA određuje stvarnu
ulogu na konkretnom ugovoru. Tehnička podrška nije obavezna i ne prenosi se svim
ugovorima automatski. U tabeli nema DATUM_OD/DATUM_DO.

STATUS A/P u vezi označava aktivno/prethodno angažovanje. Jedinstveni indeksi
dozvoljavaju najviše jednog aktivnog SLM-a po ugovoru i sprečavaju duplu aktivnu
vezu istog lica i uloge. VERSION štiti od zastarelih izmena.
ID_KORISNIKA je prazan samo za migrirane veze, jer stvarni autor stare veze nije poznat.

Stare UGO_EVID kolone ID_UGO_DOB_LICA, IME, TELEFON, EMAIL još se ne brišu:
zadržavaju stare podatke i kompatibilnost postojeće strukture/view-a. Nova
registracija ih popunjava radi kompatibilnosti, ali aktuelni SLM, detalji, liste
i Excel izvoz čitaju iz UGO_EVID_LICA. Promena angažovanja ne sinhronizuje stare
kolone. Spoljni izveštaji koji direktno čitaju te kolone/view moraju se prilagoditi
pre njihovog kasnijeg uklanjanja.

## Rad u aplikaciji

- Unos ugovora: izbor aktivnog SLM-a ili unos novog, čak i kada već postoje drugi.
  Lice, ugovor i početno angažovanje čuvaju se u istoj transakciji.
- Detalji ugovora: „Promeni SLM-a“, „Dodaj lice na ugovor“, „Završi angažovanje“.
  Novo lice iz ovog dijaloga prvo se sačuva u imeniku, zatim se izabere i poveže.
- Zamena SLM-a pasivizira prethodnu vezu i dodaje novu u jednoj transakciji.
  Prethodna angažovanja ostaju dostupna. Nema fizičkog brisanja veze.
- „Moji partneri“ ostaje zajednički imenik. Izmena imena/telefona/emaila utiče na
  sva angažovanja. Brisanje lica koje ima veze sa ugovorima nije dozvoljeno.
- Pasivizacija u imeniku ne briše istoriju. Lice se više ne nudi za nova
  angažovanja, a na ugovoru ostaje upozorenje za zamenu. Pri pasivizaciji se
  prikazuju otvoreni ugovori dostupni pravima korisnika; tu mogu postojati i
  angažovanja u drugim OJ koja taj korisnik nema pravo da pregleda.

Izmene angažovanja zahtevaju IUD pravo za OJ tog otvorenog ugovora. Kratke
transakcije zaključavaju ugovor i izabrano lice; ID/verzija prethodne veze
sprečavaju da dva korisnika prepišu zamenu SLM-a. Otvoren dijalog ne drži dugotrajno
zaključavanje: zastarela izmena se odbija pri čuvanju. Postojeći lease mehanizam
za izmenu zajedničkog imenika ostaje na snazi.
