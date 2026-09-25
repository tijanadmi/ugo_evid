import SupplierAddress from '../contracts/SupplierAddress';
import { isServiceLevelManager } from '../contracts/supplierContacts';
import { value } from '../../utils/contractFormatting';

export default function PartnerCard({ partner }) {
  const people = partner.lica_dobavljaca || [];
  return <article className="contract-card partner-card">
    <div className="partner-heading"><div><h2>{value(partner, 'naziv')}</h2><SupplierAddress item={partner}/>{partner.sifra && <small className="muted">Šifra: {partner.sifra}</small>}</div><span className="partner-count">Lica: {people.length}</span></div>
    {!people.length ? <p className="compact-empty">Nema evidentiranih kontakt osoba.</p> : <ul className="partner-people" aria-label={`Kontakt osobe: ${partner.naziv}`}>
      {people.map(person => <li key={person.id} className={`partner-person${isServiceLevelManager(person) ? ' partner-person-slm' : ''}`}>
        <div className="partner-person-identity"><strong>{value(person, 'ime')}</strong><span>{value(person, 'radno_mesto')}</span></div>
        <span className="partner-role">{value(person, 'rola_lica')}</span>
        <div className="partner-person-contact"><span>{value(person, 'email')}</span><span>{value(person, 'telefon')}</span></div>
      </li>)}
    </ul>}
  </article>;
}
