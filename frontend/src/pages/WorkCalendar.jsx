import { useRef, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import toast from 'react-hot-toast';
import { useAuth } from '../context/useAuth';
import ConfirmAction from '../ui/ConfirmAction';
import DocumentDialog from '../features/documents/DocumentDialog';
import { announceBusinessChange } from '../services/DataSync';
import './WorkCalendar.css';

const months=['Januar','Februar','Mart','April','Maj','Jun','Jul','Avgust','Septembar','Oktobar','Novembar','Decembar'];
const weekdays=['Pon','Uto','Sre','Čet','Pet','Sub','Ned'];
const iso=(year,month,day)=>`${year}-${String(month).padStart(2,'0')}-${String(day).padStart(2,'0')}`;
function localToday(){const p=new Intl.DateTimeFormat('en',{year:'numeric',month:'2-digit',day:'2-digit',timeZone:'Europe/Belgrade'}).formatToParts(new Date());return Object.fromEntries(p.filter(x=>x.type!=='literal').map(x=>[x.type,Number(x.value)]));}

export default function WorkCalendar(){
 const {api,user}=useAuth(),client=useQueryClient();const [params,setParams]=useSearchParams();const today=localToday();
 const selectedYear=Number(params.get('godina')),selectedMonth=Number(params.get('mesec'));
 const year=Number.isInteger(selectedYear)&&selectedYear>=1900&&selectedYear<=2200?selectedYear:today.year;
 const month=Number.isInteger(selectedMonth)&&selectedMonth>=1&&selectedMonth<=12?selectedMonth:params.has('godina')?1:today.month;
 const [draft,setDraft]=useState(null),[action,setAction]=useState(null),[busy,setBusy]=useState(false),[blocked,setBlocked]=useState(false),[preview,setPreview]=useState(null),[cancelDraft,setCancelDraft]=useState(false),[selection,setSelection]=useState('');
 const pending=useRef(false),queryKey=['work-calendar',user.username,year];
 const query=useQuery({queryKey,queryFn:({signal})=>api(`/radni_kalendar/${year}`,{signal}),retry:false,refetchOnWindowFocus:true});
 const items=query.data?.items||[],byDate=new Map(items.map(d=>[d.datum,d])),total=(Date.UTC(year+1,0,1)-Date.UTC(year,0,1))/86400000,approved=items.filter(d=>d.potvrdjen==='D').length;
 const canWrite=!!query.data&&!query.error&&!busy,hasDialog=!!draft||!!action||!!preview;
 const first=(new Date(Date.UTC(year,month-1,1)).getUTCDay()+6)%7,days=new Date(Date.UTC(year,month,0)).getUTCDate();
 const selectedDate=selection.startsWith(iso(year,month,1).slice(0,7))?selection:iso(year,month,1);
 const selectedDay=byDate.get(selectedDate);
 function navigate(y,m){if(y<1900||y>2200||pending.current||hasDialog)return;setParams({godina:String(y),mesec:String(m)});setBlocked(false);}
 function shift(delta){const d=new Date(Date.UTC(year,month-1+delta,1));navigate(d.getUTCFullYear(),d.getUTCMonth()+1);}
 function edit(day){setBlocked(false);setDraft({action:'change',version:query.data.version,od:day?.datum||selectedDate,do:day?.datum||selectedDate,radni:day?.radni||'N',opis:day?.opis||''});}
 async function submit(change,optimisticDay){
  if(pending.current)return;pending.current=true;setBusy(true);const before=query.data;
  // Cancel an in-flight focus refresh before presenting the optimistic color.
  await client.cancelQueries({queryKey});
  if(optimisticDay)client.setQueryData(queryKey,{...before,items:before.items.map(d=>d.datum===optimisticDay?{...d,radni:change.radni,potvrdjen:'N',izvor:'RUCNO'}:d)});
  try{await api(`/radni_kalendar/${year}`,{method:'POST',body:change});setDraft(null);setAction(null);setPreview(null);toast.success(change.action==='import'?'Praznici su uvezeni. Proverite i potvrdite godinu.':'Kalendar je sačuvan.',{id:'calendar-save'});announceBusinessChange(client);await query.refetch();}
  catch(e){if(optimisticDay)client.setQueryData(queryKey,before);toast.error(e.message);if(e.status===409){setBlocked(true);setAction(null);setPreview(null);await query.refetch();}}
  finally{pending.current=false;setBusy(false);}
 }
 function toggle(day){if(!canWrite||pending.current||!day)return;setSelection(day.datum);submit({action:'change',version:query.data.version,od:day.datum,do:day.datum,radni:day.radni==='D'?'N':'D',opis:day.opis||''},day.datum);}
 async function download(){
  if(!canWrite||pending.current)return;pending.current=true;setBusy(true);const snapshot=query.data;
  try{const result=await api(`/radni_kalendar/${year}/praznici`);setPreview({version:snapshot.version,rows:result.items.map(h=>{const d=snapshot.items.find(x=>x.datum===h.datum);const protectedDay=d?.izvor==='RUCNO';return {...h,protectedDay,selected:!protectedDay};})});}
  catch(e){toast.error(e.message);}finally{pending.current=false;setBusy(false);}
 }
 return <section className="contract-card work-calendar" aria-label="Radni kalendar">
  <header className="calendar-heading"><div><h1>Radni kalendar</h1><p>Klik na dan menja radni status i čuva izmenu.</p></div><label>Godina<input aria-label="Godina" type="number" min="1900" max="2200" value={year} disabled={busy||hasDialog} onChange={e=>{const value=Number(e.target.value);if(value>=1900&&value<=2200)navigate(value,month);}}/></label></header>
  <div className="calendar-tools">
   <button className="button" disabled={!canWrite} onClick={()=>submit({action:'generate',version:query.data.version})}>Generiši kalendar</button>
   <button className="button" disabled={!canWrite||items.length!==total} onClick={download}>Preuzmi praznike za Srbiju</button>
   <button className="button" disabled={!canWrite||!items.length} onClick={()=>edit(selectedDay)}>Izmeni dane</button>
   <button className="button primary" disabled={!canWrite||items.length!==total||approved===total} onClick={()=>setAction({action:'approve',version:query.data.version})}>Potvrdi godinu</button>
  </div>
  <div className="calendar-month-nav"><button className="button" aria-label="Prethodni mesec" disabled={busy||hasDialog||(year===1900&&month===1)} onClick={()=>shift(-1)}>←</button><label><span className="sr-only">Mesec</span><select aria-label="Mesec" disabled={busy||hasDialog} value={month} onChange={e=>navigate(year,Number(e.target.value))}>{months.map((m,i)=><option key={m} value={i+1}>{m}</option>)}</select></label><h2>{year}.</h2><button className="button" aria-label="Sledeći mesec" disabled={busy||hasDialog||(year===2200&&month===12)} onClick={()=>shift(1)}>→</button><button className="button calendar-today" disabled={busy||hasDialog} onClick={()=>navigate(today.year,today.month)}>Danas</button></div>
  <div className="calendar-legend"><span><i className="working"/>Radni dan</span><span><i className="nonworking"/>Neradni dan / vikend</span><span><i className="unapproved"/>Čeka potvrdu</span></div>
  {query.isPending?<p role="status">Učitavanje kalendara...</p>:query.error?<p role="alert">Kalendar nije dostupan. <button className="button" onClick={()=>query.refetch()}>Osveži</button></p>:<>
   <p className="calendar-status" role="status">Generisano {items.length} od {total} dana. Potvrđeno {approved} od {total} dana.{busy?' Čuvanje / preuzimanje…':items.length<total?' Generišite godinu da biste uređivali sve dane.':''}</p>
   <div className="calendar-weekdays" aria-hidden="true">{weekdays.map((d,i)=><span className={i>4?'weekend':''} key={d}>{d}</span>)}</div>
   <div className="calendar-month" aria-label={`${months[month-1]} ${year}`}>
    {Array.from({length:42},(_,index)=>{const n=index-first+1;if(n<1||n>days)return <div key={index} className="calendar-empty" aria-hidden="true"/>;const key=iso(year,month,n),day=byDate.get(key),working=day?.radni==='D',isToday=key===iso(today.year,today.month,today.day);return <button key={index} type="button" className={`calendar-day ${day?working?'working':'nonworking':'missing'}${key===selectedDate?' selected':''}${isToday?' today':''}`} disabled={!canWrite||!day} aria-label={`${key}, ${day?working?'radni dan':'neradni dan':'nije generisan'}${day?.opis?', '+day.opis:''}`} aria-pressed={!!day&&!working} aria-current={isToday?'date':undefined} title={day?`${day.opis||key} · ${day.potvrdjen==='D'?'Potvrđen':'Čeka potvrdu'} · Klik za promenu statusa`:'Prvo generišite kalendar'} onClick={()=>toggle(day)}>
     <span className="calendar-day-top"><strong>{n}</strong>{day?.potvrdjen==='N'&&<span className="calendar-pending" aria-label="Čeka potvrdu"/>}</span>
     <span className="calendar-day-state">{day?working?'Radni':'Neradni':'Nije generisan'}</span>
     <span className="calendar-day-description">{day?.opis}</span>
     {day?.izvor==='RUCNO'&&<span className="calendar-day-source">Ručna izmena</span>}
    </button>;})}
   </div>
   <footer className="calendar-footer"><span><strong>{selectedDate}</strong> · {selectedDay?.opis||'Bez opisa'}</span><button className="button" disabled={!canWrite||!selectedDay} onClick={()=>edit(selectedDay)}>Opis / korekcija</button></footer>
  </>}
  {action&&<ConfirmAction modal title={`Potvrditi kalendar za ${year}. godinu?`} confirmLabel="Potvrdi" disabled={busy} onCancel={()=>setAction(null)} onConfirm={()=>submit(action)}><p>Potvrđujem da su provereni praznici i izuzetno radni dani. Sačuvani rokovi zahteva ostaju nepromenjeni.</p></ConfirmAction>}
  {preview&&<DocumentDialog title={`Praznici Srbije — ${year}.`} busy={busy} onClose={()=>setPreview(null)}>
   <div className="calendar-import"><p>Predlog: Nager.Date, uz dopunu vaskršnjeg perioda i pravila prenosa državnih praznika. Proverite prema <a href="https://www.nbs.rs/sr/o_nbs/radno-vreme-i-neradni-dani/" target="_blank" rel="noreferrer">zvaničnoj objavi</a>. Uvoz ne potvrđuje godinu.</p><p>Ručne korekcije nisu označene za uvoz. Označi ih samo ako želiš da ih zameniš. Uvoze se samo izabrani datumi.</p>
    <ul className="calendar-holiday-list">{preview.rows.map(h=><li key={h.datum}><label><input type="checkbox" disabled={busy} checked={h.selected} onChange={e=>setPreview(p=>({...p,rows:p.rows.map(x=>x.datum===h.datum?{...x,selected:e.target.checked}:x)}))}/><span><strong>{h.datum} — {h.naziv}</strong><small>{h.protectedDay?'Ručna korekcija — označavanjem odobravaš zamenu':h.poreklo}</small></span></label></li>)}</ul>
    <div className="contact-actions"><button className="button" disabled={busy} onClick={()=>setPreview(null)}>Odustani</button><button className="button primary" disabled={busy||!preview.rows.some(h=>h.selected)} onClick={()=>submit({action:'import',version:preview.version,praznici:preview.rows.filter(h=>h.selected).map(({datum,naziv,protectedDay})=>({datum,naziv,zameni_rucno:protectedDay}))})}>Uvezi izabrane praznike</button></div>
   </div>
  </DocumentDialog>}
  {draft&&<DocumentDialog title="Izmena radnih dana" busy={busy} onClose={()=>setCancelDraft(true)}><form className="contact-editor" onSubmit={e=>{e.preventDefault();submit(draft);}}><fieldset disabled={busy||blocked}>
   <label>Od datuma<input type="date" required min={`${year}-01-01`} max={`${year}-12-31`} value={draft.od} onChange={e=>setDraft({...draft,od:e.target.value})}/></label>
   <label>Do datuma<input type="date" required min={draft.od} max={`${year}-12-31`} value={draft.do} onChange={e=>setDraft({...draft,do:e.target.value})}/></label>
   <label>Radni dan<select aria-label="Radni dan" value={draft.radni} onChange={e=>setDraft({...draft,radni:e.target.value})}><option value="N">Ne — neradni dan</option><option value="D">Da — radni dan</option></select></label>
   <label>Opis<input maxLength={200} value={draft.opis} onChange={e=>setDraft({...draft,opis:e.target.value})}/></label>
   <p>Promena važi za sve datume u opsegu. Ručne izmene su zaštićene od uvoza praznika i traže ponovnu potvrdu.</p>
  </fieldset>{blocked&&<p role="alert">Kalendar je promenjen. Zatvorite prozor i ponovo otvorite izmenu.</p>}<div className="contact-actions"><button type="button" className="button" disabled={busy} onClick={()=>setCancelDraft(true)}>Odustani</button><button className="button primary" disabled={busy||blocked}>Sačuvaj dane</button></div></form>
   {cancelDraft&&<ConfirmAction modal title="Odustati od izmena?" disabled={busy} onCancel={()=>setCancelDraft(false)} onConfirm={()=>{setDraft(null);setCancelDraft(false);}}><p>Nesačuvane izmene biće izgubljene.</p></ConfirmAction>}
  </DocumentDialog>}
 </section>;
}
