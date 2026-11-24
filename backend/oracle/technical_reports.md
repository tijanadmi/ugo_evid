# Šabloni i izveštaji tehničke podrške

## Redosled primene

Zaustaviti backend tokom migracije i prethodno napraviti rezervnu kopiju baze.
Kao TED pokrenuti jednom, redom:

1. `technical_period_quarter.sql` — dodaje opciono `UGO_TEH_PERIOD.BROJ_KVARTALA`, celobrojno 1–9999.
2. `technical_report_templates.sql` — kreira `UGO_DOC_TEMPLATE` i `UGO_TEH_IZVESTAJ`.
3. Objaviti novu verziju backenda i frontenda zajedno. Novi backend očekuje obe migracije.

Oracle DDL automatski potvrđuje promene. Ako skripta stane, pregledati grešku i stanje baze pre ponavljanja; ROLLBACK ne poništava već izvršen DDL. Skripte nisu automatski pokrenute na poslovnoj bazi.

## PDF na Ubuntu serveru

Instalirati LibreOffice Writer i fontove korišćene u ugovornim obrascima. U lokalni `app.env` na serveru dodati putanju do izvršnog programa, na primer:

```dotenv
LIBREOFFICE_PATH=/usr/bin/libreoffice
```

Postojeći `DOCUMENT_ROOT` ostaje lokacija dokumenata. Nema promene DB konekcije niti drugih poverljivih podešavanja. Ne dodavati stvarni `app.env` u Git.

Backend korisnik mora imati pristup izvršnom programu, privremenom direktorijumu i `DOCUMENT_ROOT`. Konverzija koristi zaseban privremeni LibreOffice profil po zahtevu, najviše dva istovremena procesa po backend instanci i rok od 60 sekundi. Poziva se direktno, bez komandnog shell-a. Izlaz se proverava pre čuvanja.

Bez `LIBREOFFICE_PATH` moguće je generisati DOCX; PDF opcija je onemogućena. Ako je putanja podešena, ali konverter ne radi, zahtev za DOCX+PDF prijavljuje grešku i ne menja kvartal niti stvara uspešan zapis izveštaja. Može se izabrati samo DOCX.

Zvanična dokumentacija parametara konverzije: https://help.libreoffice.org/latest/en-GB/text/shared/guide/start_parameters.html

Stvarna PDF konverzija i vizuelna podudarnost s Wordom moraju se proveriti na serveru: LibreOffice nije instaliran u razvojnom okruženju. Posebno proveriti ćirilicu, fontove, tabele i prelome stranica.

## Priprema i učitavanje šablona

U Wordu ostaviti stalni tekst po ugovoru, uključujući RUN broj, podatke partnera i potpisnika. Na mestima za popunjavanje uneti običan tekst oznake:

| Oznaka | Izvor |
|---|---|
| `{{datum_od}}` | Početak izabranog tehničkog perioda |
| `{{datum_do}}` | Kraj izabranog tehničkog perioda |
| `{{datum_potpisa}}` | Postojeći datum potpisa perioda |
| `{{broj_kvartala}}` | Potvrđen broj kvartala |
| `{{sysdate}}` | Kompatibilna stara oznaka za datum potpisa, nikada današnji datum |

Period od/do i datum potpisa (jedna od dve oznake datuma) moraju postojati u šablonu. Kvartal je opcion u tekstu šablona. Datumi se ispisuju kao `dd.mm.yyyy` bez dodatne završne tačke; interpunkciju dodati u samom šablonu po potrebi.

Oznake rade u običnim Word pasusima, tabelama, zaglavlju i podnožju, uključujući oznake koje je Word podelio na više tekstualnih delova zbog stilova. Nepoznate oznake se odbijaju. Učitava se DOCX do 10 MB, bez makroa, ugrađenih objekata, spoljnih veza i Word polja; takve elemente treba zameniti običnim tekstom. Izgled i statični delovi dokumenta ostaju u šablonu.

Na detaljima ugovora otvoriti **Šabloni tehničke podrške**, izabrati tip (potvrda ili zapisnik) i učitati fajl. Svako učitavanje pravi novu, neizmenjivu verziju. Za generisanje se koristi najnovija verzija za ugovor i tip dokumenta izabranog perioda. Stare verzije mogu se preuzeti; već generisani dokumenti se ne menjaju.

## Generisanje

1. U **Izmeni period** uneti datum potpisa ako nedostaje. Tu se može uneti i opcioni broj kvartala.
2. U meniju reda izabrati **Generiši / preuzmi izveštaj**.
3. Ako kvartal nije sačuvan, aplikacija predlaže broj ranijih završenih perioda tog ugovora + 1. Izabrani period i kasniji periodi se ne broje. Korisnik može promeniti predlog (npr. ako istorijska evidencija nije potpuna).
4. Izabrati da li je potreban i PDF, pa generisati. DOCX se uvek čuva.
5. Preuzeti dokumente iz istorije tog perioda. Potpisan primerak se naknadno učitava kroz postojeće **Dodaj dokument**.

Generisanje čuva kvartal i uvećava verziju perioda, ali ne menja status niti datum potpisa. Datum potpisa se nikada ne izmišlja ili automatski postavlja. Generisan dokument ne predstavlja potvrdu da je zaista potpisan.

Izveštaji se vode odvojeno od učitanih/potpisanih dokumenata u `UGO_DOC`. `UGO_TEH_IZVESTAJ` čuva korišćenu verziju šablona i neizmenjive vrednosti perioda, kvartala i potpisa. Nove generacije dodaju novi zapis; ne prepisuju starije. Period sa generisanim izveštajima ne može se fizički obrisati, radi očuvanja istorije; može se urediti/otkazati prema pravilima aplikacije.

## Prava, zaključavanje i čuvanje

- GLE dozvoljava pregled/preuzimanje za dostupne OJ. IUD i otvoren ugovor potrebni su za učitavanje šablona i generisanje. ADMIN sam po sebi ne daje pristup ugovoru.
- Generisanje ponovo proverava verziju perioda, datum potpisa i najnoviji šablon u transakciji. Aktivno uređivanje održavanja u drugom tabu blokira upis. Promene tokom konverzije uzrokuju konflikt i zahtevaju osvežavanje, bez prepisivanja tuđih izmena.
- Šabloni su u podfolderu `sabloni`, a izveštaji u `izvestaji/<id_perioda>` postojećeg foldera ugovora pod `DOCUMENT_ROOT`. Imena fajlova na disku su nasumična. Preuzimanje ide samo kroz autentifikovan API sa proverom OJ.
- Backup mora obuhvatiti bazu i `DOCUMENT_ROOT` zajedno.
- Ako odgovor baze pri upisu nije pouzdan, fajl se zadržava da se ne bi obrisao fajl za potvrđen zapis. Takvi retki prekidi mogu ostaviti nereferencirane fajlove; ne prikazuju se korisnicima. Ovaj tok ne koristi `doc-reconcile` za obične upload dokumente i nema automatsko brisanje takvih ostataka.

## Provere u razvoju

Go testovi pokrivaju zamenu oznaka kroz više Word delova, očuvanje XML strukture, odbijanje nepodržanih šablona, čuvanje/preuzimanje izveštaja, konflikt verzije/šablona, zaključavanje i zabranu pristupa drugoj OJ. Browser testovi koriste simulirani API. Provereno je popunjavanje priloženog IPS DOCX šablona bez menjanja originala. Oracle migracije i stvarna LibreOffice konverzija zahtevaju proveru na ciljnom okruženju.
