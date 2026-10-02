import { useEffect, useRef, useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import toast from 'react-hot-toast';
import ContactEditor from './ContactEditor';
import ConfirmAction from '../../ui/ConfirmAction';
import Icon from '../../ui/Icon';
import { announceBusinessChange } from '../../services/DataSync';

export default function AddContactDialog({ supplierID, onClose }) {
  const ref = useRef(null);
  const client = useQueryClient();
  const [busy, setBusy] = useState(false);
  const [confirmClose, setConfirmClose] = useState(false);
  useEffect(() => {
    const dialog = ref.current;
    dialog.showModal();
    return () => dialog.close();
  }, []);
  function close() {
    if (!busy) setConfirmClose(true);
  }
  function saved() {
    announceBusinessChange(client);
    toast.success('Lice je dodato.');
    onClose();
  }
  return <dialog ref={ref} className="detail-dialog partner-dialog" aria-labelledby="add-contact-title"
    onCancel={event => { event.preventDefault(); close(); }}
    onClick={event => { if (event.target === ref.current) close(); }}>
    <div className="detail-header"><h2 id="add-contact-title">Unos novog lica</h2>
      <button type="button" className="icon-button" disabled={busy} aria-label="Zatvori unos lica" onClick={close}><Icon name="close"/></button>
    </div>
    <ContactEditor supplierID={supplierID} onBusy={setBusy} onCancel={close} onSaved={saved}/>
    {confirmClose && <ConfirmAction modal title="Odustati od unosa lica?" disabled={busy} onCancel={() => setConfirmClose(false)} onConfirm={onClose}>
      <p>Nesačuvani podaci biće izgubljeni.</p>
    </ConfirmAction>}
  </dialog>;
}
