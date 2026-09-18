export default function ResponsiblePersons({ item }) {
  const people = Array.from({ length: 6 }, (_, index) => {
    const slot = index + 1;
    return {
      slot,
      code: String(item[`odg_zap_${slot}`] ?? '').trim(),
      name: String(item[`naziv_odg_zap_${slot}`] ?? '').trim(),
    };
  }).filter(person => person.code || person.name);

  if (!people.length) return <span>—</span>;
  return <ul className="responsible-persons" aria-label="Odgovorna lica">
    {people.map(person => <li key={person.slot}>
      {person.code && <span className="person-code">{person.code}</span>}
      {person.code && person.name && ' — '}
      {person.name}
    </li>)}
  </ul>;
}
