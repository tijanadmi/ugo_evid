import RowMenu from '../../ui/RowMenu';
import SupplierAddress from '../contracts/SupplierAddress';
import { isServiceLevelManager } from '../contracts/supplierContacts';
import { value } from '../../utils/contractFormatting';

export default function PartnerCard({ partner, onEdit, onDelete }) {
  const people = partner.lica_dobavljaca || [];
  return <article className="contract-card partner-card">
    <div className="partner-heading"><div><h2>{value(partner, 'naziv')}</h2><SupplierAddress item={partner}/>{partner.sifra && <small className="muted">Šifra: {partner.sifra}</small>}</div><span className="partner-count">Lica: {people.length}</span></div>
    {!people.length ? <p className="compact-empty">Nema evidentiranih kontakt osoba.</p> : <ul className="partner-people" aria-label={`Kontakt osobe: ${partner.naziv}`}>
      {people.map(person => <li key={person.id} className={`partner-person${isServiceLevelManager(person) ? ' partner-person-slm' : ''}`}>
        <div className="partner-person-identity"><strong>{value(person, 'ime')}</strong>{person.status === 'N' && <span>Neaktivan</span>}<span>{value(person, 'radno_mesto')}</span></div>
        <span className="partner-role">{value(person, 'rola_lica')}</span>
        <div className="partner-person-contact"><span>{value(person, 'email')}</span><span>{value(person, 'telefon')}</span></div>
        {(onEdit || onDelete) && <RowMenu label={`Akcije za lice ${person.ime}`} actions={[
          ...(onEdit ? [{ label: 'Izmeni', icon: 'edit', onClick: () => onEdit(person) }] : []),
          ...(onDelete ? [{ label: 'Obriši', icon: 'trash', danger: true, onClick: () => onDelete(person) }] : []),
        ]}/>}
      </li>)}
    </ul>}
  </article>;
}
