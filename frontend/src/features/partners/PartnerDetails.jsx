import { useEffect, useRef, useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import toast from 'react-hot-toast';
import { useAuth } from '../../context/AuthContext';
import { deleteContact } from '../../services/apiContacts';
import { announceBusinessChange } from '../../services/DataSync';
import { useContactSession } from './useContactSession';
import PartnerCard from './PartnerCard';
import ContactEditor from './ContactEditor';
import ConfirmDelete from '../../ui/ConfirmDelete';
import Icon from '../../ui/Icon';

export default function PartnerDetails({ partner, onClose, onRefresh }) {
  const { api } = useAuth();
  const ref = useRef(null);
  const actionPending = useRef(false);
  const [editor, setEditor] = useState(null);
  const [busy, setBusy] = useState(false);
  const [deleteBlocked, setDeleteBlocked] = useState(false);
  const client = useQueryClient();
  const owner = useContactSession(api);
  useEffect(() => {
    const dialog = ref.current;
    dialog.showModal();
    return () => dialog.close();
  }, []);
  function close() {
    if (busy || actionPending.current) return;
    if (editor?.mode === 'edit' && !window.confirm('Odustati od uređivanja lica?')) return;
    onClose();
  }
  async function open(person, mode) {
    if (actionPending.current || busy) return;
    actionPending.current = true; setBusy(true);
    try {
      const acquired = await owner.acquire(person.id, mode);
      if (acquired) { setDeleteBlocked(false); setEditor({ mode, id: person.id }); }
      else await onRefresh();
    } finally { actionPending.current = false; setBusy(false); }
  }
  async function cancel() {
    setBusy(true);
    await owner.release();
    setEditor(null); setBusy(false);
  }
  function saved(message = 'Promena je sačuvana.') {
    owner.committed(); setEditor(null);
    toast.success(message);
    announceBusinessChange(client);
  }
  async function remove() {
    if (actionPending.current || busy || owner.lost || !owner.session || deleteBlocked) return;
    actionPending.current = true; setBusy(true);
    try {
      if (!await owner.renew()) return;
      await deleteContact(api, editor.id, owner.session.person.version, owner.session.token);
      saved('Lice je obrisano.');
    } catch (error) { toast.error(error.message, { id: 'operation-error' }); setDeleteBlocked(true); }
    finally { actionPending.current = false; setBusy(false); }
  }
  return <dialog className="detail-dialog partner-dialog" ref={ref} aria-labelledby="partner-detail-title" onCancel={event => { event.preventDefault(); close(); }} onClick={event => { if (event.target === ref.current) close(); }}>
    <div className="detail-header"><h2 id="partner-detail-title">Detalji partnera</h2><button className="icon-button" disabled={busy} aria-label="Zatvori detalje partnera" onClick={close}><Icon name="close"/></button></div>
    {editor?.mode === 'delete' ? <ConfirmDelete resourceName={`lice ${owner.session?.person.ime || ''}`} onConfirm={remove} disabled={busy} blocked={deleteBlocked || owner.lost || !owner.session} onCloseModal={cancel}/> : editor ?
      <ContactEditor supplierID={partner.id} personID={editor.id} session={owner.session} leaseLost={owner.lost} beforeWrite={owner.renew} onReload={async () => { await owner.release(); await owner.acquire(editor.id, 'edit'); }} onBusy={setBusy} onCancel={cancel} onSaved={() => saved()}/> : <>
        <div className="partner-add"><button className="button primary" disabled={busy} onClick={() => setEditor({ mode: 'add' })}>+ Dodaj lice</button></div>
        <PartnerCard partner={partner} onEdit={person => open(person, 'edit')} onDelete={person => open(person, 'delete')}/>
      </>}
    <div className="detail-footer"><button className="button" disabled={busy} onClick={close}>Zatvori</button></div>
  </dialog>;
}
