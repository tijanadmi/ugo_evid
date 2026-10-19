# Dokumenta tehničkih perioda

Jednom kao TED izvršiti `technical_period_documents.sql`, nakon
`development_request_documents.sql`. Za vreme migracije aplikacija treba da bude zaustavljena.

`UGO_DOC.ID_TEH_PERIOD` povezuje svaki novi dokument sa periodom iz čijeg reda
je pokrenut upload. Moguće je imati više dokumenata za period. Upload proverava
pripadnost ugovoru, postojanje perioda, tip dokumenta i zaključavanje održavanja.
Tip potvrde/zapisnika se automatski preuzima iz perioda.

U tabeli perioda postoje Upload i Prikaži. Prikaz uključuje nove dokumente i
ranije povezani `UGO_TEH_PERIOD.ID_UGO_DOC`. Ručni izbor dokumenata uklonjen je
iz dijaloga perioda i razvojnih zahteva. Izmene njihovih podataka više ne pišu
stare kolone `ID_UGO_DOC` i `ID_ZAHTEV_DOC`, pa postojeće veze ostaju očuvane.

Brisanje i dalje važi samo za planirani tehnički period. FK ON DELETE SET NULL
uklanja vezu novih dokumenata sa obrisanim periodom, ali fajlovi i metapodaci ostaju
u Dokumentima ugovora. Razvojni zahtevi zadržavaju ranija pravila brisanja.
