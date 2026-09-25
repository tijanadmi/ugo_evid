import { useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { usePartners } from './usePartners';
import PartnerDetails from './PartnerDetails';
import Icon from '../../ui/Icon';

const positive = (raw, fallback) => Number.isSafeInteger(Number(raw)) && Number(raw) > 0 && Number(raw) <= 2147483647 ? Number(raw) : fallback;

function NameFilter({ name, onSearch }) {
  const [draft, setDraft] = useState(name);
  return <form className="partner-filter" onSubmit={event => { event.preventDefault(); onSearch(draft.trim()); }}>
    <label htmlFor="partner-name">Naziv dobavljača</label>
    <div className="partner-filter-controls"><input id="partner-name" type="search" maxLength={200} value={draft} onChange={event => setDraft(event.target.value)} placeholder="Unesite deo naziva…"/>
      <button className="button" type="submit">Pretraži</button>
      {(name || draft) && <button className="text-button" type="button" onClick={() => { setDraft(''); onSearch(''); }}>Poništi</button>}
    </div>
  </form>;
}

export default function PartnersList() {
  const [params, setParams] = useSearchParams();
  const page = positive(params.get('page_id'), 1);
  const size = Math.min(positive(params.get('page_size'), 20), 100);
  const name = (params.get('naziv') || '').trim();
  const { data, error, isFetching, refetch } = usePartners(page, size, name);
  const [selected, setSelected] = useState(null);
  const pages = Math.max(1, Math.ceil((data?.total || 0) / size));
  const queryKey = params.toString();
  function change(pageID, pageSize = size, naziv = name) {
    setSelected(null);
    setParams({ page_id: String(pageID), page_size: String(pageSize), ...(naziv ? { naziv } : {}) });
  }
  const address = partner => [partner.adresa, partner.grad].filter(value => value?.trim()).join(', ') || '—';
  return <>
    <div className="page-heading"><div><h1>Moji partneri</h1><p>Dobavljači iz ugovora vaše organizacione jedinice.</p></div><button className="button" disabled={isFetching} onClick={() => { setSelected(null); refetch(); }}><Icon name="refresh"/>Osveži</button></div>
    <section className="contracts-panel" aria-label="Pregled partnera">
      <div className="table-toolbar"><NameFilter key={name} name={name} onSearch={naziv => change(1, size, naziv)}/><div className="results-total" aria-live="polite">{isFetching ? 'Učitavanje…' : error ? 'Pregled nije dostupan' : <><strong>{data?.total || 0}</strong> partnera</>}</div></div>
      {isFetching ? <div className="table-state" role="status">Učitavanje partnera…</div>
        : error ? <div className="table-state" role="alert"><h2>Pregled partnera nije dostupan</h2><p>{error.message}</p><button className="button" onClick={() => refetch()}>Pokušaj ponovo</button></div>
        : data?.items?.length ? <div className="table-scroll" tabIndex="0" aria-label="Tabela partnera"><table className="partners-table">
          <thead><tr><th scope="col">Naziv partnera</th><th scope="col">Adresa i grad</th><th scope="col"><span className="sr-only">Detalji</span></th></tr></thead>
          <tbody>{data.items.map(partner => <tr key={partner.id}>
            <th scope="row"><button className="contract-link" onClick={() => setSelected({ partner, queryKey })}>{partner.naziv || '—'}</button></th>
            <td>{address(partner)}</td>
            <td><button className="icon-button" aria-label={`Detalji partnera ${partner.naziv}`} onClick={() => setSelected({ partner, queryKey })}><Icon name="chevron"/></button></td>
          </tr>)}</tbody>
        </table></div>
        : <div className="table-state"><h2>Nema partnera za prikaz</h2><p>{name ? 'Nema dobavljača koji odgovaraju unetom nazivu.' : 'Za vašu organizacionu jedinicu nema partnera na ovoj stranici.'}</p>{page > 1 && <button className="button" onClick={() => change(1)}>Prva stranica</button>}</div>}
      <div className="table-footer"><label className="page-size">Po stranici <select aria-label="Broj partnera po stranici" value={size} onChange={event => change(1, Number(event.target.value))}>{[...new Set([10,20,50,100,size])].sort((a,b)=>a-b).map(n => <option key={n}>{n}</option>)}</select></label><nav className="pagination" aria-label="Paginacija partnera"><span>Stranica {page} od {pages}</span><button className="icon-button" aria-label="Prethodna stranica" disabled={page <= 1 || isFetching} onClick={() => change(page-1)}>‹</button><button className="icon-button" aria-label="Sledeća stranica" disabled={page >= pages || isFetching || !!error} onClick={() => change(page+1)}>›</button></nav></div>
    </section>
    {selected && selected.queryKey === queryKey && !isFetching && !error && <PartnerDetails partner={selected.partner} onClose={() => setSelected(null)}/>}
  </>;
}
