# Administracija portala

## Primena

1. Napraviti rezervnu kopiju baze i zaustaviti instance backenda tokom migracije.
2. Pre ove migracije moraju biti primenjeni `access_roles.sql` i `document_deletion.sql`. Ne pokretati ponovo već primenjene skripte.
3. Kao korisnik TED pokrenuti `portal_admin.sql` jednom. Oracle DDL automatski potvrđuje promene: ako skripta stane, pregledati stanje i grešku pre nastavka, jer ROLLBACK ne poništava prethodni DDL.
4. U lokalnoj kopiji `bootstrap_admin.sql` zameniti placeholder postojećim aktivnim AD nalogom iz UGO_KOR. Pokrenuti skriptu. Ona odbija izvršavanje ako već postoji aktivan ADMIN i beleži dodelu u istoriji. Postojeća GLE/IUD prava korisnika ostaju sačuvana. Ne dodavati stvarni nalog ili poverljive konfiguracije u Git.
5. Objaviti backend i frontend ove verzije zajedno, pokrenuti backend i ponovo se prijaviti. U zaglavlju se pojavljuje **Administracija**.

Skripte nisu izvršene na poslovnoj Oracle bazi iz razvojnog okruženja. Testovi koriste simuliranu bazu/API i privremene fajlove; potrebno je proveriti migraciju i unos korisnika na testnoj Oracle bazi.

## Prava i korisnici

- ADMIN upravlja korisnicima, njihovim pravima, radnim kalendarom i administratorskim obaveštenjima. ADM_KALENDAR ostaje pravo samo za kalendar.
- ADMIN ne daje automatski pristup ugovorima. GLE/IUD se posebno dodeljuju za pojedinačnu OJ ili sve OJ.
- Novi korisnik je postojeći AD nalog kome se omogućava pristup portalu. Aplikacija ne kreira nalog u Active Directory niti upravlja lozinkama. AD identitet proverava se prilikom prijave.
- AD korisničko ime postojećeg korisnika nije izmenjivo. Korisnik se deaktivira, ne briše. Forma prikazuje efektivna prava i dozvoljava do 50 dodela po korisniku.
- Verzija korisnika sprečava prepisivanje promena iz drugog taba. Posle konflikta potrebno je ponovo učitati podatke.
- Sve administratorske izmene korisnika i prava koriste zajednički zaključan slog PORTAL_ADMIN_LOCK, uključujući ponovnu proveru prava izvršioca i zabranu uklanjanja poslednjeg aktivnog administratora. Promena i audit upisuju se u istoj transakciji.
- Zaštita poslednjeg administratora važi za API i bootstrap skriptu; proizvoljne direktne SQL izmene mogu je zaobići.
- Endpoint `/users` takođe zahteva ADMIN; `/users/prava` i dalje vraća prava trenutno prijavljenog korisnika.

## Obaveštenja

Brisanje se pokušava odmah, pri pokretanju backenda i potom svakog minuta. Ako brisanje i nakon 15 minuta od zahteva ne uspeva, nastaje jedno trajno obaveštenje za dokument. Sledeći neuspesi ažuriraju isti zapis, poslednju grešku i brojače; ne stvaraju nova obaveštenja svakog minuta.

Administracija prikazuje naziv dokumenta, ID ugovora, broj pokušaja i detalje greške. Administrator može preuzeti problem, a onaj ko ga je preuzeo može ga vratiti u otvorene. Preuzimanje ne zaustavlja automatske pokušaje brisanja. Uzrok poput prava pristupa direktorijumu rešava se na serveru; aplikacija ne menja sistemska prava.

Uspešno brisanje automatski označava obaveštenje kao rešeno. Problem dokumenta koji još čeka brisanje ne može se ručno proglasiti rešenim. Rešena obaveštenja i istorija ostaju dostupni. Broj nerešenih obaveštenja u zaglavlju i lista osvežavaju se na 30 sekundi dok je aplikacija otvorena.

Email slanje nije deo ove izmene. Evidencija obaveštenja nastaje u bazi i kada administrator nije prijavljen; vidi je pri sledećem otvaranju portala.
