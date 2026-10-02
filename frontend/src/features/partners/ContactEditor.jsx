import { useEffect, useRef, useState } from 'react';
import toast from 'react-hot-toast';
import { useAuth } from '../../context/AuthContext';
import { getContactRoles, saveContact } from '../../services/apiContacts';

const empty = { ime: '', radno_mesto: '', telefon: '', email: '', id_ugo_dob_lica_rola: 0, status: 'A' };
const toDraft = person => ({ ime: person.ime || '', radno_mesto: person.radno_mesto || '', telefon: person.telefon || '', email: person.email || '', id_ugo_dob_lica_rola: person.ugo_dob_lica_rola?.id || 0, status: person.status || 'A' });

export default function ContactEditor({ supplierID, personID, session, leaseLost, beforeWrite, onReload, onCancel, onSaved, onBusy }) {
  const { api } = useAuth();
  const [draft, setDraft] = useState(() => session ? toDraft(session.person) : empty);
  const [roles, setRoles] = useState([]);
  const [ready, setReady] = useState(false);
  const [blocked, setBlocked] = useState(false);
  const [busy, setBusy] = useState(false);
  const working = useRef(false);
  const lost = blocked || leaseLost || (personID && !session);
  useEffect(() => {
    if (session) { setDraft(toDraft(session.person)); setBlocked(false); }
  }, [session]);
  useEffect(() => {
    let stopped = false;
    getContactRoles(api).then(items => { if (!stopped) { setRoles(items); setReady(true); } }).catch(error => { if (!stopped) toast.error(error.message, { id: 'operation-error' }); });
    return () => { stopped = true; };
  }, [api]);
  async function submit(event) {
    event.preventDefault();
    if (working.current || !ready || lost) return;
    working.current = true; setBusy(true); onBusy(true);
    try {
      if (personID && !await beforeWrite()) return;
      await saveContact(api, personID, { ...draft, id_sap_dobavljac: supplierID, ...(personID ? { version: session.person.version, lock_token: session.token } : {}) });
      onSaved();
    } catch (error) {
      toast.error(error.message, { id: 'operation-error' });
      if ([0, 409, 423, 404].includes(error.status) || error.status >= 500) setBlocked(true);
    } finally { working.current = false; setBusy(false); onBusy(false); }
  }
  async function reload() {
    if (!window.confirm('Učitavanje će zameniti unos u formi aktuelnim podacima. Nastaviti?')) return;
    setBusy(true); onBusy(true);
    try { await onReload(); } finally { setBusy(false); onBusy(false); }
  }
  return <form className="contact-editor" onSubmit={submit} aria-label={personID ? 'Izmena lica' : 'Dodavanje lica'}>
    <h3>{personID ? 'Izmeni lice' : 'Dodaj lice'}</h3>
    {!ready && <p role="status">Učitavanje podataka…</p>}
    <fieldset disabled={!ready || busy}>
      <label>Ime i prezime<input name="ime" required maxLength={100} value={draft.ime} onChange={e => setDraft({ ...draft, ime: e.target.value })}/></label>
      <label>Radno mesto<input name="radno_mesto" maxLength={200} value={draft.radno_mesto} onChange={e => setDraft({ ...draft, radno_mesto: e.target.value })}/></label>
      <label>Telefon<input name="telefon" type="tel" maxLength={100} value={draft.telefon} onChange={e => setDraft({ ...draft, telefon: e.target.value })}/></label>
      <label>Email<input name="email" type="email" maxLength={100} value={draft.email} onChange={e => setDraft({ ...draft, email: e.target.value })}/></label>
      <label>Rola<select value={draft.id_ugo_dob_lica_rola} onChange={e => setDraft({ ...draft, id_ugo_dob_lica_rola: Number(e.target.value) })}>
        <option value={0}>Bez role</option>{draft.id_ugo_dob_lica_rola !== 0 && !roles.some(role => role.id === draft.id_ugo_dob_lica_rola) && <option value={draft.id_ugo_dob_lica_rola}>Trenutna rola</option>}{roles.map(role => <option key={role.id} value={role.id}>{role.naziv}</option>)}
      </select></label>
      <label>Status<select value={draft.status} onChange={e => setDraft({ ...draft, status: e.target.value })}><option value="A">Aktivan</option><option value="N">Neaktivan</option>{!['A', 'N'].includes(draft.status) && <option value={draft.status}>{draft.status}</option>}</select></label>
    </fieldset>
    {lost && personID && <button className="button" type="button" disabled={busy} onClick={reload}>Učitaj aktuelne podatke</button>}
    <div className="contact-actions"><button className="button" type="button" disabled={busy} onClick={onCancel}>Odustani</button><button className="button primary" type="submit" disabled={!ready || lost || busy}>{busy ? 'Obrada…' : 'Sačuvaj'}</button></div>
  </form>;
}
