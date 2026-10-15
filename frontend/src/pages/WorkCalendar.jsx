import { useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import toast from 'react-hot-toast';
import { useAuth } from '../context/useAuth';
import ConfirmAction from '../ui/ConfirmAction';
import DocumentDialog from '../features/documents/DocumentDialog';
import { announceBusinessChange } from '../services/DataSync';
import '../features/contracts/maintenanceTracking.css';

export default function WorkCalendar(){
 const {api,user}=useAuth(),client=useQueryClient();const [params,setParams]=useSearchParams();
 const selected=Number(params.get('godina'));const year=Number.isInteger(selected)&&selected>=1900&&selected<=2200?selected:Number(new Intl.DateTimeFormat('en',{year:'numeric',timeZone:'Europe/Belgrade'}).format(new Date()));
 const [draft,setDraft]=useState(null),[action,setAction]=useState(null),[busy,setBusy]=useState(false),[blocked,setBlocked]=useState(false),[filter,setFilter]=useState('all');
 const query=useQuery({queryKey:['work-calendar',user.username,year],queryFn:({signal})=>api(`/radni_kalendar/${year}`,{signal}),retry:false,refetchOnWindowFocus:true});
 const items=query.data?.items||[],total=(Date.UTC(year+1,0,1)-Date.UTC(year,0,1))/86400000,approved=items.filter(d=>d.potvrdjen==='D').length;
 function edit(day){setBlocked(false);setDraft({action:'change',version:query.data.version,od:day?.datum||`${year}-01-01`,do:day?.datum||`${year}-01-01`,radni:day?.radni||'N',opis:day?.opis||''});}
 async function submit(change){
  if(busy)return;setBusy(true);
  try{await api(`/radni_kalendar/${year}`,{method:'POST',body:change});setDraft(null);setAction(null);toast.success('Kalendar je sačuvan.');announceBusinessChange(client);await query.refetch();}
  catch(e){toast.error(e.message);if(e.status===409){setBlocked(true);setAction(null);query.refetch();}}
  finally{setBusy(false);}
 }
 return <section className="contract-card" aria-label="Radni kalendar">
  <h1>Radni kalendar</h1><p>Kalendar je zajednički za sve ugovore. Proverite praznike i izuzetke pre potvrde godine. Sačuvani rokovi razvojnih zahteva ostaju nepromenjeni.</p>
  <label>Godina <input aria-label="Godina" type="number" min="1900" max="2200" value={year} disabled={busy||!!draft||!!action} onChange={e=>{const value=Number(e.target.value);if(value>=1900&&value<=2200){setParams({godina:String(value)});setBlocked(false);}}}/></label>
  {query.isPending?<p>Učitavanje kalendara...</p>:query.error?<p role="alert">Kalendar nije dostupan. <button className="button" onClick={()=>query.refetch()}>Osveži</button></p>:<>
   <p role="status">Generisano {items.length} od {total} dana. Potvrđeno {approved} od {total} dana.</p>
   <div className="tracking-actions">
    <button className="button" disabled={busy} onClick={()=>submit({action:'generate',version:query.data.version})}>Generiši kalendar</button>
    <button className="button" disabled={busy||!items.length} onClick={()=>edit()}>Izmeni dane</button>
    <button className="button primary" disabled={busy||items.length!==total||approved===total} onClick={()=>setAction({action:'approve',version:query.data.version})}>Potvrdi godinu</button>
   </div>
   <p>Generisanje dodaje nedostajuće datume: ponedeljak–petak kao radne, subotu i nedelju kao neradne. Postojeće izmene i potvrde ostaju sačuvane.</p>
   <label>Prikaži <select aria-label="Prikaži dane" value={filter} onChange={e=>setFilter(e.target.value)}><option value="all">Sve dane</option><option value="N">Neradne dane</option><option value="unapproved">Nepotvrđene dane</option></select></label>
   <div className="tracking-table"><table><thead><tr><th>Radnja</th><th>Datum</th><th>Dan</th><th>Radni dan</th><th>Opis</th><th>Potvrđen</th></tr></thead><tbody>{items.filter(d=>filter==='all'||(filter==='N'?d.radni==='N':d.potvrdjen!=='D')).map(d=><tr key={d.datum}><td><button className="button" disabled={busy} aria-label={`Izmeni ${d.datum}`} onClick={()=>edit(d)}>Izmeni</button></td><td>{d.datum}</td><td>{new Intl.DateTimeFormat('sr-Latn',{weekday:'long',timeZone:'UTC'}).format(new Date(d.datum+'T00:00:00Z'))}</td><td>{d.radni==='D'?'Da':'Ne'}</td><td>{d.opis||'—'}</td><td>{d.potvrdjen==='D'?'Da':'Ne'}</td></tr>)}</tbody></table></div>
  </>}
  {action && <ConfirmAction modal title={`Potvrditi kalendar za ${year}. godinu?`} confirmLabel="Potvrdi" disabled={busy} onCancel={()=>setAction(null)} onConfirm={()=>submit(action)}><p>Potvrđujem da su provereni svi praznici i izuzetno radni dani. Kalendar će se koristiti za nove obračune rokova.</p></ConfirmAction>}
  {draft && <DocumentDialog title="Izmena radnih dana" busy={busy} onClose={()=>setDraft(null)}><form className="contact-editor" onSubmit={e=>{e.preventDefault();submit(draft);}}><fieldset disabled={busy||blocked}>
   <label>Od datuma<input type="date" required min={`${year}-01-01`} max={`${year}-12-31`} value={draft.od} onChange={e=>setDraft({...draft,od:e.target.value})}/></label>
   <label>Do datuma<input type="date" required min={draft.od} max={`${year}-12-31`} value={draft.do} onChange={e=>setDraft({...draft,do:e.target.value})}/></label>
   <label>Radni dan<select aria-label="Radni dan" value={draft.radni} onChange={e=>setDraft({...draft,radni:e.target.value})}><option value="N">Ne — neradni dan</option><option value="D">Da — radni dan</option></select></label>
   <label>Opis<input maxLength={200} value={draft.opis} onChange={e=>setDraft({...draft,opis:e.target.value})}/></label>
   <p>Promena važi za sve datume u opsegu, uključujući vikende. Izmenjeni dani postaju nepotvrđeni.</p>
  </fieldset>{blocked&&<p role="alert">Kalendar je promenjen. Zatvorite prozor i ponovo otvorite izmenu.</p>}<div className="contact-actions"><button type="button" className="button" disabled={busy} onClick={()=>setDraft(null)}>Odustani</button><button className="button primary" disabled={busy||blocked}>Sačuvaj dane</button></div></form></DocumentDialog>}
 </section>;
}
