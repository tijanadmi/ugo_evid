import { useId, useRef, useState } from 'react';
import toast from 'react-hot-toast';

export default function DocumentForm({ types, initial, upload, maxBytes, onSave, onCancel, busy, blocked }) {
 const fileHelp = useId();
 const [draft, setDraft] = useState(() => ({ id_tip_doc: initial.id_tip_doc || types[0]?.id || 0, naziv: initial.naziv || '', broj_dokumenta: initial.broj_dokumenta || '', datum_dokumenta: initial.datum_dokumenta || '', napomena: initial.napomena || '' }));
 const [file, setFile] = useState(null);
 const submitting = useRef(false);
 function change(key, value) { setDraft(current => ({ ...current, [key]: value })); }
 async function submit(event) {
  event.preventDefault();
  if (submitting.current || busy || blocked) return;
  if (upload && (!file || file.size === 0 || file.size > maxBytes)) { toast.error(`Izaberite fajl do ${Math.round(maxBytes / 1024 / 1024)} MB.`); return; }
  if (!draft.naziv.trim()) { toast.error('Unesite naziv dokumenta.'); return; }
  submitting.current = true;
  try { await onSave({ ...draft, naziv: draft.naziv.trim() }, file); } finally { submitting.current = false; }
 }
 return <form className="contact-editor" aria-label={upload ? 'Dodavanje dokumenta' : 'Izmena dokumenta'} onSubmit={submit}>
   <fieldset disabled={busy || blocked}>
     <label>Tip dokumenta<select required value={draft.id_tip_doc} onChange={event => change('id_tip_doc', Number(event.target.value))}>
       <option value={0} disabled>Izaberite tip</option>
       {draft.id_tip_doc > 0 && !types.some(type => type.id === draft.id_tip_doc) && <option value={draft.id_tip_doc}>Trenutni tip (neaktivan)</option>}
       {types.map(type => <option key={type.id} value={type.id}>{type.naziv}</option>)}
     </select></label>
     {upload && <label>Fajl<input type="file" aria-label="Fajl" aria-describedby={fileHelp} required accept=".pdf,.docx,.jpg,.jpeg,.png" onChange={event => { const next = event.target.files[0] || null; setFile(next); if (next && !draft.naziv) change('naziv', next.name.slice(0, 200)); }}/><small id={fileHelp}>PDF, DOCX, JPG ili PNG, do {Math.round(maxBytes / 1024 / 1024)} MB.</small></label>}
     <label>Naziv dokumenta<input required maxLength={200} value={draft.naziv} onChange={event => change('naziv', event.target.value)}/></label>
     <label>Broj dokumenta<input maxLength={100} value={draft.broj_dokumenta} onChange={event => change('broj_dokumenta', event.target.value)}/></label>
     <label>Datum dokumenta<input type="date" value={draft.datum_dokumenta} onChange={event => change('datum_dokumenta', event.target.value)}/></label>
     <label>Napomena<textarea maxLength={2000} rows={3} value={draft.napomena} onChange={event => change('napomena', event.target.value)}/></label>
   </fieldset>
   {blocked && <p>Podaci nisu potvrđeni ili je pravo izmene isteklo. Zatvorite prozor i osvežite listu pre nastavka.</p>}
   <div className="contact-actions"><button className="button" type="button" disabled={busy} onClick={onCancel}>Odustani</button><button className="button primary" type="submit" disabled={busy || blocked || !draft.id_tip_doc}>{busy ? 'Čuvanje…' : 'Sačuvaj dokument'}</button></div>
 </form>;
}
