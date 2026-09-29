import { useEffect, useRef, useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import PartnerCard from './PartnerCard';
import ContactEditor from './ContactEditor';
import Icon from '../../ui/Icon';

export default function PartnerDetails({ partner, onClose, onRefresh }) {
  const ref = useRef(null);
  const [editor, setEditor] = useState(null);
  const [busy, setBusy] = useState(false);
  const [notice, setNotice] = useState('');
  const queryClient = useQueryClient();
  useEffect(() => {
    const dialog = ref.current;
    dialog.showModal();
    return () => dialog.close();
  }, []);
  function close() {
    if (busy) return;
    if (editor && !window.confirm('Odustati od uređivanja lica?')) return;
    onClose();
  }
  async function saved() {
    setEditor(null);
    setNotice('Promena je sačuvana.');
    queryClient.invalidateQueries({ queryKey: ['contracts'] });
    queryClient.invalidateQueries({ queryKey: ['contract'] });
    queryClient.invalidateQueries({ queryKey: ['partners'], refetchType: 'none' });
    const result = await onRefresh();
    if (result.isError) setNotice('Promena je sačuvana, ali osvežavanje spiska nije uspelo. Zatvorite detalje i osvežite pregled.');
  }
  return <dialog className="detail-dialog partner-dialog" ref={ref} aria-labelledby="partner-detail-title" onCancel={event => { event.preventDefault(); close(); }} onClick={event => { if (event.target === ref.current) close(); }}>
    <div className="detail-header"><h2 id="partner-detail-title">Detalji partnera</h2><button className="icon-button" disabled={busy} aria-label="Zatvori detalje partnera" onClick={close}><Icon name="close"/></button></div>
    {notice && <p className="contact-notice" role="status">{notice}</p>}
    {editor ? <ContactEditor key={`${editor.mode}-${editor.id || 'new'}`} supplierID={partner.id} personID={editor.id} mode={editor.mode} onBusy={setBusy} onCancel={() => setEditor(null)} onSaved={saved}/> : <>
      <div className="partner-add"><button className="button primary" onClick={() => { setNotice(''); setEditor({ mode: 'add' }); }}>+ Dodaj lice</button></div>
      <PartnerCard partner={partner} onEdit={person => { setNotice(''); setEditor({ mode: 'edit', id: person.id }); }} onDelete={person => { setNotice(''); setEditor({ mode: 'delete', id: person.id }); }}/>
    </>}
    <div className="detail-footer"><button className="button" disabled={busy} onClick={close}>Zatvori</button></div>
  </dialog>;
}
