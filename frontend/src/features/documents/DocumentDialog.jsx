import { useEffect, useId, useRef, useState } from 'react';
import Icon from '../../ui/Icon';

export default function DocumentDialog({ title, onClose, busy, children, wide = false }) {
 const ref = useRef(null);
 const id = useId();
 const [expanded, setExpanded] = useState(false);
 function toggleSize() {
  ref.current.style.removeProperty('width');
  ref.current.style.removeProperty('height');
  setExpanded(current => !current);
 }
 useEffect(() => { const dialog = ref.current; dialog.showModal(); return () => dialog.close(); }, []);
 return <dialog ref={ref} className={`detail-dialog document-dialog${wide ? ' document-preview' : ''}${wide && expanded ? ' is-expanded' : ''}`} aria-labelledby={id}
   onCancel={event => { event.preventDefault(); if (!busy) onClose(); }}
   onClick={event => {
    if (event.target !== ref.current || busy) return;
    const bounds = ref.current.getBoundingClientRect();
    if (event.clientX < bounds.left || event.clientX > bounds.right || event.clientY < bounds.top || event.clientY > bounds.bottom) onClose();
   }}>
   <div className="detail-header"><h2 id={id}>{title}</h2><div className="document-window-actions">{wide && <button type="button" className="button" onClick={toggleSize} aria-pressed={expanded}>{expanded ? 'Vrati veličinu' : 'Uvećaj prozor'}</button>}<button type="button" className="icon-button" disabled={busy} onClick={onClose} aria-label="Zatvori dokument"><Icon name="close"/></button></div></div>
   {children}
 </dialog>;
}
