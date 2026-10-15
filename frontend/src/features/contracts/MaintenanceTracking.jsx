import { Link } from 'react-router-dom';
import RowMenu from '../../ui/RowMenu';
import { useRef, useState } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import toast from 'react-hot-toast';
import { useAuth } from '../../context/useAuth';
import { useContactSession } from '../partners/useContactSession';
import { maintenanceService } from './maintenanceAPI';
import { announceBusinessChange } from '../../services/DataSync';
import DocumentDialog from '../documents/DocumentDialog';
import ConfirmAction from '../../ui/ConfirmAction';
import Reminders from './Reminders';
import './maintenanceTracking.css';

const labels = { PLANIRAN:'Planiran', PRIPREMA:'U pripremi', NA_POTPISU:'Na potpisu', ZAVRSEN:'Završen', OTKAZAN:'Otkazan', NACRT:'Nacrt', PODNET:'Podnet', U_RADU:'U radu' };
const empty = { id:0, version:'0', period_od:'',period_do:'',id_tip_doc:0,id_vrsta_rada:0,broj_zahteva:'',datum_zahteva:'',broj_casova:1,rok_radnih_dana:1,rok:'',datum_potpisa:'',id_ugo_doc:0,id_zahtev_doc:0,napomena:'' };
export default function MaintenanceTracking({ item, settings, documentTypes }) {
 const {api,user}=useAuth();const client=useQueryClient();const owner=useContactSession(api,maintenanceService);
 const [editor,setEditor]=useState(null),[busy,setBusy]=useState(false),[confirm,setConfirm]=useState(false),[blocked,setBlocked]=useState(false);
 const pending=useRef(false),id=item.id_ugo_evid;
 const query=useQuery({queryKey:['maintenance-tracking',user.username,id],queryFn:({signal})=>api(`/ugo_evid/${id}/pracenje`,{signal}),retry:false,refetchInterval:30000,refetchOnWindowFocus:true});
 const documents=useQuery({queryKey:['documents',user.username,id],queryFn:({signal})=>api(`/ugo_evid/${id}/dokumenti`,{signal}),retry:false});
 const data=query.data,canWrite=item.otvoren_ug==='X' && !!data && !query.error;
 async function open(kind,row,generate=false,remove=false){
  if(pending.current || editor)return;pending.current=true;setBusy(true);setBlocked(false);
  try {
   const session=await owner.acquire(id,'edit');if(!session)return;
   const fresh=await api(`/ugo_evid/${id}/pracenje`);client.setQueryData(['maintenance-tracking',user.username,id],fresh);
   const source=row?fresh[kind].find(x=>x.id===row.id):null;if(row && !source)throw new Error('Slog više nije dostupan.');
   if(remove && source?.status!=='PLANIRAN')throw new Error('Samo period sa statusom Planiran može da se obriše.');
   const current=session.person.settings;
   setEditor({kind,generate,remove,mode:[1,3,6,12].includes(current.odrzavanje.teh_period_meseci)?'calendar':'anchor',preview:null,settings:current,row:source||{...empty,status:kind==='periodi'?'PLANIRAN':'NACRT',id_tip_doc:current.odrzavanje.id_teh_tip_doc}});
  }catch(e){toast.error(e.message);await owner.release();}finally{pending.current=false;setBusy(false);}
 }
 function change(key,value){setEditor(s=>({...s,row:{...s.row,[key]:value}}));}
 async function discard(){setBusy(true);await owner.release();setEditor(null);setConfirm(false);setBusy(false);}
 async function preview(){
  if(pending.current)return;pending.current=true;setBusy(true);
  try{if(!await owner.renew())return;const result=await api(`/ugo_evid/${id}/pracenje/preview`,{method:'POST',body:{mode:editor.mode}});setEditor(s=>({...s,preview:result.items}));}
  catch(e){toast.error(e.message);}finally{pending.current=false;setBusy(false);}
 }
 async function save(event){
  event?.preventDefault();if(pending.current || owner.lost || blocked)return;
  pending.current=true;setBusy(true);
  try{
   if(!await owner.renew())return;
   await api(`/ugo_evid/${id}/pracenje/${editor.kind}`,{method:editor.remove?'DELETE':'POST',body:{lock_token:owner.session.token,items:editor.generate?editor.preview:[editor.row]}});
   owner.committed();setEditor(null);toast.success('Podaci su sačuvani.');announceBusinessChange(client);
  }catch(e){toast.error(e.message);if([0,404,409,423].includes(e.status)||e.status>=500)setBlocked(true);}
  finally{pending.current=false;setBusy(false);}
 }
 const periods=data?.periodi||[];const active=periods.filter(p=>p.status!=='OTKAZAN').sort((a,b)=>a.period_od.localeCompare(b.period_od));
 const start=settings.odrzavanje.pocetak_odrzavanja||settings.sap_pocetak;
 const hasGap=active.length>0 && (active[0].period_od>start || active.at(-1).period_do<settings.sap_kraj || active.some((p,i)=>i>0 && (new Date(p.period_od+'T00:00:00Z')-new Date(active[i-1].period_do+'T00:00:00Z'))>86400000));
 function docSelect(key,label){return <label>{label}<select value={editor.row[key]} onChange={e=>change(key,Number(e.target.value))}><option value="0">Bez povezanog dokumenta</option>{documents.data?.items?.map(d=><option key={d.id} value={d.id}>{d.naziv}</option>)}</select></label>;}
 return <section className="maintenance-tracking" aria-label="Praćenje održavanja">
  <h2>Praćenje održavanja</h2>
  {query.isPending?<p>Učitavanje perioda i zahteva...</p>:query.error?<p role="alert">Praćenje nije dostupno. Proverite da li je izvršen maintenance_tracking.sql. <button className="button" onClick={()=>query.refetch()}>Pokušaj ponovo</button></p>:<>
   <h3>Tehnički periodi</h3>
   {canWrite && <div className="tracking-actions"><button className="button" disabled={busy} onClick={()=>open('periodi')}>Dodaj period</button><button className="button" disabled={busy || periods.length>0 || !settings.odrzavanje.teh_period_meseci} onClick={()=>open('periodi',null,true)}>Generiši periode</button></div>}
   <p className="muted">Generisanje je dostupno dok lista nema perioda. Postojeći periodi se menjaju pojedinačno. Otkazani periodi ostaju u evidenciji.</p>
   {hasGap && <p role="status">Postoje nepokriveni delovi trajanja održavanja. Proverite datume perioda.</p>}
   <div className="tracking-table"><table><thead><tr><th>Radnja</th><th>Period</th><th>Status</th><th>Rok dokumenta</th><th>Potpisano</th></tr></thead><tbody>{periods.map(p=><tr key={p.id}><td>{canWrite && <RowMenu label={`Akcije za period ${p.period_od} - ${p.period_do}`} actions={[{label:'Izmeni period',icon:'edit',onClick:()=>open('periodi',p)},...(p.status==='PLANIRAN'?[{label:'Obriši period',icon:'trash',danger:true,onClick:()=>open('periodi',p,false,true)}]:[])]}/>}</td><td>{p.period_od} — {p.period_do}</td><td>{labels[p.status]}</td><td>{p.rok||'Nije određen'}</td><td>{p.datum_potpisa||'—'}</td></tr>)}</tbody></table></div>
   {!periods.length && <p>Nema evidentiranih tehničkih perioda.</p>}
   {settings.odrzavanje.ima_razvojnu_podrsku && <>
    <h3>Razvojna podrška</h3>
    <div className="tracking-table"><table><thead><tr><th>Vrsta rada</th><th>Ugovoreno</th><th>Zauzeto / utrošeno</th><th>Preostalo</th></tr></thead><tbody>{data.fondovi.map(f=><tr key={f.id_vrsta_rada}><td>{f.naziv}</td><td>{f.ugovoreno}</td><td>{f.zauzeto}</td><td>{f.preostalo}</td></tr>)}</tbody></table></div>
    <p className="muted">Nacrt i otkazan zahtev ne zauzimaju časove. Završeni zahtevi ostaju u obračunu.</p>
    {canWrite && <button className="button" disabled={busy || !data.fondovi.length} onClick={()=>open('zahtevi')}>Dodaj razvojni zahtev</button>}
    <div className="tracking-table"><table><thead><tr><th>Radnja</th><th>Zahtev / datum</th><th>Vrsta rada</th><th>Časovi / radni dani</th><th>Potpisati pre</th><th>Status</th></tr></thead><tbody>{data.zahtevi.map(p=><tr key={p.id}><td>{canWrite && <button className="button" disabled={busy} onClick={()=>open('zahtevi',p)}>Izmeni zahtev</button>}</td><td>{p.broj_zahteva}<br/>{p.datum_zahteva}</td><td>{settings.vrste_rada.find(w=>w.id===p.id_vrsta_rada)?.naziv}</td><td>{p.broj_casova} / {p.rok_radnih_dana}</td><td>{p.rok}</td><td>{labels[p.status]}</td></tr>)}</tbody></table></div>
    {!data.zahtevi.length && <p>Nema evidentiranih razvojnih zahteva.</p>}
   </>}
  </>}
  <Reminders contractID={id}/>
  {editor?.remove && <ConfirmAction modal title="Obrisati tehnički period?" confirmLabel="Obriši period" cancelLabel="Odustani" variation="danger" disabled={busy} blocked={blocked||owner.lost} onConfirm={()=>save()} onCancel={discard}>
   <p>Period {editor.row.period_od} — {editor.row.period_do} biće trajno obrisan. Povezani dokument ostaje sačuvan.</p>
   {(blocked||owner.lost) && <p>Pravo brisanja više nije važeće. Zatvorite prozor i osvežite podatke.</p>}
  </ConfirmAction>}
  {editor && !editor.remove && <DocumentDialog title={editor.generate?'Generisanje tehničkih perioda':editor.kind==='periodi'?'Tehnički period':'Razvojni zahtev'} onClose={()=>setConfirm(true)} busy={busy}>
   <form className="contact-editor tracking-editor" onSubmit={save}><fieldset disabled={busy || owner.lost || blocked}>
    {editor.generate?<>
     <label>Način obračuna<select value={editor.mode} onChange={e=>setEditor(s=>({...s,mode:e.target.value,preview:null}))}><option value="calendar" disabled={![1,3,6,12].includes(editor.settings.odrzavanje.teh_period_meseci)}>Kalendarski mesec / kvartal / polugodište / godina</option><option value="anchor">Od datuma početka održavanja</option></select></label>
     <p>Početak: {editor.settings.odrzavanje.pocetak_odrzavanja||editor.settings.sap_pocetak}; kraj: {editor.settings.sap_kraj}. Poslednji period se završava na kraju ugovora.</p>
     <button className="button" type="button" onClick={preview}>Prikaži predlog</button>
     {editor.preview && <div className="tracking-preview"><ol>{editor.preview.map((p,i)=><li key={i}>{p.period_od} — {p.period_do}</li>)}</ol></div>}
    </>:<>
     {editor.kind==='periodi'?<>
      <label>Period od<input required type="date" value={editor.row.period_od} onChange={e=>change('period_od',e.target.value)}/></label>
      <label>Period do<input required type="date" value={editor.row.period_do} onChange={e=>change('period_do',e.target.value)}/></label>
      <label>Tip dokumenta<select required value={editor.row.id_tip_doc} onChange={e=>change('id_tip_doc',Number(e.target.value))}><option value="0">Izaberite tip</option>{documentTypes.filter(t=>['POTVRDA_TEH','ZAPISNIK_TEH'].includes(t.sifra)).map(t=><option key={t.id} value={t.id}>{t.naziv}</option>)}</select></label>
      <label>Rok dokumenta (opciono)<input type="date" value={editor.row.rok} onChange={e=>change('rok',e.target.value)}/></label>
     </>:<>
      <label>Broj zahteva<input required maxLength={100} value={editor.row.broj_zahteva} onChange={e=>change('broj_zahteva',e.target.value)}/></label>
      <label>Datum zahteva<input required type="date" value={editor.row.datum_zahteva} onChange={e=>change('datum_zahteva',e.target.value)}/></label>
      <label>Vrsta rada<select aria-label="Vrsta rada" required value={editor.row.id_vrsta_rada||''} onChange={e=>change('id_vrsta_rada',Number(e.target.value))}><option value="">Izaberite vrstu rada</option>{data.fondovi.map(f=><option key={f.id_vrsta_rada} value={f.id_vrsta_rada}>{f.naziv} — preostalo {f.preostalo} č.</option>)}</select></label>
      <label>Broj časova<input required type="number" min="1" max="2147483647" step="1" value={editor.row.broj_casova} onChange={e=>change('broj_casova',Number(e.target.value))}/></label>
      <label>Rok u radnim danima<input required type="number" min="1" max="3660" step="1" value={editor.row.rok_radnih_dana} onChange={e=>change('rok_radnih_dana',Number(e.target.value))}/></label>
      <p>Rok se računa pri čuvanju iz kalendara radnih dana, počev od narednog dana. Zapisnik mora biti potpisan pre datuma roka.{editor.row.rok && ` Sačuvani rok: ${editor.row.rok}.`}</p>
      <p><Link target="_blank" rel="noopener noreferrer" to={`/ugovori/radni-kalendar${editor.row.datum_zahteva?'?godina='+editor.row.datum_zahteva.slice(0,4):''}`}>Otvori radni kalendar u novom tabu</Link></p>
      {docSelect('id_zahtev_doc','Dokument zahteva')}
     </>}
     <label>Status<select aria-label="Status" value={editor.row.status} onChange={e=>change('status',e.target.value)}>{(editor.kind==='periodi'?['PLANIRAN','PRIPREMA','NA_POTPISU','ZAVRSEN','OTKAZAN']:['NACRT','PODNET','U_RADU','NA_POTPISU','ZAVRSEN','OTKAZAN']).map(s=><option key={s} value={s}>{labels[s]}</option>)}</select></label>
     <label>Datum potpisa<input required={editor.row.status==='ZAVRSEN'} type="date" value={editor.row.datum_potpisa} onChange={e=>change('datum_potpisa',e.target.value)}/></label>
     {docSelect('id_ugo_doc','Potvrda / zapisnik')}
     <p className="muted">Dokument prvo dodajte u odeljku Dokumenta ugovora, zatim ga ovde povežite.</p>
     <label>Napomena<textarea maxLength={2000} value={editor.row.napomena} onChange={e=>change('napomena',e.target.value)}/></label>
    </>}
   </fieldset>
   {(blocked||owner.lost) && <p role="alert">Pravo izmene više nije važeće. Zatvorite prozor i ponovo učitajte podatke.</p>}
   <div className="contact-actions"><button className="button" type="button" disabled={busy} onClick={()=>setConfirm(true)}>Odustani</button><button className="button primary" disabled={busy||blocked||owner.lost||(editor.generate&&!editor.preview?.length)}>Sačuvaj</button></div>
   </form>
   {confirm && <ConfirmAction modal title="Odustati od unosa?" disabled={busy} onCancel={()=>setConfirm(false)} onConfirm={discard}><p>Nesačuvani podaci biće izgubljeni.</p></ConfirmAction>}
  </DocumentDialog>}
 </section>;
}
