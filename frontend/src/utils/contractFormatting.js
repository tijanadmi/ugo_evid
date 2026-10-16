export const number = new Intl.NumberFormat('sr-Latn-RS', { maximumFractionDigits: 2 });
export function formatDate(raw) {
  if (!raw) return '—';
  const match = String(raw).match(/^(\d{4})-(\d{2})-(\d{2})(?:T|$| )/);
  return match ? `${match[3]}.${match[2]}.${match[1]}` : '—';
}
const dateKeys = new Set(['pocetak_ug', 'kraj_ug', 'datpri', 'datizm']);
const numberKeys = new Set(['vrednost_ug', 'kurs_ug']);
const rate = new Intl.NumberFormat('sr-Latn-RS', { maximumFractionDigits: 5 });
export function value(item, key) {
  const raw = item[key];
  if (raw === null || raw === undefined || raw === '') return '—';
  if (dateKeys.has(key)) return formatDate(raw);
  if (numberKeys.has(key)) return (key === 'kurs_ug' ? rate : number).format(raw);
  return String(raw);
}
