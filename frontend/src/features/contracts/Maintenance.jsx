import { useRef, useState } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import toast from 'react-hot-toast';
import { useAuth } from '../../context/useAuth';
import { announceBusinessChange } from '../../services/DataSync';
import DocumentDialog from '../documents/DocumentDialog';
import ConfirmAction from '../../ui/ConfirmAction';

export default function Maintenance({ item }) {
 const { api, user } = useAuth();
 const client = useQueryClient();
 const [editor, setEditor] = useState(null);
 const [busy, setBusy] = useState(false);
 const pending = useRef(false);
 const [confirmClose, setConfirmClose] = useState(false);
 const [blocked, setBlocked] = useState(false);
 const query = useQuery({ queryKey: ['contract-settings', user.username, item.id_ugo_evid], queryFn: ({ signal }) => api(`/ugo_evid/${item.id_ugo_evid}/podesavanja`, { signal }), retry: false, refetchOnWindowFocus: true, refetchInterval: 30000 });
 const types = useQuery({ queryKey: ['contract-types', user.username], queryFn: ({ signal }) => api('/ugo_tip_ugovora', { signal }), retry: false });
 const settings = query.data?.settings;
 const currentType = types.data?.items?.find(type => type.id === settings?.id_tip_ugovora);
 const maintenance = settings?.odrzavanje;
 const canWrite = item.otvoren_ug === 'X' && settings && !query.error && !types.error;
 function start(mode) {
  if (mode !== 'type' && !query.data.document_types?.some(type => ['POTVRDA_TEH', 'ZAPISNIK_TEH'].includes(type.sifra))) { toast.error('Nedostaju aktivni tipovi dokumenata tehničke podrške.'); return; }
  setBlocked(false);
  setEditor(mode === 'type' ? { mode, id_tip_ugovora: settings.id_tip_ugovora || '', version: settings.version } : { mode, datum_potpisa: '', datum_kraja: '', ima_razvojnu_podrsku: false, teh_period_meseci: '1', id_teh_tip_doc: '', teh_podsetnik_dana: 7, raz_podsetnik_dana: 3 });
 }
 function close() { if (!pending.current) setConfirmClose(true); }
 function change(key, value) { setEditor(current => ({ ...current, [key]: value })); }
 async function submit(event) {
  event.preventDefault(); if (pending.current || blocked) return;
  if (editor.mode !== 'type' && editor.datum_kraja < editor.datum_potpisa) { toast.error('Datum kraja ne sme biti pre datuma potpisa.'); return; }
  pending.current = true; setBusy(true);
  try {
   const body = editor.mode === 'type' ? { id_tip_ugovora: Number(editor.id_tip_ugovora), version: editor.version } : { datum_potpisa: editor.datum_potpisa, datum_kraja: editor.datum_kraja, ima_razvojnu_podrsku: editor.ima_razvojnu_podrsku, teh_period_meseci: editor.teh_period_meseci === '' ? null : Number(editor.teh_period_meseci), id_teh_tip_doc: Number(editor.id_teh_tip_doc), teh_podsetnik_dana: Number(editor.teh_podsetnik_dana), raz_podsetnik_dana: Number(editor.raz_podsetnik_dana) };
   await api(`/ugo_evid/${item.id_ugo_evid}/${editor.mode === 'type' ? 'tip' : 'odrzavanje'}`, { method: editor.mode === 'type' ? 'PUT' : 'POST', body });
   setEditor(null); toast.success('Podaci su sačuvani.'); announceBusinessChange(client);
  } catch (error) { toast.error(error.message); if ([0, 404, 409, 423].includes(error.status) || error.status >= 500) { setBlocked(true); query.refetch(); } }
  finally { pending.current = false; setBusy(false); }
 }
 return <section className="contract-card maintenance-panel" aria-label="Tip ugovora i održavanje">
  <h2>Tip ugovora i održavanje</h2>
  {query.isPending || types.isPending ? <p role="status">Učitavanje podešavanja...</p> : query.error || types.error ? <><p>Podešavanja nisu dostupna.</p><button className="button" onClick={() => { query.refetch(); types.refetch(); }}>Osveži podešavanja</button></> : <>
   <p>Tip ugovora: <strong>{currentType?.naziv || (settings?.id_tip_ugovora ? 'Neaktivan tip' : 'Nije određen')}</strong></p>
   {canWrite && !maintenance && <button className="button" onClick={() => start('type')}>Odredi tip ugovora</button>}
   {currentType?.sifra === 'ODRZAVANJE' && !maintenance && <><p>Osnovni podaci održavanja još nisu uneti.</p>{canWrite && <button className="button primary" onClick={() => start('maintenance')}>Unesi održavanje</button>}</>}
   {maintenance && <dl className="maintenance-summary">
    <div><dt>Datum potpisa / kraj održavanja</dt><dd>{maintenance.datum_potpisa} / {maintenance.datum_kraja}</dd></div>
    <div><dt>Podrška</dt><dd>{maintenance.ima_razvojnu_podrsku ? 'Tehnička i razvojna' : 'Tehnička'}</dd></div>
    <div><dt>Period tehničke podrške</dt><dd>{maintenance.teh_period_meseci ? `${maintenance.teh_period_meseci} meseci` : 'Posebno ugovoreni periodi'}</dd></div>
    <div><dt>Dokument tehničke podrške</dt><dd>{query.data.document_types?.find(type => type.id === maintenance.id_teh_tip_doc)?.naziv || 'Neaktivan tip dokumenta'}</dd></div>
    <div><dt>Podsetnik za tehničku podršku</dt><dd>{maintenance.teh_podsetnik_dana} dana pre završetka perioda</dd></div>
    {maintenance.ima_razvojnu_podrsku && <div><dt>Podsetnik za razvojnu podršku</dt><dd>{maintenance.raz_podsetnik_dana} dana pre roka</dd></div>}
   </dl>}
  </>}
  {editor && <DocumentDialog title={editor.mode === 'type' ? 'Tip ugovora' : 'Unos održavanja'} onClose={close} busy={busy}>
   <form className="contact-editor" aria-label="Podešavanja ugovora" onSubmit={submit}><fieldset disabled={busy || blocked}>
    {editor.mode === 'type' ? <label>Tip ugovora<select aria-label="Tip ugovora" required value={editor.id_tip_ugovora} onChange={e => change('id_tip_ugovora', e.target.value)}><option value="">Izaberite tip</option>{types.data?.items?.map(type => <option key={type.id} value={type.id}>{type.naziv}</option>)}</select></label> : <>
     <label>Datum potpisa ugovora<input aria-label="Datum potpisa ugovora" required type="date" value={editor.datum_potpisa} onChange={e => change('datum_potpisa', e.target.value)}/></label>
     <label>Datum kraja održavanja<input aria-label="Datum kraja održavanja" required type="date" min={editor.datum_potpisa} value={editor.datum_kraja} onChange={e => change('datum_kraja', e.target.value)}/></label>
     <label>Razvojna podrška<select aria-label="Razvojna podrška" value={String(editor.ima_razvojnu_podrsku)} onChange={e => change('ima_razvojnu_podrsku', e.target.value === 'true')}><option value="false">Ne</option><option value="true">Da</option></select></label>
     <label>Period tehničke podrške (meseci)<input aria-label="Period tehničke podrške (meseci)" type="number" min="1" max="120" step="1" value={editor.teh_period_meseci} onChange={e => change('teh_period_meseci', e.target.value)}/><small>1 mesečno, 3 kvartalno, 12 godišnje. Prazno za posebno ugovorene periode.</small></label>
     <label>Dokument tehničke podrške<select aria-label="Dokument tehničke podrške" required value={editor.id_teh_tip_doc} onChange={e => change('id_teh_tip_doc', e.target.value)}><option value="">Izaberite dokument</option>{query.data.document_types?.filter(type => ['POTVRDA_TEH', 'ZAPISNIK_TEH'].includes(type.sifra)).map(type => <option key={type.id} value={type.id}>{type.naziv}</option>)}</select></label>
     <label>Podsetnik za tehničku podršku (dana ranije)<input aria-label="Podsetnik za tehničku podršku (dana ranije)" required type="number" min="0" max="365" step="1" value={editor.teh_podsetnik_dana} onChange={e => change('teh_podsetnik_dana', e.target.value)}/></label>
     {editor.ima_razvojnu_podrsku && <label>Podsetnik za razvojnu podršku (dana ranije)<input aria-label="Podsetnik za razvojnu podršku (dana ranije)" required type="number" min="0" max="365" step="1" value={editor.raz_podsetnik_dana} onChange={e => change('raz_podsetnik_dana', e.target.value)}/></label>}
    </>}
   </fieldset>
   {blocked && <p>Proverite osvežene podatke. Zatvorite prozor pre novog pokušaja.</p>}
   <div className="contact-actions"><button type="button" className="button" disabled={busy} onClick={close}>Odustani</button><button className="button primary" disabled={busy || blocked}>Sačuvaj podešavanja</button></div></form>
   {confirmClose && <ConfirmAction modal title="Odustati od unosa?" disabled={busy} onCancel={() => setConfirmClose(false)} onConfirm={() => { setEditor(null); setConfirmClose(false); }}><p>Nesačuvani podaci biće izgubljeni.</p></ConfirmAction>}
  </DocumentDialog>}
 </section>;
}
