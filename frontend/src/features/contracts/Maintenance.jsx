import { useContactSession } from '../partners/useContactSession';
import { maintenanceService } from './maintenanceAPI';
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
 const owner = useContactSession(api, maintenanceService);
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
 async function start(mode) {
  if (pending.current || editor) return;
  if (mode !== 'type' && !query.data.document_types?.some(type => ['POTVRDA_TEH', 'ZAPISNIK_TEH'].includes(type.sifra))) { toast.error('Nedostaju aktivni tipovi dokumenata tehničke podrške.'); return; }
  setBlocked(false);
  if(mode==='type'){setEditor({mode,id_tip_ugovora:settings.id_tip_ugovora||'',version:settings.version});return;}
  let source=null, currentSettings=settings;
  if(mode==='edit'){
   pending.current=true;setBusy(true);
   try { const session=await owner.acquire(item.id_ugo_evid,'edit');if(!session)return;source=session.person;currentSettings=source.settings; }
   finally {pending.current=false;setBusy(false);}
  }
  const months=source?.teh_period_meseci ?? (source ? null : 1);
  const period=months===null?'custom':[1,3,6,12].includes(months)?String(months):'other';
  setEditor({mode,version:source?.version,pocetak_odrzavanja:source?.pocetak_odrzavanja||'',overrideStart:!!source?.pocetak_odrzavanja,
   ima_razvojnu_podrsku:source?.ima_razvojnu_podrsku||false,period,teh_period_meseci:months===null?'':String(months),
   id_teh_tip_doc:source?.id_teh_tip_doc||'',teh_podsetnik_dana:source?.teh_podsetnik_dana??7,raz_podsetnik_dana:source?.raz_podsetnik_dana??3,
   fondovi:(currentSettings.vrste_rada||[]).map(type=>({id_vrsta_rada:type.id,ugovoreno_casova:source?.fondovi?.find(f=>f.id_vrsta_rada===type.id)?.ugovoreno_casova||0})),
   settings:currentSettings});
 }
 async function discard(){setBusy(true);await owner.release();setEditor(null);setConfirmClose(false);setBusy(false);}
 function close() { if (!pending.current) setConfirmClose(true); }
 function change(key, value) { setEditor(current => ({ ...current, [key]: value })); }
 async function submit(event) {
  event.preventDefault(); if (pending.current || blocked || owner.lost) return;
  if(editor.mode!=='type'){
   const start=editor.overrideStart?editor.pocetak_odrzavanja:editor.settings.sap_pocetak;
   if(!start || !editor.settings.sap_kraj || start>editor.settings.sap_kraj){toast.error('Proverite početak održavanja i kraj ugovora u SAP-u.');return;}
   if(editor.ima_razvojnu_podrsku && !editor.fondovi.some(f=>Number(f.ugovoreno_casova)>0)){toast.error('Unesite ugovorene časove za bar jednu vrstu rada.');return;}
  }
  pending.current = true; setBusy(true);
  try {
   if(editor.mode==='edit' && !await owner.renew())return;
   const body = editor.mode === 'type' ? { id_tip_ugovora: Number(editor.id_tip_ugovora), version: editor.version } : {
    pocetak_odrzavanja:editor.overrideStart?editor.pocetak_odrzavanja:'',
    ima_razvojnu_podrsku:editor.ima_razvojnu_podrsku,
    teh_period_meseci:editor.period==='custom'?null:Number(editor.period==='other'?editor.teh_period_meseci:editor.period),
    id_teh_tip_doc:Number(editor.id_teh_tip_doc),teh_podsetnik_dana:Number(editor.teh_podsetnik_dana),raz_podsetnik_dana:Number(editor.raz_podsetnik_dana),
    fondovi:editor.fondovi.map(f=>({...f,ugovoreno_casova:editor.ima_razvojnu_podrsku?Number(f.ugovoreno_casova):0})),
    ...(editor.mode==='edit'?{version:editor.version,lock_token:owner.session.token}:{})
   };
   await api(`/ugo_evid/${item.id_ugo_evid}/${editor.mode === 'type' ? 'tip' : 'odrzavanje'}`, { method: editor.mode === 'maintenance' ? 'POST' : 'PUT', body });
   owner.committed();
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
    <div><dt>Početak održavanja / kraj ugovora</dt><dd>{maintenance.pocetak_odrzavanja || settings.sap_pocetak || 'Nedostaje u SAP-u'} / {settings.sap_kraj || 'Nedostaje u SAP-u'}</dd></div>
    {(maintenance.datum_potpisa || maintenance.datum_kraja) && <div><dt>Ranije uneti datumi (za proveru)</dt><dd>{maintenance.datum_potpisa || '—'} / {maintenance.datum_kraja || '—'}</dd></div>}
    {maintenance.ima_razvojnu_podrsku && settings.vrste_rada?.map(type=><div key={type.id}><dt>{type.naziv} — ugovoreni časovi</dt><dd>{maintenance.fondovi?.find(f=>f.id_vrsta_rada===type.id)?.ugovoreno_casova||0}</dd></div>)}
    <div><dt>Podrška</dt><dd>{maintenance.ima_razvojnu_podrsku ? 'Tehnička i razvojna' : 'Tehnička'}</dd></div>
    <div><dt>Period tehničke podrške</dt><dd>{maintenance.teh_period_meseci ? `${maintenance.teh_period_meseci} meseci` : 'Posebno ugovoreni periodi'}</dd></div>
    <div><dt>Dokument tehničke podrške</dt><dd>{query.data.document_types?.find(type => type.id === maintenance.id_teh_tip_doc)?.naziv || 'Neaktivan tip dokumenta'}</dd></div>
    <div><dt>Podsetnik za tehničku podršku</dt><dd>{maintenance.teh_podsetnik_dana} dana pre završetka perioda</dd></div>
    {maintenance.ima_razvojnu_podrsku && <div><dt>Podsetnik za razvojnu podršku</dt><dd>{maintenance.raz_podsetnik_dana} dana pre roka</dd></div>}
   </dl>}
   {canWrite && maintenance && <button className="button" disabled={busy} onClick={()=>start('edit')}>Izmeni održavanje</button>}
  </>}
  {editor && <DocumentDialog title={editor.mode === 'type' ? 'Tip ugovora' : editor.mode === 'edit' ? 'Izmena održavanja' : 'Unos održavanja'} onClose={close} busy={busy}>
   <form className="contact-editor" aria-label="Podešavanja ugovora" onSubmit={submit}><fieldset disabled={busy || blocked || owner.lost}>
    {editor.mode === 'type' ? <label>Tip ugovora<select aria-label="Tip ugovora" required value={editor.id_tip_ugovora} onChange={e => change('id_tip_ugovora', e.target.value)}><option value="">Izaberite tip</option>{types.data?.items?.map(type => <option key={type.id} value={type.id}>{type.naziv}</option>)}</select></label> : <>
     <p>Važenje ugovora iz SAP-a: <strong>{editor.settings.sap_pocetak || 'Početak nije upisan'} — {editor.settings.sap_kraj || 'Kraj nije upisan'}</strong></p>
     <label>Početak održavanja<select aria-label="Početak održavanja" value={String(editor.overrideStart)} onChange={e=>change('overrideStart',e.target.value==='true')}><option value="false">Preuzmi početak važenja iz SAP-a</option><option value="true">Drugi datum početka održavanja</option></select></label>
     {editor.overrideStart && <label>Drugi datum početka<input aria-label="Drugi datum početka" type="date" required max={editor.settings.sap_kraj||undefined} value={editor.pocetak_odrzavanja} onChange={e=>change('pocetak_odrzavanja',e.target.value)}/></label>}
     <label>Razvojna podrška<select aria-label="Razvojna podrška" value={String(editor.ima_razvojnu_podrsku)} onChange={e => change('ima_razvojnu_podrsku', e.target.value === 'true')}><option value="false">Ne</option><option value="true">Da</option></select></label>
     <label>Period potvrđivanja tehničke podrške<select aria-label="Period potvrđivanja tehničke podrške" value={editor.period} onChange={e=>change('period',e.target.value)}><option value="1">Mesečno</option><option value="3">Kvartalno</option><option value="6">Polugodišnje</option><option value="12">Godišnje</option><option value="other">Drugi pravilni period</option><option value="custom">Posebno ugovoreni periodi</option></select></label>
     {editor.period==='other' && <label>Broj meseci<input aria-label="Broj meseci" required type="number" min="1" max="120" step="1" value={editor.teh_period_meseci} onChange={e=>change('teh_period_meseci',e.target.value)}/></label>}
     {editor.ima_razvojnu_podrsku && editor.settings.vrste_rada?.map(type=><label key={type.id}>{type.naziv} — ugovoreni časovi<input aria-label={`${type.naziv} — ugovoreni časovi`} required type="number" min="0" max="2147483647" step="1" value={editor.fondovi.find(f=>f.id_vrsta_rada===type.id)?.ugovoreno_casova??0} onChange={e=>change('fondovi',editor.fondovi.map(f=>f.id_vrsta_rada===type.id?{...f,ugovoreno_casova:e.target.value}:f))}/></label>)}
     <label>Dokument tehničke podrške<select aria-label="Dokument tehničke podrške" required value={editor.id_teh_tip_doc} onChange={e => change('id_teh_tip_doc', e.target.value)}><option value="">Izaberite dokument</option>{query.data.document_types?.filter(type => ['POTVRDA_TEH', 'ZAPISNIK_TEH'].includes(type.sifra)).map(type => <option key={type.id} value={type.id}>{type.naziv}</option>)}</select></label>
     <label>Podsetnik za tehničku podršku (dana ranije)<input aria-label="Podsetnik za tehničku podršku (dana ranije)" required type="number" min="0" max="365" step="1" value={editor.teh_podsetnik_dana} onChange={e => change('teh_podsetnik_dana', e.target.value)}/></label>
     {editor.ima_razvojnu_podrsku && <label>Podsetnik za razvojnu podršku (dana ranije)<input aria-label="Podsetnik za razvojnu podršku (dana ranije)" required type="number" min="0" max="365" step="1" value={editor.raz_podsetnik_dana} onChange={e => change('raz_podsetnik_dana', e.target.value)}/></label>}
    </>}
   </fieldset>
   {(blocked || owner.lost) && <p>Proverite osvežene podatke. Zatvorite prozor pre novog pokušaja.</p>}
   <div className="contact-actions"><button type="button" className="button" disabled={busy} onClick={close}>Odustani</button><button className="button primary" disabled={busy || blocked || owner.lost}>Sačuvaj podešavanja</button></div></form>
   {confirmClose && <ConfirmAction modal title="Odustati od unosa?" disabled={busy} onCancel={() => setConfirmClose(false)} onConfirm={discard}><p>Nesačuvani podaci biće izgubljeni.</p></ConfirmAction>}
  </DocumentDialog>}
 </section>;
}
