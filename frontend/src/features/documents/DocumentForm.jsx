import { useCallback, useEffect, useId, useRef, useState } from 'react';
import toast from 'react-hot-toast';

export default function DocumentForm({ fixedType, types, initial, upload, maxBytes, onSave, onCancel, busy, blocked }) {
 const fileHelp = useId();
 const [draft, setDraft] = useState(() => ({ id_tip_doc: initial.id_tip_doc || types[0]?.id || 0, naziv: initial.naziv || '', broj_dokumenta: initial.broj_dokumenta || '', datum_dokumenta: initial.datum_dokumenta || '', napomena: initial.napomena || '' }));
 const [file, setFile] = useState(null);
 const submitting = useRef(false);
 const formRef = useRef(null);
 const inputRef = useRef(null);
 const [dragging, setDragging] = useState(false);
 const selectFile = useCallback(next => {
  setFile(next);
  if (next) setDraft(current => ({ ...current, naziv: current.naziv || next.name.slice(0, 200) }));
 }, []);
 useEffect(() => {
  if (!upload) return;
  const dialog = formRef.current.closest('dialog');
  if (!dialog) return;
  let depth = 0;
  const isFile = event => Array.from(event.dataTransfer?.types || []).includes('Files');
  const unavailable = () => busy || blocked || submitting.current || dialog.querySelector('dialog[open]');
  function enter(event) {
   if (!isFile(event)) return;
   event.preventDefault();
   depth++;
   if (!unavailable()) setDragging(true);
  }
  function over(event) {
   if (!isFile(event)) return;
   event.preventDefault();
   event.dataTransfer.dropEffect = unavailable() ? 'none' : 'copy';
  }
  function leave(event) {
   if (!isFile(event)) return;
   depth = Math.max(0, depth - 1);
   if (!depth) setDragging(false);
  }
  function drop(event) {
   if (!isFile(event)) return;
   event.preventDefault();
   depth = 0; setDragging(false);
   if (unavailable()) return;
   const files = event.dataTransfer.files;
   if (files.length !== 1) { toast.error('Prevucite jedan dokument.'); return; }
   inputRef.current.files = files;
   selectFile(files[0]);
  }
  dialog.addEventListener('dragenter', enter);
  dialog.addEventListener('dragover', over);
  dialog.addEventListener('dragleave', leave);
  dialog.addEventListener('drop', drop);
  return () => {
   dialog.removeEventListener('dragenter', enter);
   dialog.removeEventListener('dragover', over);
   dialog.removeEventListener('dragleave', leave);
   dialog.removeEventListener('drop', drop);
  };
 }, [upload, busy, blocked, selectFile]);
 function change(key, value) { setDraft(current => ({ ...current, [key]: value })); }
 async function submit(event) {
  event.preventDefault();
  if (submitting.current || busy || blocked) return;
  if (upload && (!file || file.size === 0 || file.size > maxBytes)) { toast.error(`Izaberite fajl do ${Math.round(maxBytes / 1024 / 1024)} MB.`); return; }
  if (upload && !/\.(pdf|docx|jpe?g|png)$/i.test(file.name)) { toast.error('Izaberite PDF, DOCX, JPG ili PNG dokument.'); return; }
  if (!draft.naziv.trim()) { toast.error('Unesite naziv dokumenta.'); return; }
  submitting.current = true;
  try { await onSave({ ...draft, naziv: draft.naziv.trim() }, file); } finally { submitting.current = false; }
 }
 return <form ref={formRef} className="contact-editor" aria-label={upload ? 'Dodavanje dokumenta' : 'Izmena dokumenta'} onSubmit={submit}>
   {upload && <p className={`document-drop-hint${dragging && !busy && !blocked ? ' is-dragging' : ''}`} role="status">{dragging && !busy && !blocked ? 'Pustite dokument ovde.' : 'Prevucite dokument na ovaj prozor ili izaberite fajl dugmetom ispod.'}{file && <span>Izabran fajl: {file.name}</span>}</p>}
   <fieldset disabled={busy || blocked}>
     <label>Tip dokumenta{fixedType ? <input readOnly value={fixedType.naziv}/> : <select required value={draft.id_tip_doc} onChange={event => change('id_tip_doc', Number(event.target.value))}>
       <option value={0} disabled>Izaberite tip</option>
       {draft.id_tip_doc > 0 && !types.some(type => type.id === draft.id_tip_doc) && <option value={draft.id_tip_doc}>Trenutni tip (neaktivan)</option>}
       {types.map(type => <option key={type.id} value={type.id}>{type.naziv}</option>)}
     </select>}</label>
     {upload && <label>Fajl<input ref={inputRef} type="file" aria-label="Fajl" aria-describedby={fileHelp} required accept=".pdf,.docx,.jpg,.jpeg,.png" onChange={event => selectFile(event.target.files[0] || null)}/><small id={fileHelp}>PDF, DOCX, JPG ili PNG, do {Math.round(maxBytes / 1024 / 1024)} MB.</small></label>}
     <label>Naziv dokumenta<input required maxLength={200} value={draft.naziv} onChange={event => change('naziv', event.target.value)}/></label>
     <label>Broj dokumenta<input maxLength={100} value={draft.broj_dokumenta} onChange={event => change('broj_dokumenta', event.target.value)}/></label>
     {/* Datum dokumenta je privremeno sakriven; polje i postojece vrednosti ostaju sacuvani. */}
     {/* <label>Datum dokumenta<input type="date" value={draft.datum_dokumenta} onChange={event => change('datum_dokumenta', event.target.value)}/></label> */}
     <label>Napomena<textarea maxLength={2000} rows={3} value={draft.napomena} onChange={event => change('napomena', event.target.value)}/></label>
   </fieldset>
   {blocked && <p>Podaci nisu potvrđeni ili je pravo izmene isteklo. Zatvorite prozor i osvežite listu pre nastavka.</p>}
   <div className="contact-actions"><button className="button" type="button" disabled={busy} onClick={onCancel}>Odustani</button><button className="button primary" type="submit" disabled={busy || blocked || !draft.id_tip_doc}>{busy ? 'Čuvanje…' : 'Sačuvaj dokument'}</button></div>
 </form>;
}
