import { useEffect, useRef } from 'react';
import PartnerCard from './PartnerCard';
import Icon from '../../ui/Icon';

export default function PartnerDetails({ partner, onClose }) {
  const ref = useRef(null);
  useEffect(() => {
    const dialog = ref.current;
    dialog.showModal();
    return () => dialog.close();
  }, []);
  return <dialog className="detail-dialog partner-dialog" ref={ref} aria-labelledby="partner-detail-title" onClose={() => { if (!ref.current?.open) onClose(); }} onClick={event => { if (event.target === ref.current) onClose(); }}>
    <div className="detail-header"><h2 id="partner-detail-title">Detalji partnera</h2><button className="icon-button" aria-label="Zatvori detalje partnera" onClick={onClose}><Icon name="close"/></button></div>
    <PartnerCard partner={partner}/>
    <div className="detail-footer"><button className="button" onClick={onClose}>Zatvori</button></div>
  </dialog>;
}
