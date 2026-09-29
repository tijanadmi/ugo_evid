import { useEffect, useRef, useState } from 'react';
import { useAuth } from '../../context/AuthContext';
import { acquireContact, renewContact, releaseContact, getContact, getContactRoles, saveContact, deleteContact } from '../../services/apiContacts';

const empty = { ime: '', radno_mesto: '', telefon: '', email: '', id_ugo_dob_lica_rola: 0, status: 'A' };
const toDraft = person => ({ ime: person.ime || '', radno_mesto: person.radno_mesto || '', telefon: person.telefon || '', email: person.email || '', id_ugo_dob_lica_rola: person.ugo_dob_lica_rola?.id || 0, status: person.status || 'A' });

export default function ContactEditor({ supplierID, personID, mode, onCancel, onSaved, onBusy }) {
  const { api } = useAuth();
  const [draft, setDraft] = useState(empty);
  const [roles, setRoles] = useState([]);
  const [ready, setReady] = useState(false);
  const [lost, setLost] = useState(false);
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const [attempt, setAttempt] = useState(0);
  const lease = useRef(null);
  const version = useRef(null);
  const working = useRef(false);
  const deleting = mode === 'delete';

  useEffect(() => {
    let stopped = false;
    let ownedToken = null;
    let timer;
    let renewing = false;
    setReady(false);
    setLost(false);
    setError('');
    const release = token => releaseContact(api, personID, token).catch(() => {});
    async function start() {
      try {
        if (personID) {
          const acquired = await acquireContact(api, personID);
          ownedToken = acquired.lock_token;
          if (stopped) { await release(ownedToken); return; }
          lease.current = ownedToken;
          // Read only after acquiring: the version must belong to this edit session.
          const person = await getContact(api, personID);
          if (stopped) return;
          version.current = person.version;
          setDraft(toDraft(person));
        }
        const available = deleting ? [] : await getContactRoles(api);
        if (stopped) return;
        setRoles(available);
        setReady(true);
        if (personID) {
          timer = setInterval(async () => {
            if (renewing || working.current || !lease.current) return;
            renewing = true;
            try { await renewContact(api, personID, ownedToken); }
            catch (failure) {
              if (!stopped && lease.current === ownedToken) {
                setLost(true);
                setError(`${failure.message} Uneti podaci su sačuvani u formi. Za nastavak učitajte aktuelne podatke.`);
                clearInterval(timer);
              }
            } finally { renewing = false; }
          }, 30000);
        }
      } catch (failure) {
        if (ownedToken) { await release(ownedToken); if (lease.current === ownedToken) lease.current = null; ownedToken = null; }
        if (!stopped) { setError(failure.message); setLost(true); }
      }
    }
    // StrictMode may clean up the first effect before this microtask runs.
    Promise.resolve().then(() => { if (!stopped) start(); });
    return () => {
      stopped = true;
      clearInterval(timer);
      if (ownedToken && lease.current === ownedToken) release(ownedToken);
      if (lease.current === ownedToken) lease.current = null;
    };
  }, [api, personID, deleting, attempt]);

  async function submit(event) {
    event.preventDefault();
    if (working.current || !ready || lost) return;
    working.current = true;
    setBusy(true);
    onBusy(true);
    setError('');
    try {
      if (deleting) await deleteContact(api, personID, version.current, lease.current);
      else await saveContact(api, personID, { ...draft, id_sap_dobavljac: supplierID, ...(personID ? { version: version.current, lock_token: lease.current } : {}) });
      lease.current = null;
      onSaved();
    } catch (failure) {
      setError(failure.message);
      // A failed network response may conceal a successful commit. Never blindly
      // retry a mutation with the same lease or duplicate an insert.
      if ([0, 409, 423, 404].includes(failure.status) || failure.status >= 500) setLost(true);
    } finally {
      working.current = false;
      setBusy(false);
      onBusy(false);
    }
  }

  async function reload() {
    if (!window.confirm('Učitavanje će zameniti unos u formi aktuelnim podacima. Nastaviti?')) return;
    setBusy(true);
    onBusy(true);
    try {
      if (lease.current) await releaseContact(api, personID, lease.current);
      lease.current = null;
      setAttempt(value => value + 1);
    } catch (failure) { setError(failure.message); }
    finally { setBusy(false); onBusy(false); }
  }
  return <form className="contact-editor" onSubmit={submit} aria-label={deleting ? 'Brisanje lica' : personID ? 'Izmena lica' : 'Dodavanje lica'}>
    <h3>{deleting ? 'Brisanje lica' : personID ? 'Izmeni lice' : 'Dodaj lice'}</h3>
    {error && <p className="contact-error" role="alert">{error}</p>}
    {!ready && !error && <p role="status">Učitavanje podataka…</p>}
    {deleting ? ready && <p>Da li želite da obrišete lice <strong>{draft.ime}</strong>? Lice povezano sa ugovorima možete deaktivirati kroz izmenu.</p> : <fieldset disabled={!ready || busy}>
      <label>Ime i prezime<input name="ime" required maxLength={100} value={draft.ime} onChange={e => setDraft({ ...draft, ime: e.target.value })}/></label>
      <label>Radno mesto<input name="radno_mesto" maxLength={200} value={draft.radno_mesto} onChange={e => setDraft({ ...draft, radno_mesto: e.target.value })}/></label>
      <label>Telefon<input name="telefon" type="tel" maxLength={100} value={draft.telefon} onChange={e => setDraft({ ...draft, telefon: e.target.value })}/></label>
      <label>Email<input name="email" type="email" maxLength={100} value={draft.email} onChange={e => setDraft({ ...draft, email: e.target.value })}/></label>
      <label>Rola<select value={draft.id_ugo_dob_lica_rola} onChange={e => setDraft({ ...draft, id_ugo_dob_lica_rola: Number(e.target.value) })}>
        <option value={0}>Bez role</option>{draft.id_ugo_dob_lica_rola !== 0 && !roles.some(role => role.id === draft.id_ugo_dob_lica_rola) && <option value={draft.id_ugo_dob_lica_rola}>Trenutna rola</option>}{roles.map(role => <option key={role.id} value={role.id}>{role.naziv}</option>)}
      </select></label>
      <label>Status<select value={draft.status} onChange={e => setDraft({ ...draft, status: e.target.value })}><option value="A">Aktivan</option><option value="N">Neaktivan</option>{!['A', 'N'].includes(draft.status) && <option value={draft.status}>{draft.status}</option>}</select></label>
    </fieldset>}
    {lost && personID && <button className="button" type="button" disabled={busy} onClick={reload}>Učitaj aktuelne podatke</button>}
    {lost && !personID && <p>Proverite spisak lica pre ponovnog dodavanja.</p>}
    <div className="contact-actions"><button className="button" type="button" disabled={busy} onClick={onCancel}>Odustani</button><button className={`button ${deleting ? 'danger' : 'primary'}`} type="submit" disabled={!ready || lost || busy}>{busy ? 'Obrada…' : deleting ? 'Potvrdi brisanje' : 'Sačuvaj'}</button></div>
  </form>;
}
