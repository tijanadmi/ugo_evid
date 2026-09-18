import { useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import Icon from '../../ui/Icon';
import { number } from '../../utils/contractFormatting';
import { useContracts } from './useContracts';
import { useOrganizations } from '../organizations/useOrganizations';
import ContractDetails from './ContractDetails';
import ContractRow from './ContractRow';
function positive(value, fallback) {
  const parsed = Number(value);
  return Number.isSafeInteger(parsed) && parsed > 0 && parsed <= 2147483647 ? parsed : fallback;
}

export default function ContractTable({ status }) {
  const [params, setParams] = useSearchParams();
  const page = positive(params.get('page_id'), 1);
  const pageSize = Math.min(positive(params.get('page_size'), 20), 100);
  const orgID = positive(params.get('id_ugo_org'), 0);
  const isOpen = status === 'otvoreni';
  const title = isOpen ? 'Otvoreni ugovori' : 'Zatvoreni ugovori';
  const [selected, setSelected] = useState(null);
  const contractsQuery = useContracts({ status, page, pageSize, orgID });
  const organizationsQuery = useOrganizations();
  const data = contractsQuery.data || { items: [], total: 0 };
  const loading = contractsQuery.isFetching;
  const error = contractsQuery.error?.message || '';
  const organizations = organizationsQuery.data || [];
  const orgLoading = organizationsQuery.isPending;
  const orgError = organizationsQuery.error?.message || '';
  const updated = contractsQuery.dataUpdatedAt ? new Date(contractsQuery.dataUpdatedAt) : null;

  function change(values) { setSelected(null); setParams({ page_id: '1', page_size: String(pageSize), id_ugo_org: String(orgID), ...values }); }
  const pages = Math.max(1, Math.ceil(data.total / pageSize));
  return <>
    <div className="breadcrumb">Evidencija ugovora <Icon name="chevron" width="14" height="14"/> <span>{title}</span></div>
    <div className="page-heading"><div><div className="heading-row"><h1>{title}</h1><span className={`status-badge ${isOpen ? 'open' : 'closed'}`}><span/>{isOpen ? 'Otvoreni' : 'Zatvoreni'}</span></div><p>Pregled ugovora, dobavljača i odgovornih lica.</p></div><button className="button" onClick={() => contractsQuery.refetch()} disabled={loading}><Icon name="refresh"/>Osveži</button></div>
    <section className="contracts-panel" aria-label={title}>
      <div className="table-toolbar"><div className="organization-filter"><label htmlFor="organization">Organizaciona jedinica</label><select id="organization" value={orgID} disabled={orgLoading} onChange={e => change({ id_ugo_org: e.target.value })}>
        <option value="0">{orgLoading ? 'Učitavanje jedinica…' : 'Sve organizacione jedinice'}</option>
        {orgID > 0 && !organizations.some(org => org.id === orgID) && <option value={orgID}>Organizaciona jedinica {orgID}</option>}
        {organizations.map(org => <option key={org.id} value={org.id}>{org.sifra} — {org.naziv}</option>)}
      </select></div><div className="results-total" aria-live="polite">{loading ? 'Učitavanje…' : error ? 'Pregled nije dostupan' : <><strong>{number.format(data.total)}</strong> evidencija</>}</div></div>
      {orgError && <div className="alert warning" role="alert">Lista organizacionih jedinica nije učitana. {orgError} <button className="text-button" onClick={() => organizationsQuery.refetch()}>Pokušaj ponovo</button></div>}
      {loading ? <div className="table-state" role="status"><span className="spinner"/>Učitavanje ugovora…</div> : error ? <div className="table-state"><Icon name="file" width="36" height="36"/><h3>Pregled trenutno nije dostupan</h3><p role="alert">{error}</p><button className="button" onClick={() => contractsQuery.refetch()}>Pokušaj ponovo</button></div> : data.items.length === 0 ? <div className="table-state"><span className="empty-icon"><Icon name={isOpen ? 'open' : 'closed'} width="32" height="32"/></span><h3>Nema ugovora za prikaz</h3><p>{page > 1 ? 'Na ovoj stranici nema rezultata.' : 'Za izabranu organizacionu jedinicu nema odgovarajućih evidencija.'}</p>{page > 1 && <button className="button" onClick={() => change({ page_id: '1' })}>Prva stranica</button>}</div> : <div className="table-scroll" tabIndex="0" aria-label="Tabela ugovora, horizontalno pomeranje"><table>
        <caption className="sr-only">{title} — stranica {page}</caption><thead><tr><th scope="col">Broj ugovora</th><th scope="col">Predmet ugovora</th><th scope="col">Dobavljač</th><th scope="col">Služba</th><th scope="col">Početak</th><th scope="col">Završetak</th><th scope="col">Odgovorna lica</th><th scope="col">Kontakt</th><th scope="col"><span className="sr-only">Detalji</span></th></tr></thead>
        <tbody>{data.items.map(item => <ContractRow key={item.id_ugo_evid} item={item} onSelect={setSelected}/>)}</tbody>
      </table></div>}
      <div className="table-footer"><label className="page-size">Po stranici <select aria-label="Broj redova po stranici" value={pageSize} onChange={e => change({ page_size: e.target.value })}>{[...new Set([10, 20, 50, 100, pageSize])].sort((a,b) => a-b).map(n => <option key={n} value={n}>{n}</option>)}</select></label><nav className="pagination" aria-label="Paginacija"><span>{!loading && !error ? `Stranica ${page} od ${pages}` : '…'}</span><button className="icon-button previous" disabled={page <= 1 || loading} aria-label="Prethodna stranica" onClick={() => change({ page_id: String(page - 1) })}><Icon name="chevron"/></button><button className="icon-button" disabled={page >= pages || loading || !!error} aria-label="Sledeća stranica" onClick={() => change({ page_id: String(page + 1) })}><Icon name="chevron"/></button></nav></div>
    </section>
    <div className="page-note"><span>Prikazane su evidencije ugovora iz poslovne baze.</span>{updated && !loading && !error && <span>Poslednje osvežavanje: {updated.toLocaleTimeString('sr-Latn-RS', { hour: '2-digit', minute: '2-digit' })}</span>}</div>
    {selected && <ContractDetails item={selected} onClose={() => setSelected(null)}/>}
  </>;
}
