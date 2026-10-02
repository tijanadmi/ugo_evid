import { test, expect } from '@playwright/test';

async function setup(context, sharedState) {
  const state = sharedState || { people: [{ id: 7, ime: 'Ana', radno_mesto: 'Menadžer', telefon: '011', email: 'ana@example.test', status: 'A', version: '1', sap_dobavljac: { id: 20 }, ugo_dob_lica_rola: { id: 1 } }], token: null, renewFail: false, conflict: false, linked: false, writes: [], releases: 0, acquisitions: 0 };
  await context.addInitScript(() => sessionStorage.setItem('ugo-evid-session', JSON.stringify({ access_token: 'access', refresh_token: 'refresh', access_token_expires_at: new Date(Date.now() + 3600000).toISOString(), refresh_token_expires_at: new Date(Date.now() + 7200000).toISOString(), user: { username: 'test.user' } })));
  await context.route('**/api/**', async route => {
    const req = route.request();
    const path = new URL(req.url()).pathname;
    const method = req.method();
    const body = req.postData() ? req.postDataJSON() : {};
    const json = (body, status = 200) => route.fulfill({ status, contentType: 'application/json', body: JSON.stringify(body) });
    if (path === '/api/moji_partneri') return json({ total: 1, items: [{ id: 20, naziv: 'Moj partner', lica_dobavljaca: state.people.map(person => ({ ...person, rola_lica: 'Kontakt' })) }] });
    if (path === '/api/ugo_dob_lica_rola') {
      const size = Number(new URL(req.url()).searchParams.get('page_size'));
      if (!Number.isInteger(size) || size < 5 || size > 100) return json({ error: 'Invalid role page size' }, 400);
      return json({ total: 1, items: [{ id: 1, naziv: 'Kontakt', status: 'A' }] });
    }
    if (path.endsWith('/lock')) {
      if (method === 'POST') {
        if (state.token) return json({ error: 'Lice trenutno uređuje drugi korisnik ili drugi tab.' }, 423);
        state.acquisitions++;
        state.token = String(state.acquisitions).padStart(64, '0');
        return json({ lock_token: state.token, expires_at: new Date(Date.now() + 120000).toISOString() });
      }
      if (method === 'DELETE') {
        if (body.lock_token === state.token) { state.token = null; state.releases++; }
        return json({});
      }
      if (state.renewFail || body.lock_token !== state.token) return json({ error: 'Pravo izmene je isteklo.' }, 409);
      return json({ lock_token: state.token, expires_at: new Date(Date.now() + 120000).toISOString() });
    }
    if (path === '/api/ugo_dob_lica' && method === 'POST') {
      state.writes.push(body);
      const person = { ...body, id: 8, version: '1', sap_dobavljac: { id: 20 }, ugo_dob_lica_rola: { id: body.id_ugo_dob_lica_rola } };
      state.people.push(person);
      return json({ data: person });
    }
    if (/\/ugo_dob_lica\/\d+$/.test(path)) {
      const id = Number(path.split('/').at(-1));
      const person = state.people.find(person => person.id === id);
      if (!person) return json({ error: 'Lice nije pronađeno.' }, 404);
      if (method === 'GET') return json({ data: person });
      if (body.lock_token !== state.token || state.conflict || body.version !== person.version) return json({ error: 'Podaci su promenjeni. Učitajte aktuelne podatke.' }, 409);
      if (method === 'DELETE' && state.linked) return json({ error: 'Lice je povezano sa evidencijom ugovora. Umesto brisanja izaberite status Neaktivan.' }, 409);
      state.writes.push(body);
      state.token = null;
      if (method === 'DELETE') { state.people = state.people.filter(person => person.id !== id); return json({ status: 'deleted' }); }
      Object.assign(person, body, { version: String(Number(person.version) + 1), ugo_dob_lica_rola: { id: body.id_ugo_dob_lica_rola } });
      return json({ data: person });
    }
    return json({ error: 'Unknown test route' }, 404);
  });
  return state;
}
async function openPartner(page) {
  await page.goto('/ugovori/partneri');
  await page.getByRole('button', { name: 'Moj partner', exact: true }).click();
}

test('add, edit, deactivate and delete a contact without closing partner details', async ({ context, page }) => {
  const state = await setup(context);
  await openPartner(page);
  await page.getByRole('button', { name: '+ Dodaj lice', exact: true }).click();
  await expect(page.getByLabel('Ime i prezime')).toBeEnabled();
  await page.getByLabel('Ime i prezime').fill('Novi kontakt');
  await page.getByLabel('Email', { exact: true }).fill('novi@example.test');
  await page.getByRole('button', { name: 'Sačuvaj', exact: true }).click();
  await expect(page.locator('.partner-person').filter({ hasText: 'Novi kontakt' })).toBeVisible();
  expect(state.writes[0].id_sap_dobavljac).toBe(20);
  const ana = page.locator('.partner-person').filter({ hasText: 'Ana' });
  await ana.getByRole('button', { name: 'Akcije za lice Ana', exact: true }).click();
  await page.getByRole('menuitem', { name: 'Izmeni', exact: true }).click();
  await expect(page.getByLabel('Ime i prezime')).toHaveValue('Ana');
  await page.getByLabel('Ime i prezime').fill('Ana izmenjena');
  await page.getByRole('combobox', { name: 'Status', exact: true }).selectOption('N');
  await page.getByRole('button', { name: 'Sačuvaj', exact: true }).click();
  await expect(ana).toContainText('Ana izmenjena');
  await expect(ana).toContainText('Neaktivan');
  expect(state.writes[1].version).toBe('1');
  expect(state.writes[1].lock_token).toHaveLength(64);
  await page.getByRole('button', { name: 'Akcije za lice Novi kontakt', exact: true }).click();
  await page.getByRole('menuitem', { name: 'Obriši', exact: true }).click();
  await expect(page.getByRole('region', { name: 'Brisanje lica' })).toContainText('Novi kontakt');
  await page.getByRole('button', { name: 'Potvrdi brisanje' }).click();
  await expect(page.locator('.partner-person')).toHaveCount(1);
  await expect(page.getByRole('dialog')).toBeVisible();
});

test('two tabs cannot edit the same contact; cancellation releases it', async ({ context, page }) => {
  const state = await setup(context);
  await openPartner(page);
  await page.getByRole('button', { name: 'Akcije za lice Ana', exact: true }).click();
  await page.getByRole('menuitem', { name: 'Izmeni', exact: true }).click();
  await expect(page.getByRole('button', { name: 'Sačuvaj', exact: true })).toBeEnabled();
  const second = await context.newPage();
  await openPartner(second);
  await second.getByRole('button', { name: 'Akcije za lice Ana', exact: true }).click();
  await second.getByRole('menuitem', { name: 'Izmeni', exact: true }).click();
  await expect(second.getByRole('alert')).toContainText('drugi korisnik');
  await expect(second.getByRole('form', { name: 'Izmena lica' })).toHaveCount(0);
  await page.getByRole('button', { name: 'Odustani', exact: true }).click();
  await page.getByRole('button', { name: 'OK', exact: true }).click();
  await expect.poll(() => state.token).toBe(null);
  await second.getByRole('button', { name: 'Akcije za lice Ana', exact: true }).click();
  await second.getByRole('menuitem', { name: 'Izmeni', exact: true }).click();
  await expect(second.getByRole('button', { name: 'Sačuvaj', exact: true })).toBeEnabled();
  expect(state.acquisitions).toBe(2);
  await second.getByRole('button', { name: 'Odustani', exact: true }).click();
  await second.getByRole('button', { name: 'OK', exact: true }).click();
  await expect.poll(() => state.token).toBe(null);
});

test('lost renewal preserves draft and disables saving', async ({ context, page }) => {
  const state = await setup(context);
  await page.clock.install();
  await openPartner(page);
  await page.getByRole('button', { name: 'Akcije za lice Ana', exact: true }).click();
  await page.getByRole('menuitem', { name: 'Izmeni', exact: true }).click();
  await expect(page.getByLabel('Ime i prezime')).toHaveValue('Ana');
  await page.getByLabel('Ime i prezime').fill('Nezatvoren unos');
  state.renewFail = true;
  await page.clock.fastForward(31000);
  await expect(page.getByRole('alert')).toContainText('isteklo');
  await expect(page.getByLabel('Ime i prezime')).toHaveValue('Nezatvoren unos');
  await expect(page.getByRole('button', { name: 'Sačuvaj', exact: true })).toBeDisabled();
  expect(state.writes).toHaveLength(0);
});

test('version conflict preserves draft; explicit reload obtains new version', async ({ context, page }) => {
  const state = await setup(context);
  await openPartner(page);
  await page.getByRole('button', { name: 'Akcije za lice Ana', exact: true }).click();
  await page.getByRole('menuitem', { name: 'Izmeni', exact: true }).click();
  await expect(page.getByLabel('Ime i prezime')).toHaveValue('Ana');
  await page.getByLabel('Ime i prezime').fill('Moj unos');
  state.people[0].version = '2';
  state.people[0].ime = 'Tuđa izmena';
  await page.getByRole('button', { name: 'Sačuvaj', exact: true }).click();
  await expect(page.getByRole('alert')).toContainText('Podaci su promenjeni');
  await expect(page.getByLabel('Ime i prezime')).toHaveValue('Moj unos');
  await expect(page.getByRole('button', { name: 'Sačuvaj', exact: true })).toBeDisabled();
  await page.getByRole('button', { name: 'Učitaj aktuelne podatke' }).click();
  await page.getByRole('button', { name: 'OK', exact: true }).click();
  await expect(page.getByLabel('Ime i prezime')).toHaveValue('Tuđa izmena');
  await expect(page.getByRole('button', { name: 'Sačuvaj', exact: true })).toBeEnabled();
});

test('linked contact cannot be deleted and offers deactivation guidance', async ({ context, page }) => {
  const state = await setup(context);
  state.linked = true;
  await openPartner(page);
  await page.getByRole('button', { name: 'Akcije za lice Ana', exact: true }).click();
  await page.getByRole('menuitem', { name: 'Obriši', exact: true }).click();
  await expect(page.getByRole('button', { name: 'Potvrdi brisanje' })).toBeEnabled();
  await page.getByRole('button', { name: 'Potvrdi brisanje' }).click();
  await expect(page.getByRole('alert')).toContainText('Neaktivan');
  expect(state.people).toHaveLength(1);
  expect(state.writes).toHaveLength(0);
});

test('editing in one tab blocks delete and shows only toast in the second tab', async ({ context, page }) => {
  const state = await setup(context);
  await openPartner(page);
  await page.getByRole('button', { name: 'Akcije za lice Ana', exact: true }).click();
  await page.getByRole('menuitem', { name: 'Izmeni', exact: true }).click();
  await expect(page.getByLabel('Ime i prezime')).toHaveValue('Ana');
  const token = state.token;
  const second = await context.newPage();
  await openPartner(second);
  await second.getByRole('button', { name: 'Akcije za lice Ana', exact: true }).click();
  await second.getByRole('menuitem', { name: 'Obriši', exact: true }).click();
  await expect(second.getByRole('alert')).toContainText('drugi korisnik');
  await expect(second.getByRole('alert')).toBeVisible();
  await expect(second.getByRole('region', { name: 'Brisanje lica' })).toHaveCount(0);
  await expect(second.getByRole('button', { name: 'Potvrdi brisanje' })).toHaveCount(0);
  expect(state.token).toBe(token);
  expect(state.writes).toHaveLength(0);
  await expect(page.getByLabel('Ime i prezime')).toHaveValue('Ana');
});

test('delete confirmation owns the lock until cancellation', async ({ context, page }) => {
  const state = await setup(context);
  await openPartner(page);
  await page.getByRole('button', { name: 'Akcije za lice Ana', exact: true }).click();
  await page.getByRole('menuitem', { name: 'Obriši', exact: true }).click();
  await expect(page.getByRole('region', { name: 'Brisanje lica' })).toBeVisible();
  const second = await context.newPage();
  await openPartner(second);
  await second.getByRole('button', { name: 'Akcije za lice Ana', exact: true }).click();
  await second.getByRole('menuitem', { name: 'Izmeni', exact: true }).click();
  await expect(second.getByRole('alert')).toContainText('drugi korisnik');
  await expect(second.getByRole('form', { name: 'Izmena lica' })).toHaveCount(0);
  await page.getByRole('button', { name: 'Odustani', exact: true }).click();
  await expect.poll(() => state.token).toBeNull();
});

test('deleting broadcasts fresh contacts to another already-open partner dialog', async ({ context, page }) => {
  const state = await setup(context);
  await openPartner(page);
  const second = await context.newPage();
  await openPartner(second);
  await expect(second.locator('.partner-person')).toHaveCount(1);
  await page.getByRole('button', { name: 'Akcije za lice Ana', exact: true }).click();
  await page.getByRole('menuitem', { name: 'Obriši', exact: true }).click();
  await page.getByRole('button', { name: 'Potvrdi brisanje' }).click();
  await expect(page.getByText('Lice je obrisano.', { exact: true })).toBeVisible();
  await expect(second.locator('.partner-person')).toHaveCount(0);
  await expect(second.getByText('Nema evidentiranih kontakt osoba.')).toBeVisible();
  expect(state.people).toHaveLength(0);
});

test('independent browser session refreshes before opening cached partner details', async ({ browser, context, page }) => {
  const state = await setup(context);
  const independent = await browser.newContext();
  try {
    await setup(independent, state);
    const second = await independent.newPage();
    await second.goto('/ugovori/partneri');
    await expect(second.getByRole('button', { name: 'Moj partner', exact: true })).toBeVisible();
    await openPartner(page);
    await page.getByRole('button', { name: 'Akcije za lice Ana', exact: true }).click();
    await page.getByRole('menuitem', { name: 'Obriši', exact: true }).click();
    await page.getByRole('button', { name: 'Potvrdi brisanje' }).click();
    await expect.poll(() => state.people.length).toBe(0);
    await second.getByRole('button', { name: 'Moj partner', exact: true }).click();
    await expect(second.getByText('Nema evidentiranih kontakt osoba.')).toBeVisible();
    await expect(second.locator('.partner-person')).toHaveCount(0);
  } finally { await independent.close(); }
});


test('discard confirmation keeps draft and lease on cancel or Escape, releases on OK', async ({ context, page }) => {
  const state = await setup(context);
  await openPartner(page);
  await page.getByRole('button', { name: 'Akcije za lice Ana', exact: true }).click();
  await page.getByRole('menuitem', { name: 'Izmeni', exact: true }).click();
  await page.getByLabel('Ime i prezime').fill('Nesacuvan unos');
  const token = state.token;
  await page.getByRole('button', { name: 'Zatvori detalje partnera' }).click();
  const confirmation = page.getByRole('dialog', { name: 'Odustati od uređivanja lica?' });
  await expect(confirmation).toBeVisible();
  await confirmation.getByRole('button', { name: 'Otkaži' }).click();
  await expect(page.getByLabel('Ime i prezime')).toHaveValue('Nesacuvan unos');
  expect(state.token).toBe(token);
  expect(state.releases).toBe(0);
  await page.keyboard.press('Escape');
  await expect(confirmation).toBeVisible();
  await page.keyboard.press('Escape');
  await expect(confirmation).toHaveCount(0);
  await expect(page.getByLabel('Ime i prezime')).toHaveValue('Nesacuvan unos');
  expect(state.token).toBe(token);
  await page.getByRole('button', { name: 'Zatvori detalje partnera' }).click();
  await confirmation.getByRole('button', { name: 'OK', exact: true }).click();
  await expect(page.getByRole('dialog')).toHaveCount(0);
  await expect.poll(() => state.token).toBeNull();
  expect(state.writes).toHaveLength(0);
});


test('open contract adds contact for its supplier, preserves cancelled draft and refreshes contacts', async ({ context, page }) => {
  const state = await setup(context);
  let open = true;
  await context.route('**/api/ugo_evid/*/detalji', route => route.fulfill({ json: {
    id_ugo_evid: 2, id_sap_dobavljac: 20, otvoren_ug: open ? 'X' : '',
    br_ugovor: '4600012345', naziv: 'Moj partner', lica_dobavljaca: state.people,
  } }));
  await page.goto('/ugovori/detalji/2');
  await page.getByRole('button', { name: '+ Unos novog lica', exact: true }).click();
  await page.getByLabel('Ime i prezime').fill('Kontakt iz ugovora');
  await page.getByRole('button', { name: 'Odustani', exact: true }).click();
  await page.getByRole('button', { name: 'Otkaži', exact: true }).click();
  await expect(page.getByLabel('Ime i prezime')).toHaveValue('Kontakt iz ugovora');
  await page.getByRole('button', { name: 'Sačuvaj', exact: true }).click();
  await expect(page.getByRole('dialog')).toHaveCount(0);
  await expect(page.locator('.partner-person').filter({ hasText: 'Kontakt iz ugovora' })).toBeVisible();
  expect(state.writes).toHaveLength(1);
  expect(state.writes[0].id_sap_dobavljac).toBe(20);
  await expect(page.getByText('Lice je dodato.', { exact: true })).toBeVisible();
  open = false;
  await page.reload();
  await expect(page.getByText('Zatvoren ugovor', { exact: true })).toBeVisible();
  await expect(page.getByRole('button', { name: '+ Unos novog lica', exact: true })).toHaveCount(0);
});


test('contract contact menus edit and delete with shared locking and refreshed data', async ({ context, page }) => {
  const state = await setup(context);
  await context.route('**/api/ugo_evid/*/detalji', route => route.fulfill({ json: {
    id_ugo_evid: 2, id_sap_dobavljac: 20, otvoren_ug: 'X',
    br_ugovor: '4600012345', naziv: 'Moj partner', lica_dobavljaca: state.people,
    ime: 'Istorijski kontakt', email: 'snapshot@example.test',
  } }));
  await page.goto('/ugovori/detalji/2');
  await expect(page.getByRole('button', { name: 'Akcije za lice Istorijski kontakt' })).toHaveCount(0);
  await page.getByRole('button', { name: 'Akcije za lice Ana', exact: true }).click();
  await page.getByRole('menuitem', { name: 'Izmeni', exact: true }).click();
  await expect(page.getByLabel('Ime i prezime')).toHaveValue('Ana');
  const second = await context.newPage();
  await second.goto('/ugovori/detalji/2');
  await second.getByRole('button', { name: 'Akcije za lice Ana', exact: true }).click();
  await second.getByRole('menuitem', { name: 'Obriši', exact: true }).click();
  await expect(second.getByRole('alert')).toContainText('drugi korisnik');
  await expect(second.getByRole('dialog')).toHaveCount(0);
  await page.getByLabel('Ime i prezime').fill('Ana iz ugovora');
  await page.getByRole('button', { name: 'Sačuvaj', exact: true }).click();
  await expect(page.getByRole('dialog')).toHaveCount(0);
  await expect(page.getByRole('button', { name: 'Akcije za lice Ana iz ugovora', exact: true })).toBeVisible();
  expect(state.writes[0].version).toBe('1');
  expect(state.writes[0].lock_token).toHaveLength(64);
  await page.getByRole('button', { name: 'Akcije za lice Ana iz ugovora', exact: true }).click();
  await page.getByRole('menuitem', { name: 'Obriši', exact: true }).click();
  await page.getByRole('button', { name: 'Odustani', exact: true }).click();
  await expect.poll(() => state.token).toBeNull();
  expect(state.people).toHaveLength(1);
  await page.getByRole('button', { name: 'Akcije za lice Ana iz ugovora', exact: true }).click();
  await page.getByRole('menuitem', { name: 'Obriši', exact: true }).click();
  await page.getByRole('button', { name: 'Potvrdi brisanje' }).click();
  await expect(page.getByRole('dialog')).toHaveCount(0);
  await expect(page.getByRole('button', { name: 'Akcije za lice Ana iz ugovora', exact: true })).toHaveCount(0);
  await expect(second.getByRole('button', { name: 'Akcije za lice Ana iz ugovora', exact: true })).toHaveCount(0);
  expect(state.people).toHaveLength(0);
});
