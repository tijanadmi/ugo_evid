import { useSearchParams } from 'react-router-dom';
import { usePartners } from './usePartners';
import PartnerCard from './PartnerCard';
import Icon from '../../ui/Icon';

const positive = (raw, fallback) => Number.isSafeInteger(Number(raw)) && Number(raw) > 0 && Number(raw) <= 2147483647 ? Number(raw) : fallback;

export default function PartnersList() {
  const [params, setParams] = useSearchParams();
  const page = positive(params.get('page_id'), 1);
  const size = Math.min(positive(params.get('page_size'), 20), 100);
  const { data, error, isFetching, refetch } = usePartners(page, size);
  const pages = Math.max(1, Math.ceil((data?.total || 0) / size));
  function change(pageID, pageSize = size) { setParams({ page_id: String(pageID), page_size: String(pageSize) }); }
  return <>
    <div className="page-heading"><div><h1>Moji partneri</h1><p>Dobavljači iz otvorenih i zatvorenih ugovora vaše organizacione jedinice.</p></div><button className="button" disabled={isFetching} onClick={() => refetch()}><Icon name="refresh"/>Osveži</button></div>
    {isFetching ? <div className="table-state" role="status">Učitavanje partnera…</div>
      : error ? <div className="table-state" role="alert"><h2>Pregled partnera nije dostupan</h2><p>{error.message}</p><button className="button" onClick={() => refetch()}>Pokušaj ponovo</button></div>
      : <>
        <p className="muted partners-total">Ukupno partnera: {data?.total || 0}</p>
        {data?.items?.length ? <div className="partners-list">{data.items.map(partner => <PartnerCard partner={partner} key={partner.id}/>)}</div>
          : <div className="table-state"><h2>Nema partnera za prikaz</h2><p>{page > 1 ? 'Na ovoj stranici nema rezultata.' : 'Za vašu organizacionu jedinicu nema dobavljača povezanih sa evidencijama ugovora.'}</p>{page > 1 && <button className="button" onClick={() => change(1)}>Prva stranica</button>}</div>}
      </>}
    <div className="table-footer"><label className="page-size">Po stranici <select aria-label="Broj partnera po stranici" value={size} onChange={e => change(1, Number(e.target.value))}>{[...new Set([10,20,50,100,size])].sort((a,b)=>a-b).map(n => <option key={n}>{n}</option>)}</select></label><nav className="pagination" aria-label="Paginacija partnera"><span>Stranica {page} od {pages}</span><button className="icon-button" aria-label="Prethodna stranica" disabled={page <= 1 || isFetching} onClick={() => change(page-1)}>‹</button><button className="icon-button" aria-label="Sledeća stranica" disabled={page >= pages || isFetching || !!error} onClick={() => change(page+1)}>›</button></nav></div>
  </>;
}
