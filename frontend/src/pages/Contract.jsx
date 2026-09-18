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
        <div className="compact-contract-heading">
          <div className="contract-heading-meta"><span>SAP UGOVOR</span><span className={`status-badge ${item.otvoren_ug === 'X' ? 'open' : 'closed'}`}>{item.otvoren_ug === 'X' ? 'Otvoren ugovor' : 'Zatvoren ugovor'}</span></div>
          <h1><span>{value(item, 'br_ugovor')}</span><span className="contract-title-divider"> / </span>{value(item, 'predmet_ugovora')}</h1>
        </div>
        <ContractDetails item={item}/>
      </>}
  </>;
}
