import { useEffect, useRef, useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import toast from 'react-hot-toast';
import { useAuth } from '../../context/AuthContext';
import { deleteContact } from '../../services/apiContacts';
import { announceBusinessChange, invalidateBusinessData } from '../../services/DataSync';
import { useContactSession } from './useContactSession';
import ContactEditor from './ContactEditor';
import ConfirmAction from '../../ui/ConfirmAction';
import ConfirmDelete from '../../ui/ConfirmDelete';
import Icon from '../../ui/Icon';

function ContactDialog({ mode, onClose, busy, children }) {
  const ref = useRef(null);
  useEffect(() => {
    const element = ref.current;
    element.showModal();
    return () => element.close();
  }, []);
  return <dialog ref={ref} className="detail-dialog partner-dialog" aria-label={mode === 'edit' ? 'Izmena lica' : 'Brisanje lica'}
    onCancel={event => { event.preventDefault(); onClose(); }}
    onClick={event => { if (event.target === ref.current) onClose(); }}>
    <div className="detail-header"><h2>{mode === 'edit' ? 'Izmena lica' : 'Brisanje lica'}</h2>
      <button type="button" className="icon-button" disabled={busy} aria-label="Zatvori lice" onClick={onClose}><Icon name="close"/></button>
    </div>
    {children}
  </dialog>;
}

// Keep the lease owner outside individual rows so refreshed contacts cannot
// interrupt an active edit or release its lock.
export function useContactActions(supplierID) {
  const { api } = useAuth();
  const client = useQueryClient();
  const owner = useContactSession(api);
  const pending = useRef(false);
  const [editor, setEditor] = useState(null);
  const [busy, setBusy] = useState(false);
  const [blocked, setBlocked] = useState(false);
  const [confirmClose, setConfirmClose] = useState(false);
  async function open(person, mode) {
    if (pending.current || busy || editor) return;
    pending.current = true; setBusy(true);
    try {
      if (await owner.acquire(person.id, mode)) { setBlocked(false); setEditor({ id: person.id, mode }); }
      else await invalidateBusinessData(client);
    } finally { pending.current = false; setBusy(false); }
  }
  async function cancel() {
    setBusy(true);
    await owner.release();
    setEditor(null); setConfirmClose(false); setBusy(false);
  }
  function close() {
    if (busy || pending.current) return;
    if (editor.mode === 'edit') setConfirmClose(true);
    else cancel();
  }
  function saved(message = 'Promena je sačuvana.') {
    owner.committed(); setEditor(null);
    toast.success(message); announceBusinessChange(client);
  }
  async function remove() {
    if (pending.current || busy || blocked || owner.lost || !owner.session) return;
    pending.current = true; setBusy(true);
    try {
      if (!await owner.renew()) return;
      await deleteContact(api, editor.id, owner.session.person.version, owner.session.token);
      saved('Lice je obrisano.');
    } catch (error) { toast.error(error.message, { id: 'operation-error' }); setBlocked(true); }
    finally { pending.current = false; setBusy(false); }
  }
  const dialog = editor && <ContactDialog mode={editor.mode} busy={busy} onClose={close}>
    {editor.mode === 'delete' ? <ConfirmDelete resourceName={`lice ${owner.session?.person.ime || ''}`} onConfirm={remove}
      disabled={busy} blocked={blocked || owner.lost || !owner.session} onCloseModal={cancel}/> :
      <ContactEditor supplierID={supplierID} personID={editor.id} session={owner.session} leaseLost={owner.lost}
        beforeWrite={owner.renew} onReload={async () => { await owner.release(); await owner.acquire(editor.id, 'edit'); }}
        onBusy={setBusy} onCancel={close} onSaved={() => saved()}/>}
    {confirmClose && <ConfirmAction modal title="Odustati od uređivanja lica?" disabled={busy} onCancel={() => setConfirmClose(false)} onConfirm={cancel}>
      <p>Nesačuvane izmene biće izgubljene.</p>
    </ConfirmAction>}
  </ContactDialog>;
  return { open, busy, dialog };
}
