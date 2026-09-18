export const number = new Intl.NumberFormat('sr-Latn-RS', { maximumFractionDigits: 2 });
const date = new Intl.DateTimeFormat('sr-Latn-RS');
const dateKeys = new Set(['pocetak_ug', 'kraj_ug', 'datpri', 'datizm']);
const numberKeys = new Set(['vrednost_ug', 'kurs_ug']);
const rate = new Intl.NumberFormat('sr-Latn-RS', { maximumFractionDigits: 5 });
export function value(item, key) {
  const raw = item[key];
  if (raw === null || raw === undefined || raw === '') return '—';
  if (dateKeys.has(key)) return Number.isNaN(Date.parse(raw)) ? '—' : date.format(new Date(raw));
  if (numberKeys.has(key)) return (key === 'kurs_ug' ? rate : number).format(raw);
  return String(raw);
}
