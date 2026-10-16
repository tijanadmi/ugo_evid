import { useEffect, useRef, useState } from 'react';
import { formatDate } from '../utils/contractFormatting';
import './dateInput.css';

function isoDate(text) {
 const match = text.match(/^(\d{2})\.(\d{2})\.(\d{4})$/);
 const iso = match ? `${match[3]}-${match[2]}-${match[1]}` : text;
 if (!/^\d{4}-\d{2}-\d{2}$/.test(iso)) return null;
 const date = new Date(`${iso}T00:00:00Z`);
 return !Number.isNaN(date.getTime()) && date.toISOString().slice(0,10) === iso ? iso : null;
}

export default function DateInput({ value='', onChange, min, max, ...props }) {
 const [text,setText] = useState(value ? formatDate(value) : '');
 const ref = useRef(null);
 const calendar = useRef(null);
 const [fallback,setFallback] = useState(false);
 useEffect(()=>{setText(value ? formatDate(value) : '');ref.current?.setCustomValidity('');},[value]);
 function change(event) {
  const next=event.target.value;
  setText(next);
  const iso=isoDate(next);
  const valid=next==='' || (iso && (!min || iso>=min) && (!max || iso<=max));
  event.target.setCustomValidity(valid?'':'Unesite važeći datum u formatu dd.mm.yyyy.');
  if(valid) onChange({target:{value:iso||''}});
 }
 function pick(event) {
  const next=event.target.value;
  if(!event.target.validity.valid) return;
  setText(next ? formatDate(next) : '');
  ref.current.setCustomValidity('');
  onChange({target:{value:next}});
 }
 function openCalendar() {
  try {
   if(calendar.current.showPicker) { calendar.current.showPicker(); return; }
  } catch { /* A browser without picker support can use the visible native input. */ }
  setFallback(true);
 }
 return <span className="date-input-control">
  <input {...props} ref={ref} type="text" inputMode="numeric" placeholder="dd.mm.yyyy" value={text} onChange={change} onBlur={()=>{const iso=isoDate(text);if(iso)setText(formatDate(iso));}}/>
  <button className="date-input-calendar-button" type="button" aria-label="Otvori kalendar" title="Izaberi datum iz kalendara" disabled={props.disabled || props.readOnly} onClick={openCalendar}>
   <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" aria-hidden="true"><rect x="3" y="5" width="18" height="16" rx="2"/><path d="M7 3v4M17 3v4M3 11h18M7 15h2M11 15h2M15 15h2"/></svg>
  </button>
  <input ref={calendar} className={fallback?'date-input-native-fallback':'date-input-native'} type="date" aria-label="Datum iz kalendara" tabIndex={fallback?0:-1} value={value} min={min} max={max} disabled={props.disabled || props.readOnly} onChange={pick}/>
 </span>;
}
