import { value } from '../../utils/contractFormatting';
const detailGroups = [
  ['Ugovor', [['br_ugovor', 'Broj ugovora'], ['godina', 'Godina'], ['ugovor_dms', 'DMS'], ['jn', 'Javna nabavka'], ['br_poz_plana', 'Pozicija plana'], ['predmet_ugovora', 'Predmet ugovora'], ['otvoren_ug', 'Oznaka otvorenog ugovora'], ['zzn', 'ZZN'], ['pocetak_ug', 'Početak'], ['kraj_ug', 'Završetak'], ['vrednost_ug', 'Vrednost'], ['valuta_ug', 'Valuta'], ['kurs_ug', 'Kurs']]],
  ['Dobavljač i organizacija', [['naziv', 'Dobavljač'], ['sluzba', 'Služba'], ['kom_grupa', 'Komercijalna grupa'], ['m_br_komerc', 'Matični broj komercijaliste'], ['naz_komerc', 'Komercijalista'], ['naziv_gr_plan', 'Grupa plana'], ['vrs_pred', 'Vrsta predmeta']]],
  ['Kontakt', [['ime', 'Ime'], ['telefon', 'Telefon'], ['email', 'Email']]],
  ['Odgovorna lica', Array.from({ length: 6 }, (_, i) => [[`odg_zap_${i + 1}`, `Šifra lica ${i + 1}`], [`naziv_odg_zap_${i + 1}`, `Odgovorno lice ${i + 1}`]]).flat()],
  ['Podaci evidencije', [['id_ugo_evid', 'ID evidencije'], ['id_ugo_org', 'ID organizacije'], ['id_sap_ugovor', 'ID ugovora'], ['id_sap_dobavljac', 'ID dobavljača'], ['status', 'Status evidencije'], ['datpri', 'Datum prijave'], ['datizm', 'Datum izmene']]],
];


export default function ContractDetails({ item }) {
  return <div className="detail-body">{detailGroups.map(([title, fields]) => <section key={title}><h3>{title}</h3><dl>{fields.map(([key, label]) => <div key={key}><dt>{label}</dt><dd>{value(item, key)}</dd></div>)}</dl></section>)}</div>;
}
