import { formatDate } from '../../utils/contractFormatting';
import { useQuery } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import { useAuth } from '../../context/useAuth';

export default function Reminders({contractID}){
 const {api,user}=useAuth();
 const query=useQuery({queryKey:['maintenance-reminders',user.username],queryFn:({signal})=>api('/podsetnici',{signal}),retry:false,refetchInterval:60000,refetchOnWindowFocus:true});
 const items=(query.data?.items||[]).filter(x=>!contractID||x.id_ugo_evid===contractID);
 return <section className="contract-card" aria-label="Podsetnici za održavanje"><h2>Podsetnici za održavanje</h2>
  {!contractID && <p className="muted">Obaveze za ugovore vaše organizacione jedinice.</p>}
  {query.isPending?<p>Učitavanje podsetnika...</p>:query.error?<p>Podsetnici trenutno nisu dostupni. <button className="button" onClick={()=>query.refetch()}>Osveži podsetnike</button></p>:items.length?<ul>{items.map(p=><li key={`${p.vrsta}-${p.id}`}><Link to={`/ugovori/detalji/${p.id_ugo_evid}`}>Ugovor #{p.id_ugo_evid}</Link> — {p.vrsta==='TEH'?'Tehnička podrška':'Razvojni zahtev'}: {p.opis}; {p.vrsta==='RAZ'?'potpisati pre':'period / rok'} {formatDate(p.rok)}{p.kasni && <strong> — Rok je istekao</strong>}</li>)}</ul>:<p>Nema dospelih podsetnika.</p>}
 </section>;
}
