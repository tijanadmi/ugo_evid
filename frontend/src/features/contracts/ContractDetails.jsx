import { value } from '../../utils/contractFormatting';
import ResponsiblePersons from './ResponsiblePersons';
import { supplierContacts, isServiceLevelManager } from './supplierContacts';

const fields = [
  ['godina', 'Godina'], ['jn', 'Broj nabavke'],
  ['br_poz_plana', 'Pozicija plana'], ['ugovor_dms', 'DMS broj'],
  ['pocetak_ug', 'Važi od'], ['kraj_ug', 'Važi do'],
];

export default function ContractDetails({ item }) {
	const contacts = supplierContacts(item);
  return <div className="contract-summary-grid">
    <section className="contract-card contract-facts" aria-labelledby="contract-facts-title">
      <h2 id="contract-facts-title">O ugovoru</h2>
      <dl>{fields.map(([key, label]) => <div key={key}><dt>{label}</dt><dd>{value(item, key)}</dd></div>)}
        <div className="commercial-contact"><dt>Zaduženi komercijalista</dt><dd>{value(item, 'naz_komerc')}{item.m_br_komerc && <span className="commercial-code"> · {item.m_br_komerc}</span>}</dd></div>
      </dl>
    </section>
    <section className="contract-card operational-contacts" aria-labelledby="operational-title">
      <h2 id="operational-title">Lica za operativno praćenje ugovora</h2>
      <ResponsiblePersons item={item}/>
    </section>
    <section className="contract-card partner-card" aria-labelledby="partner-title">
      <div className="partner-heading"><div><h2 id="partner-title">Podaci o partneru / dobavljaču</h2><p className="partner-name">{value(item, 'naziv')}</p></div><span className="partner-count">Kontakti: {contacts.length}</span></div>
      {!contacts.length ? <p className="compact-empty">Nema evidentiranih lica za ovog dobavljača.</p>
        : <ul className="partner-people" aria-label="Lica dobavljača">{contacts.map((person, index) => {
          const isSLM = isServiceLevelManager(person);
          return <li key={index} className={isSLM ? 'partner-person partner-person-slm' : 'partner-person'}>
            <div className="partner-person-identity"><strong>{value(person, 'ime')}</strong><span>{value(person, 'radno_mesto')}</span></div>
            <div className="partner-role">{person.saved ? <strong>Kontakt za ovaj ugovor</strong> : <span className="contact-source">Trenutni podaci</span>}{person.rola_lica && <span className="contact-role-name">{person.rola_lica}</span>}</div>
            <div className="partner-person-contact"><span>{value(person, 'email')}</span><span>{value(person, 'telefon')}</span></div>
          </li>;
        })}</ul>}
    </section>
  </div>;
}
