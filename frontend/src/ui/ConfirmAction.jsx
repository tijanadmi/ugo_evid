import { useEffect, useId, useRef } from 'react';
import { createPortal } from 'react-dom';

export default function ConfirmAction({ title, children, onConfirm, onCancel, confirmLabel = 'OK', cancelLabel = 'Otkaži', variation = 'primary', disabled = false, blocked = false, modal = false, label }) {
  const ref = useRef(null);
  const titleID = useId();
  useEffect(() => {
    if (!modal) return;
    const dialog = ref.current;
    dialog.showModal();
    return () => dialog.close();
  }, [modal]);
  const content = <section className="confirm-delete" aria-label={label}>
    <h3 id={titleID}>{title}</h3>
    {children}
    <div className="contact-actions">
      <button type="button" className="button" autoFocus={modal} disabled={disabled} onClick={onCancel}>{cancelLabel}</button>
      <button type="button" className={`button ${variation}`} disabled={disabled || blocked} onClick={onConfirm}>{confirmLabel}</button>
    </div>
  </section>;
  if (!modal) return content;
  return createPortal(<dialog ref={ref} className="detail-dialog confirmation-dialog" aria-labelledby={titleID}
    onCancel={event => { event.preventDefault(); event.stopPropagation(); if (!disabled) onCancel(); }}
    onClick={event => { event.stopPropagation(); if (event.target === ref.current && !disabled) onCancel(); }}>
    {content}
  </dialog>, document.body);
}
