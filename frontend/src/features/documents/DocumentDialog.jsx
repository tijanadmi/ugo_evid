import { useEffect, useId, useRef } from 'react';
import Icon from '../../ui/Icon';

export default function DocumentDialog({ title, onClose, busy, children, wide = false }) {
 const ref = useRef(null);
 const id = useId();
 useEffect(() => { const dialog = ref.current; dialog.showModal(); return () => dialog.close(); }, []);
 return <dialog ref={ref} className={`detail-dialog document-dialog${wide ? ' document-preview' : ''}`} aria-labelledby={id}
   onCancel={event => { event.preventDefault(); if (!busy) onClose(); }}
   onClick={event => { if (event.target === ref.current && !busy) onClose(); }}>
   <div className="detail-header"><h2 id={id}>{title}</h2><button type="button" className="icon-button" disabled={busy} onClick={onClose} aria-label="Zatvori dokument"><Icon name="close"/></button></div>
   {children}
 </dialog>;
}
