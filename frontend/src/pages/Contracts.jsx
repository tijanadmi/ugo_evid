import { useEffect, useRef, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { useAuth } from '../auth';
import Icon from '../components/Icon';

const number = new Intl.NumberFormat('sr-Latn-RS', { maximumFractionDigits: 2 });
const date = new Intl.DateTimeFormat('sr-Latn-RS');
const dateKeys = new Set(['pocetak_ug', 'kraj_ug', 'datpri', 'datizm']);
const numberKeys = new Set(['vrednost_ug', 'kurs_ug']);
function value(item, key) {
  const raw = item[key];
  if (raw === null || raw === undefined || raw === '') return '—';
  if (dateKeys.has(key)) return Number.isNaN(Date.parse(raw)) ? '—' : date.format(new Date(raw));
  if (numberKeys.has(key)) return number.format(raw);
  return String(raw);
}
function positive(value, fallback) {
  const parsed = Number(value);
  return Number.isSafeInteger(parsed) && parsed > 0 && parsed <= 2147483647 ? parsed : fallback;
}

const detailGroups = [
  ['Ugovor', [['br_ugovor', 'Broj ugovora'], ['godina', 'Godina'], ['ugovor_dms', 'DMS'], ['jn', 'Javna nabavka'], ['br_poz_plana', 'Pozicija plana'], ['predmet_ugovora', 'Predmet ugovora'], ['otvoren_ug', 'Oznaka otvorenog ugovora'], ['zzn', 'ZZN'], ['pocetak_ug', 'Početak'], ['kraj_ug', 'Završetak'], ['vrednost_ug', 'Vrednost'], ['valuta_ug', 'Valuta'], ['kurs_ug', 'Kurs']]],
  ['Dobavljač i organizacija', [['naziv', 'Dobavljač'], ['sluzba', 'Služba'], ['kom_grupa', 'Komercijalna grupa'], ['m_br_komerc', 'Matični broj komercijaliste'], ['naz_komerc', 'Komercijalista'], ['naziv_gr_plan', 'Grupa plana'], ['vrs_pred', 'Vrsta predmeta']]],
  ['Kontakt', [['ime', 'Ime'], ['telefon', 'Telefon'], ['email', 'Email']]],
  ['Odgovorna lica', Array.from({ length: 6 }, (_, i) => [[`odg_zap_${i + 1}`, `Šifra lica ${i + 1}`], [`naziv_odg_zap_${i + 1}`, `Odgovorno lice ${i + 1}`]]).flat()],
  ['Podaci evidencije', [['id_ugo_evid', 'ID evidencije'], ['id_ugo_org', 'ID organizacije'], ['id_sap_ugovor', 'ID ugovora'], ['id_sap_dobavljac', 'ID dobavljača'], ['status', 'Status evidencije'], ['datpri', 'Datum prijave'], ['datizm', 'Datum izmene']]],
];

function ContractDetails({ item, onClose }) {
  const ref = useRef(null);
  useEffect(() => { const dialog = ref.current; dialog.showModal(); return () => dialog.close(); }, []);
  return <dialog className="detail-dialog" ref={ref} onClose={onClose} aria-labelledby="detail-title" onClick={e => { if (e.target === ref.current) onClose(); }}>
    <div className="detail-header"><div><span className="eyebrow">DETALJI UGOVORA</span><h2 id="detail-title">Ugovor {value(item, 'br_ugovor')}</h2></div><button className="icon-button" onClick={onClose} aria-label="Zatvori detalje"><Icon name="close"/></button></div>
    <div className="detail-body">{detailGroups.map(([title, fields]) => <section key={title}><h3>{title}</h3><dl>{fields.map(([key, label]) => <div key={key}><dt>{label}</dt><dd>{value(item, key)}</dd></div>)}</dl></section>)}</div>
    <div className="detail-footer"><button className="button" onClick={onClose}>Zatvori</button></div>
  </dialog>;
}

export default function Contracts({ status }) {
  const { api } = useAuth();
  const [params, setParams] = useSearchParams();
  const page = positive(params.get('page_id'), 1);
  const pageSize = Math.min(positive(params.get('page_size'), 20), 100);
  const orgID = positive(params.get('id_ugo_org'), 0);
  const isOpen = status === 'otvoreni';
  const title = isOpen ? 'Otvoreni ugovori' : 'Zatvoreni ugovori';
  const [data, setData] = useState({ items: [], total: 0 });
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [version, setVersion] = useState(0);
  const [organizations, setOrganizations] = useState([]);
  const [orgLoading, setOrgLoading] = useState(true);
  const [orgError, setOrgError] = useState('');
  const [orgVersion, setOrgVersion] = useState(0);
  const [selected, setSelected] = useState(null);
  const [updated, setUpdated] = useState(null);

  useEffect(() => {
    const controller = new AbortController();
    setOrgLoading(true); setOrgError('');
    (async () => {
      const all = [];
      let next = 1;
      while (true) {
        const result = await api(`/ugo_org?page_id=${next}&page_size=100`, { signal: controller.signal });
        const items = result.items || [];
        all.push(...items);
        if (items.length === 0 || all.length >= result.total) break;
        next++;
      }
      if (!controller.signal.aborted) setOrganizations(all);
    })().catch(error => { if (!controller.signal.aborted) setOrgError(error.message); })
      .finally(() => { if (!controller.signal.aborted) setOrgLoading(false); });
    return () => controller.abort();
  }, [api, orgVersion]);

  useEffect(() => {
    const controller = new AbortController();
    setLoading(true); setError(''); setSelected(null);
    const query = new URLSearchParams({ page_id: page, page_size: pageSize, id_ugo_org: orgID });
    api(`/ugo_evid/${status}?${query}`, { signal: controller.signal })
      .then(result => {
        if (!controller.signal.aborted) {
          setData({ items: result.items || [], total: result.total || 0 });
          setUpdated(new Date());
        }
      }).catch(error => { if (!controller.signal.aborted) setError(error.message); })
      .finally(() => { if (!controller.signal.aborted) setLoading(false); });
    return () => controller.abort();
  }, [api, status, page, pageSize, orgID, version]);

  function change(values) { setParams({ page_id: '1', page_size: String(pageSize), id_ugo_org: String(orgID), ...values }); }
  const pages = Math.max(1, Math.ceil(data.total / pageSize));
  return <>
    <div className="breadcrumb">Evidencija ugovora <Icon name="chevron" width="14" height="14"/> <span>{title}</span></div>
    <div className="page-heading"><div><div className="heading-row"><h1>{title}</h1><span className={`status-badge ${isOpen ? 'open' : 'closed'}`}><span/>{isOpen ? 'Otvoreni' : 'Zatvoreni'}</span></div><p>Pregled ugovora, dobavljača i odgovornih lica.</p></div><button className="button" onClick={() => setVersion(v => v + 1)} disabled={loading}><Icon name="refresh"/>Osveži</button></div>
    <section className="contracts-panel" aria-label={title}>
      <div className="table-toolbar"><div className="organization-filter"><label htmlFor="organization">Organizaciona jedinica</label><select id="organization" value={orgID} disabled={orgLoading} onChange={e => change({ id_ugo_org: e.target.value })}>
        <option value="0">{orgLoading ? 'Učitavanje jedinica…' : 'Sve organizacione jedinice'}</option>
        {orgID > 0 && !organizations.some(org => org.id === orgID) && <option value={orgID}>Organizaciona jedinica {orgID}</option>}
        {organizations.map(org => <option key={org.id} value={org.id}>{org.sifra} — {org.naziv}</option>)}
      </select></div><div className="results-total" aria-live="polite">{loading ? 'Učitavanje…' : error ? 'Pregled nije dostupan' : <><strong>{number.format(data.total)}</strong> evidencija</>}</div></div>
      {orgError && <div className="alert warning" role="alert">Lista organizacionih jedinica nije učitana. {orgError} <button className="text-button" onClick={() => setOrgVersion(v => v + 1)}>Pokušaj ponovo</button></div>}
      {loading ? <div className="table-state" role="status"><span className="spinner"/>Učitavanje ugovora…</div> : error ? <div className="table-state"><Icon name="file" width="36" height="36"/><h3>Pregled trenutno nije dostupan</h3><p role="alert">{error}</p><button className="button" onClick={() => setVersion(v => v + 1)}>Pokušaj ponovo</button></div> : data.items.length === 0 ? <div className="table-state"><span className="empty-icon"><Icon name={isOpen ? 'open' : 'closed'} width="32" height="32"/></span><h3>Nema ugovora za prikaz</h3><p>{page > 1 ? 'Na ovoj stranici nema rezultata.' : 'Za izabranu organizacionu jedinicu nema odgovarajućih evidencija.'}</p>{page > 1 && <button className="button" onClick={() => change({ page_id: '1' })}>Prva stranica</button>}</div> : <div className="table-scroll" tabIndex="0" aria-label="Tabela ugovora, horizontalno pomeranje"><table>
        <caption className="sr-only">{title} — stranica {page}</caption><thead><tr><th scope="col">Broj ugovora</th><th scope="col">Predmet ugovora</th><th scope="col">Dobavljač</th><th scope="col">Služba</th><th scope="col">Početak</th><th scope="col">Završetak</th><th scope="col" className="numeric">Vrednost</th><th scope="col">Kontakt</th><th scope="col"><span className="sr-only">Detalji</span></th></tr></thead>
        <tbody>{data.items.map(item => <tr key={item.id_ugo_evid}>
          <th scope="row"><button className="contract-link" onClick={() => setSelected(item)}>{value(item, 'br_ugovor')}</button><small>{value(item, 'godina')}</small></th>
          <td className="subject-cell"><span>{value(item, 'predmet_ugovora')}</span><small>JN: {value(item, 'jn')}</small></td>
          <td className="supplier-cell">{value(item, 'naziv')}</td><td><span className="unit-badge">{value(item, 'sluzba')}</span></td>
          <td className="nowrap">{value(item, 'pocetak_ug')}</td><td className="nowrap">{value(item, 'kraj_ug')}</td><td className="numeric nowrap">{value(item, 'vrednost_ug')}<small>{value(item, 'valuta_ug')}</small></td>
          <td className="contact-cell">{value(item, 'ime')}<small>{value(item, 'email')}</small></td><td><button className="icon-button" onClick={() => setSelected(item)} aria-label={`Detalji ugovora ${item.br_ugovor}`}><Icon name="chevron"/></button></td>
        </tr>)}</tbody>
      </table></div>}
      <div className="table-footer"><label className="page-size">Po stranici <select aria-label="Broj redova po stranici" value={pageSize} onChange={e => change({ page_size: e.target.value })}>{[...new Set([10, 20, 50, 100, pageSize])].sort((a,b) => a-b).map(n => <option key={n} value={n}>{n}</option>)}</select></label><nav className="pagination" aria-label="Paginacija"><span>{!loading && !error ? `Stranica ${page} od ${pages}` : '…'}</span><button className="icon-button previous" disabled={page <= 1 || loading} aria-label="Prethodna stranica" onClick={() => change({ page_id: String(page - 1) })}><Icon name="chevron"/></button><button className="icon-button" disabled={page >= pages || loading || !!error} aria-label="Sledeća stranica" onClick={() => change({ page_id: String(page + 1) })}><Icon name="chevron"/></button></nav></div>
    </section>
    <div className="page-note"><span>Prikazane su evidencije ugovora iz poslovne baze.</span>{updated && !loading && !error && <span>Poslednje osvežavanje: {updated.toLocaleTimeString('sr-Latn-RS', { hour: '2-digit', minute: '2-digit' })}</span>}</div>
    {selected && <ContractDetails item={selected} onClose={() => setSelected(null)}/>}
  </>;
}
