import { test, expect } from '@playwright/test';
import { supplierContacts } from '../src/features/contracts/supplierContacts';

const person = { id: 7, ime: 'Ana Anić', email: 'ana@example.test', telefon: '011 123-456', rola_lica: 'Service Level Manager' };
const saved = { ime: ' ANA  ANIĆ ', email: 'ANA@example.test ', telefon: '(011) 123 456' };
const list = (item = {}, people = [person]) => supplierContacts({ ...saved, ...item, lica_dobavljaca: people });

test('legacy normalized contact is first and hides only the matching SLM', () => {
  const other = { ...person, id: 8, rola_lica: 'Tehnička podrška' };
  const result = list({}, [other, person]);
  expect(result).toHaveLength(2);
  expect(result[0].saved).toBe(true);
  expect(result[0].ime).toBe(saved.ime);
  expect(result[0].rola_lica).toBe(person.rola_lica);
  expect(result[1].id).toBe(8);
});

test('IDs take precedence and changed snapshot data remains visible', () => {
  expect(list({ id_ugo_dob_lica: 7 })).toHaveLength(1);
  expect(list({ id_ugo_dob_lica: 8 })).toHaveLength(2);
  expect(list({ id_ugo_dob_lica: 7, telefon: '999' })).toHaveLength(2);
  expect(list({ id_ugo_dob_lica: 7, email: '' })).toHaveLength(2);
});

test('uncertain names-only or ambiguous matches are retained', () => {
  expect(list({ email: '', telefon: '' }, [{ ...person, email: '', telefon: '' }])).toHaveLength(2);
  expect(list({}, [person, { ...person, id: 8 }])).toHaveLength(3);
  expect(list({ ime: 'Drugo lice' })).toHaveLength(2);
  expect(list({}, [{ ...person, rola_lica: 'Druga rola' }])).toHaveLength(2);
});

test('empty snapshots do not hide current contacts; saved-only contacts remain', () => {
  expect(list({ ime: '', email: null, telefon: ' ', id_ugo_dob_lica: 7 })).toEqual([{ ...person, saved: false }]);
  expect(list({}, [])).toHaveLength(1);
  expect(list({ ime: '', email: '', telefon: '' }, [])).toHaveLength(0);
});
