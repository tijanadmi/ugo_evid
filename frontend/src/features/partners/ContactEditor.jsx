import { useQuery } from "@tanstack/react-query";
import ConfirmAction from "../../ui/ConfirmAction";
import { useEffect, useRef, useState } from "react";
import toast from "react-hot-toast";
import { useAuth } from "../../context/useAuth";
import { saveContact } from "../../services/apiContacts";

const empty = {
  ime: "",
  radno_mesto: "",
  telefon: "",
  email: "",
  status: "A",
};
const toDraft = (person) => ({
  ime: person.ime || "",
  radno_mesto: person.radno_mesto || "",
  telefon: person.telefon || "",
  email: person.email || "",
  status: person.status || "A",
});

export default function ContactEditor({
  supplierID,
  personID,
  session,
  leaseLost,
  beforeWrite,
  onReload,
  onCancel,
  onSaved,
  onBusy,
  submitLabel = "Sačuvaj",
}) {
  const { api, user } = useAuth();
  const [draft, setDraft] = useState(() =>
    session ? toDraft(session.person) : { ...empty },
  );
  const engagements = useQuery({
    queryKey: ["contact-engagements", user.username, personID],
    queryFn: ({ signal }) =>
      api(`/ugo_dob_lica/${personID}/angazovanja`, { signal }),
    enabled: !!personID && draft.status !== "A",
    retry: false,
  });
  const [blocked, setBlocked] = useState(false);
  const [confirmReload, setConfirmReload] = useState(false);
  const [busy, setBusy] = useState(false);
  const working = useRef(false);
  const lost = blocked || leaseLost || (personID && !session);
  useEffect(() => {
    if (session) {
      setDraft(toDraft(session.person));
      setBlocked(false);
    }
  }, [session]);
  async function submit(event) {
    event.preventDefault();
    if (working.current || lost) return;
    working.current = true;
    setBusy(true);
    onBusy(true);
    try {
      if (personID && !(await beforeWrite())) return;
      const result = await saveContact(api, personID, {
        ...draft,
        id_sap_dobavljac: supplierID,
        ...(personID
          ? { version: session.person.version, lock_token: session.token }
          : {}),
      });
      await onSaved(result?.data);
    } catch (error) {
      toast.error(error.message, { id: "operation-error" });
      if ([0, 409, 423, 404].includes(error.status) || error.status >= 500)
        setBlocked(true);
    } finally {
      working.current = false;
      setBusy(false);
      onBusy(false);
    }
  }
  async function reload() {
    setBusy(true);
    onBusy(true);
    try {
      await onReload();
    } finally {
      setBusy(false);
      onBusy(false);
      setConfirmReload(false);
    }
  }
  return (
    <form
      className="contact-editor"
      onSubmit={submit}
      aria-label={personID ? "Izmena lica" : "Dodavanje lica"}
    >
      {confirmReload && (
        <ConfirmAction
          modal
          title="Učitati aktuelne podatke?"
          disabled={busy}
          onCancel={() => setConfirmReload(false)}
          onConfirm={reload}
        >
          <p>
            Učitavanje će zameniti unos u formi aktuelnim podacima. Nastaviti?
          </p>
        </ConfirmAction>
      )}
      <h3>{personID ? "Izmeni lice" : "Dodaj lice"}</h3>
      <fieldset disabled={busy}>
        <label>
          Ime i prezime
          <input
            name="ime"
            required
            maxLength={100}
            value={draft.ime}
            onChange={(e) => setDraft({ ...draft, ime: e.target.value })}
          />
        </label>
        <label>
          Radno mesto
          <input
            name="radno_mesto"
            maxLength={200}
            value={draft.radno_mesto}
            onChange={(e) =>
              setDraft({ ...draft, radno_mesto: e.target.value })
            }
          />
        </label>
        <label>
          Telefon
          <input
            name="telefon"
            type="tel"
            maxLength={100}
            value={draft.telefon}
            onChange={(e) => setDraft({ ...draft, telefon: e.target.value })}
          />
        </label>
        <label>
          Email
          <input
            name="email"
            type="email"
            maxLength={100}
            value={draft.email}
            onChange={(e) => setDraft({ ...draft, email: e.target.value })}
          />
        </label>
        <label>
          Status
          <select
            value={draft.status}
            onChange={(e) => setDraft({ ...draft, status: e.target.value })}
          >
            <option value="A">Aktivan</option>
            <option value="N">Neaktivan</option>
            {!["A", "N"].includes(draft.status) && (
              <option value={draft.status}>{draft.status}</option>
            )}
          </select>
        </label>
      </fieldset>
      {lost && personID && (
        <button
          className="button"
          type="button"
          disabled={busy}
          onClick={() => setConfirmReload(true)}
        >
          Učitaj aktuelne podatke
        </button>
      )}
      {personID && draft.status !== "A" && (
        <section>
          <p>
            Pasivizacija važi za imenik dobavljača i sva angažovanja. Lice se
            više ne nudi za nove ugovore; postojeće veze ostaju vidljive radi
            izbora zamene.
          </p>
          {engagements.isPending && <p>Provera otvorenih ugovora…</p>}
          {engagements.error && (
            <p role="alert">
              Provera angažovanja nije uspela. Proverite ugovore pre
              pasivizacije.
            </p>
          )}
          {!!engagements.data?.items?.length && (
            <>
              <p>
                Otvoreni ugovori dostupni vašim pravima na kojima je lice
                angažovano:
              </p>
              <ul>
                {engagements.data.items.map((p, i) => (
                  <li key={`${p.id_ugo_evid}-${i}`}>
                    {p.br_ugovor} — {p.predmet_ugovora} ({p.rola})
                  </li>
                ))}
              </ul>
            </>
          )}
        </section>
      )}
      <div className="contact-actions">
        <button
          className="button"
          type="button"
          disabled={busy}
          onClick={onCancel}
        >
          Odustani
        </button>
        <button
          className="button primary"
          type="submit"
          disabled={lost || busy}
        >
          {busy ? "Obrada…" : submitLabel}
        </button>
      </div>
    </form>
  );
}
