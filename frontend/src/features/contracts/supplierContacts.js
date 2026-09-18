const text = value => String(value ?? '').normalize('NFKC').trim().replace(/\s+/g, ' ').toLocaleLowerCase('sr-Latn');
// Ignore punctuation/spacing, but do not guess country codes or extensions.
const phone = value => text(value).replace(/[\s().\-/]/g, '');
const id = value => /^\d+$/.test(String(value ?? '')) && Number(value) > 0 ? String(value).replace(/^0+/, '') : null;
export const isServiceLevelManager = person => text(person.rola_lica).replace(/\s/g, '') === 'servicelevelmanager';

export function isSameSavedContact(saved, person) {
  const savedID = id(saved.id_ugo_dob_lica);
  const personID = id(person.id);
  if (savedID && personID && savedID !== personID) return false;
  const pairs = [[text(saved.ime), text(person.ime)], [text(saved.email), text(person.email)], [phone(saved.telefon), phone(person.telefon)]];
  // Preserve changed or newly populated fields, even for the same ID.
  if (pairs.some(([left, right]) => left !== right)) return false;
  if (savedID && personID) return pairs.some(([left]) => !!left);
  // Legacy records require a name and at least one matching contact channel.
  return !!pairs[0][0] && (!!pairs[1][0] || !!pairs[2][0]);
}

export function supplierContacts(item) {
  const people = item.lica_dobavljaca || [];
  const hasSaved = [item.ime, item.email, item.telefon].some(value => text(value));
  if (!hasSaved) return people.map(person => ({ ...person, saved: false }));
  const matches = people.filter(person => isServiceLevelManager(person) && isSameSavedContact(item, person));
  // Ambiguous matches remain visible; no arbitrary first-person selection.
  const duplicate = matches.length === 1 ? matches[0] : null;
  return [
    { ime: item.ime, email: item.email, telefon: item.telefon, saved: true,
      radno_mesto: duplicate?.radno_mesto, rola_lica: duplicate?.rola_lica },
    ...people.filter(person => person !== duplicate).map(person => ({ ...person, saved: false })),
  ];
}
