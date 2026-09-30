import { test, expect } from '@playwright/test';

async function setup(context, options = {}) {
  const state = { bodies: [], filters: [], registered: false, ...options };
  const contract = { id: 101, id_sap_dobavljac: 20, br_ugovor: '460001', godina: '2026', predmet_ugovora: 'Održavanje opreme', dobavljac: 'Primer dobavljača', otvoren_ug: 'X' };
  const person = { id: 7, ime: 'Ana Kontakt', telefon: '011 123', email: 'ana@example.test', version: '2' };
  await context.addInitScript(() => sessionStorage.setItem('ugo-evid-session', JSON.stringify({ access_token: 'access', refresh_token: 'refresh', access_token_expires_at: new Date(Date.now() + 3600000).toISOString(), refresh_token_expires_at: new Date(Date.now() + 7200000).toISOString(), user: { username: 'test.user' } })));
  await context.route('**/api/**', async route => {
    const request = route.request();
    const url = new URL(request.url());
    const json = (body, status = 200) => route.fulfill({ status, contentType: 'application/json', body: JSON.stringify(body) });
    if (url.pathname === '/api/ugo_org') return json({ total: 1, items: [{ id: 3, sifra: 'CTKS', naziv: 'Moja organizacija' }] });
    if (url.pathname === '/api/ugo_evid/otvoreni' || url.pathname === '/api/ugo_evid/zatvoreni') return json({ total: 0, items: [] });
    if (url.pathname === '/api/sapugovori/neevidentirani') {
      state.filters.push(url.searchParams.get('filter'));
      expect(url.searchParams.get('page_size')).toBe('20');
      return json({ total: state.registered || url.searchParams.get('filter') === 'Nema' ? 0 : 21, items: state.registered || url.searchParams.get('filter') === 'Nema' ? [] : [contract] });
    }
    if (url.pathname === '/api/sapugovori/101/priprema') {
      if (state.registered) return json({ error: 'Ugovor je već evidentiran.' }, 409);
      return json({ contract, contact: state.newContact ? null : person, id_ugo_org: 3 });
    }
    if (url.pathname === '/api/ugo_evid' && request.method() === 'POST') {
      const body = request.postDataJSON();
      state.bodies.push(body);
      expect(body.id_ugo_org).toBeUndefined();
      expect(body.id_sap_dobavljac).toBeUndefined();
      if (state.registered) return json({ error: 'Ugovor je već evidentiran.' }, 409);
      if (state.locked) return json({ error: 'Lice trenutno uređuje drugi korisnik.' }, 423);
      if (state.changed) return json({ error: 'Kontakt lice je promenjeno. Ponovo učitajte podatke.' }, 409);
      state.registered = true;
      return json({ data: { id: 55, ugo_org: { id: 3 }, sap_ugovor: { id: 101 } } }, 201);
    }
    return json({ error: 'Unknown route' }, 404);
  });
  return state;
}
async function openPicker(page) {
  await page.goto('/ugovori/otvoreni');
  await page.getByRole('button', { name: 'Izaberi ugovor', exact: true }).click();
  await expect(page.getByRole('dialog', { name: 'Izaberi ugovor' })).toBeVisible();
}
async function choose(page) {
  await page.getByRole('button', { name: 'Izaberi ugovor 460001', exact: true }).click();
  await expect(page.getByRole('button', { name: 'Evidentiraj ugovor', exact: true })).toBeEnabled();
}

test('filter, pagination and register using existing active role-1 contact', async ({ context, page }) => {
  const state = await setup(context);
  await openPicker(page);
  await page.getByLabel('Broj ugovora, predmet ili dobavljač').fill('Nema');
  await page.getByRole('button', { name: 'Pretraži', exact: true }).click();
  await expect(page.getByText('Nema neevidentiranih ugovora za izabrani filter.')).toBeVisible();
  await page.getByLabel('Broj ugovora, predmet ili dobavljač').fill('Oprema');
  await page.getByRole('button', { name: 'Pretraži', exact: true }).click();
  await expect(page.getByRole('button', { name: 'Sledeća', exact: true })).toBeEnabled();
  await page.getByRole('button', { name: 'Sledeća', exact: true }).click();
  await expect(page.getByRole('navigation', { name: 'Stranice izbora ugovora' })).toContainText('Stranica 2 od 2');
  await choose(page);
  await expect(page.getByRole('region', { name: 'Postojeće kontakt lice' })).toContainText('Ana Kontakt');
  await expect(page.getByRole('form', { name: 'Unos ugovora' })).toContainText('Moja organizacija');
  await page.getByRole('button', { name: 'Evidentiraj ugovor' }).click();
  await expect(page.getByRole('link', { name: 'Otvori detalje ugovora' })).toHaveAttribute('href', '/ugovori/detalji/55');
  expect(state.bodies).toEqual([{ id_sap_ugovor: 101, id_ugo_dob_lica: 7, contact_version: '2' }]);
  expect(state.filters).toContain('Oprema');
});

test('new role-1 contact is sent together with contract; work position optional', async ({ context, page }) => {
  const state = await setup(context, { newContact: true });
  await openPicker(page);
  await choose(page);
  await page.getByLabel('Ime i prezime', { exact: true }).fill('Novo lice');
  await page.getByLabel('Telefon', { exact: true }).fill('011 555');
  await page.getByLabel('Email', { exact: true }).fill('novo@example.test');
  await page.getByRole('button', { name: 'Evidentiraj ugovor' }).click();
  await expect(page.getByRole('link', { name: 'Otvori detalje ugovora' })).toBeVisible();
  expect(state.bodies).toEqual([{ id_sap_ugovor: 101, novo_lice: { ime: 'Novo lice', radno_mesto: '', telefon: '011 555', email: 'novo@example.test' } }]);
});

test('two tabs selecting one SAP contract handle a concurrent registration', async ({ context, page }) => {
  const state = await setup(context);
  await openPicker(page); await choose(page);
  const second = await context.newPage();
  await openPicker(second); await choose(second);
  await page.getByRole('button', { name: 'Evidentiraj ugovor' }).click();
  await expect(page.getByRole('link', { name: 'Otvori detalje ugovora' })).toBeVisible();
  await second.getByRole('button', { name: 'Evidentiraj ugovor' }).click();
  await expect(second.getByRole('alert')).toContainText('već evidentiran');
  await expect(second.getByRole('button', { name: 'Evidentiraj ugovor' })).toBeDisabled();
  await second.getByRole('button', { name: 'Ponovo proveri ugovor' }).click();
  await expect(second.getByRole('alert')).toContainText('već evidentiran');
  expect(state.bodies).toHaveLength(2);
});

test('contact conflict preserves input and requires rechecking before retry', async ({ context, page }) => {
  const state = await setup(context, { newContact: true, changed: true });
  await openPicker(page); await choose(page);
  await page.getByLabel('Ime i prezime', { exact: true }).fill('Moj unos');
  await page.getByLabel('Telefon', { exact: true }).fill('011');
  await page.getByLabel('Email', { exact: true }).fill('moj@example.test');
  await page.getByRole('button', { name: 'Evidentiraj ugovor' }).click();
  await expect(page.getByRole('alert')).toContainText('Kontakt lice je promenjeno');
  await expect(page.getByLabel('Ime i prezime', { exact: true })).toHaveValue('Moj unos');
  await expect(page.getByRole('button', { name: 'Evidentiraj ugovor' })).toBeDisabled();
  state.changed = false; state.newContact = false;
  await page.getByRole('button', { name: 'Ponovo proveri ugovor' }).click();
  await expect(page.getByRole('region', { name: 'Postojeće kontakt lice' })).toContainText('Ana Kontakt');
  await expect(page.getByRole('button', { name: 'Evidentiraj ugovor' })).toBeEnabled();
});

test('active contact edit blocks registration and closed list has no picker', async ({ context, page }) => {
  await setup(context, { locked: true });
  await openPicker(page); await choose(page);
  await page.getByRole('button', { name: 'Evidentiraj ugovor' }).click();
  await expect(page.getByRole('alert')).toContainText('drugi korisnik');
  await page.goto('/ugovori/zatvoreni');
  await expect(page.getByRole('heading', { name: 'Zatvoreni ugovori', exact: true })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Izaberi ugovor', exact: true })).toHaveCount(0);
});
