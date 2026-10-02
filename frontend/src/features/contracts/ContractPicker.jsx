import toast from 'react-hot-toast';
import { useEffect, useRef, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useAuth } from '../../context/AuthContext';
import Icon from '../../ui/Icon';
import { getUnregisteredContracts, prepareContract, registerContract } from '../../services/apiContractRegistration';

function RegistrationForm({ contract, organizations, onBack, onCreated, onBusy }) {
  const { api, user } = useAuth();
  const query = useQuery({ queryKey: ['contract-preparation', user.username, contract.id], queryFn: ({ signal }) => prepareContract(api, contract.id, signal), retry: false, staleTime: 0, refetchOnWindowFocus: false });
  const [draft, setDraft] = useState({ ime: '', radno_mesto: '', telefon: '', email: '' });
  const [saving, setSaving] = useState(false);
  const [mustReload, setMustReload] = useState(false);
  const submitting = useRef(false);
  const data = query.data;
  const person = data?.contact;
  const org = organizations.find(item => item.id === data?.id_ugo_org);
  async function save(event) {
    event.preventDefault();
    if (!data || query.isFetching || mustReload || submitting.current) return;
    submitting.current = true;
    setSaving(true);
    onBusy(true);
    try {
      const response = await registerContract(api, { id_sap_ugovor: contract.id, ...(person ? { id_ugo_dob_lica: person.id, contact_version: person.version } : { novo_lice: draft }) });
      onCreated(response.data);
    } catch (failure) {
      toast.error(failure.message, { id: 'operation-error' });
      // A lost response can hide a committed insert. Recheck before another write.
      setMustReload(true);
    } finally { submitting.current = false; setSaving(false); onBusy(false); }
  }
  async function reload() {
    const result = await query.refetch();
    if (!result.isError) { setMustReload(false); }
  }
  return <form className="contact-editor contract-registration-form" aria-label="Unos ugovora" onSubmit={save}>
    <div><h3>Ugovor {contract.br_ugovor}</h3><p>{contract.predmet_ugovora}</p><p className="muted">{contract.dobavljac}</p></div>
    {query.isFetching && <p role="status">Provera ugovora i kontakt lica…</p>}
    {data && !query.error && <>
      <p>Evidencija se dodaje za organizacionu jedinicu: <strong>{org ? `${org.sifra} — ${org.naziv}` : data.id_ugo_org}</strong>.</p>
      {data.contract.otvoren_ug !== 'X' && <p className="muted">Izabrani ugovor je zatvoren i biće prikazan među zatvorenim ugovorima.</p>}
      {person ? <section className="registration-contact" aria-label="Postojeće kontakt lice"><h3>Aktivno kontakt lice sa rolom 1</h3><strong>{person.ime}</strong><span>{person.radno_mesto}</span><span>{person.telefon || 'Telefon nije upisan'}</span><span>{person.email || 'Email nije upisan'}</span><p className="muted">Ovi podaci biće preuzeti u evidenciju ugovora.</p></section> : <>
        <p>Dobavljač nema aktivno lice sa rolom 1. Unesite kontakt; lice i evidencija ugovora biće sačuvani zajedno.</p>
        <fieldset disabled={saving || query.isFetching}>
          <label>Ime i prezime<input required maxLength={100} value={draft.ime} onChange={e => setDraft({ ...draft, ime: e.target.value })}/></label>
          <label>Radno mesto (opciono)<input maxLength={200} value={draft.radno_mesto} onChange={e => setDraft({ ...draft, radno_mesto: e.target.value })}/></label>
          <label>Telefon<input required type="tel" maxLength={100} value={draft.telefon} onChange={e => setDraft({ ...draft, telefon: e.target.value })}/></label>
          <label>Email<input required type="email" maxLength={100} value={draft.email} onChange={e => setDraft({ ...draft, email: e.target.value })}/></label>
        </fieldset>
      </>}
    </>}
    {(query.error || mustReload) && <button type="button" className="button" disabled={saving || query.isFetching} onClick={reload}>Ponovo proveri ugovor</button>}
    <div className="contact-actions"><button type="button" className="button" disabled={saving} onClick={onBack}>Nazad na izbor</button><button type="submit" className="button primary" disabled={!data || !!query.error || query.isFetching || saving || mustReload}>{saving ? 'Čuvanje…' : 'Evidentiraj ugovor'}</button></div>
  </form>;
}

export default function ContractPicker({ organizations, onClose, onCreated }) {
  const { api, user } = useAuth();
  const dialog = useRef(null);
  const [draftFilter, setDraftFilter] = useState('');
  const [filter, setFilter] = useState('');
  const [page, setPage] = useState(1);
  const [selected, setSelected] = useState(null);
  const [busy, setBusy] = useState(false);
  const query = useQuery({ queryKey: ['unregistered-contracts', user.username, page, filter], queryFn: ({ signal }) => getUnregisteredContracts(api, page, filter, signal), enabled: !selected, retry: false });
  useEffect(() => { const element = dialog.current; element.showModal(); return () => element.close(); }, []);
  function close() {
    if (busy) return;
    if (selected && !window.confirm('Odustati od unosa ugovora?')) return;
    onClose();
  }
  const pages = Math.max(1, Math.ceil((query.data?.total || 0) / 20));
  return <dialog ref={dialog} className="detail-dialog contract-picker" aria-labelledby="contract-picker-title" onCancel={event => { event.preventDefault(); close(); }} onClick={event => { if (event.target === dialog.current) close(); }}>
    <div className="detail-header"><h2 id="contract-picker-title">Izaberi ugovor</h2><button type="button" className="icon-button" disabled={busy} aria-label="Zatvori izbor ugovora" onClick={close}><Icon name="close"/></button></div>
    {selected ? <RegistrationForm key={selected.id} contract={selected} organizations={organizations} onBusy={setBusy} onCreated={onCreated} onBack={() => { setSelected(null); query.refetch(); }}/> : <div className="contract-picker-body">
      <form className="partner-filter" onSubmit={event => { event.preventDefault(); setFilter(draftFilter.trim()); setPage(1); }}>
        <label htmlFor="contract-picker-filter">Broj ugovora, predmet ili dobavljač</label><div className="partner-filter-controls"><input id="contract-picker-filter" type="search" maxLength={200} value={draftFilter} onChange={event => setDraftFilter(event.target.value)} placeholder="Pretraži SAP ugovore…"/><button className="button" type="submit">Pretraži</button></div>
      </form>
      <p className="muted">Prikazani su ugovori koji još nisu evidentirani.</p>
      {query.isFetching ? <p role="status">Učitavanje ugovora…</p> : query.error ? <div className="contact-error">Pregled nije dostupan. <button className="button" onClick={() => query.refetch()}>Pokušaj ponovo</button></div> : query.data?.items?.length ? <div className="table-scroll"><table className="contract-picker-table"><thead><tr><th>Broj / godina</th><th>Predmet</th><th>Dobavljač</th><th>Status</th><th><span className="sr-only">Izbor</span></th></tr></thead><tbody>{query.data.items.map(item => <tr key={item.id}><th scope="row">{item.br_ugovor}<small>{item.godina}</small></th><td>{item.predmet_ugovora}</td><td>{item.dobavljac}</td><td>{item.otvoren_ug === 'X' ? 'Otvoren' : 'Zatvoren'}</td><td><button type="button" className="button" aria-label={`Izaberi ugovor ${item.br_ugovor}`} onClick={() => setSelected(item)}>Izaberi</button></td></tr>)}</tbody></table></div> : <p>Nema neevidentiranih ugovora za izabrani filter.</p>}
      <nav className="pagination" aria-label="Stranice izbora ugovora"><span>Stranica {page} od {pages}</span><button type="button" className="button" disabled={page <= 1 || query.isFetching} onClick={() => setPage(value => value - 1)}>Prethodna</button><button type="button" className="button" disabled={page >= pages || query.isFetching || !!query.error} onClick={() => setPage(value => value + 1)}>Sledeća</button></nav>
    </div>}
  </dialog>;
}
