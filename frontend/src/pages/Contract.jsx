import { Link, useLocation, useParams } from 'react-router-dom';
import { useContract } from '../features/contracts/useContract';
import ContractDetails from '../features/contracts/ContractDetails';
import { value } from '../utils/contractFormatting';

export default function Contract() {
  const { id } = useParams();
  const { state } = useLocation();
  const contractID = Number(id);
  const valid = Number.isSafeInteger(contractID) && contractID > 0;
  const { data: item, isPending, error, refetch } = useContract(contractID);
  const fallback = item && item.otvoren_ug !== 'X' ? '/ugovori/zatvoreni' : '/ugovori/otvoreni';
  const back = /^\/ugovori\/(otvoreni|zatvoreni)(\?|$)/.test(state?.returnTo || '') ? state.returnTo : fallback;

  return <>
    <Link className="button" to={back}>← Povratak na pregled</Link>
    {!valid ? <div className="table-state" role="alert">Neispravan ID evidencije.</div>
      : error ? <div className="table-state" role="alert"><h1>{error.status === 404 ? 'Ugovor nije pronađen' : 'Detalji nisu dostupni'}</h1><p>{error.message}</p>{error.status !== 404 && <button className="button" onClick={() => refetch()}>Pokušaj ponovo</button>}</div>
      : isPending ? <div className="table-state" role="status">Učitavanje detalja ugovora…</div>
      : <>
        <div className="page-heading contract-detail-heading"><div><h1>Ugovor {value(item, 'br_ugovor')}</h1><p>{item.otvoren_ug === 'X' ? 'Otvoren ugovor' : 'Zatvoren ugovor'}</p></div></div>
        <article className="contracts-panel"><ContractDetails item={item}/></article>
        <section className="contracts-panel supplier-contacts" aria-labelledby="supplier-contacts-title">
          <h2 id="supplier-contacts-title">Lica dobavljača</h2>
          {!item.lica_dobavljaca?.length ? <p className="table-state">Nema evidentiranih lica za ovog dobavljača.</p> : <div className="table-scroll" tabIndex="0" aria-label="Lica dobavljača"><table>
            <thead><tr>{['Ime', 'Radno mesto', 'Telefon', 'Email', 'Rola lica'].map(label => <th scope="col" key={label}>{label}</th>)}</tr></thead>
            <tbody>{item.lica_dobavljaca.map((person, index) => <tr key={index}>{['ime', 'radno_mesto', 'telefon', 'email', 'rola_lica'].map(key => <td key={key}>{value(person, key)}</td>)}</tr>)}</tbody>
          </table></div>}
        </section>
      </>}
  </>;
}
