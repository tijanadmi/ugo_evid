import { useEffect, useRef, useState } from 'react';
import { formatDate } from '../utils/contractFormatting';

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
 useEffect(()=>{setText(value ? formatDate(value) : '');ref.current?.setCustomValidity('');},[value]);
 function change(event) {
  const next=event.target.value;
  setText(next);
  const iso=isoDate(next);
  const valid=next==='' || (iso && (!min || iso>=min) && (!max || iso<=max));
  event.target.setCustomValidity(valid?'':'Unesite važeći datum u formatu dd.mm.yyyy.');
  if(valid) onChange({target:{value:iso||''}});
 }
 return <input {...props} ref={ref} type="text" inputMode="numeric" placeholder="dd.mm.yyyy" value={text} onChange={change} onBlur={()=>{const iso=isoDate(text);if(iso)setText(formatDate(iso));}}/>;
}
