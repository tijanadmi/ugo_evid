import { test, expect } from '@playwright/test';

const contract = {
  id_ugo_evid: 1, id_ugo_org: 3, id_sap_ugovor: 101, id_sap_dobavljac: 20,
  br_ugovor: '4600012345', godina: '2026', predmet_ugovora: 'Održavanje telekomunikacione opreme i sistema',
  naziv: 'Primer dobavljača d.o.o.', sluzba: 'CTKS', otvoren_ug: 'X',
  jn: 'JN-12/2026', pocetak_ug: '2026-01-15T00:00:00Z', kraj_ug: null,
  vrednost_ug: 1250000, valuta_ug: 'RSD', ime: 'Petar Petrović',
  telefon: '011 123 4567', email: 'petar@example.test',
  odg_zap_1: '001', naziv_odg_zap_1: 'Prvo odgovorno lice',
  odg_zap_2: '002', naziv_odg_zap_2: 'Drugo odgovorno lice',
  odg_zap_3: '003', naziv_odg_zap_3: 'Treće odgovorno lice',
  odg_zap_4: '004', naziv_odg_zap_4: 'Četvrto odgovorno lice',
  odg_zap_5: '005', naziv_odg_zap_5: 'Peto odgovorno lice',
  odg_zap_6: '006', naziv_odg_zap_6: 'Šesto odgovorno lice',
};
const future = (minutes) => new Date(Date.now() + minutes * 60000).toISOString();

async function mockAPI(page, options = {}) {
  const calls = [];
  let renewals = 0;
  await page.route('**/api/**', async route => {
    const req = route.request();
    const url = new URL(req.url());
    calls.push(url);
    const json = (body, status = 200) => route.fulfill({ status, contentType: 'application/json', body: JSON.stringify(body) });
    if (url.pathname === '/api/users/login') {
      if (req.postDataJSON().password === 'wrong') return json({ error: 'neuspesna prijava' }, 401);
      expect(req.postDataJSON()).toEqual({ username: 'test.user', password: 'test-password' });
      return json({ user: { username: 'test.user', id: 7 }, access_token: 'access', refresh_token: 'refresh',
        access_token_expires_at: future(options.expired ? -1 : 15), refresh_token_expires_at: future(120) });
    }
    if (url.pathname === '/api/tokens/renew_access') {
      renewals++;
      expect(req.postDataJSON()).toEqual({ refresh_token: 'refresh' });
      if (options.refreshFails) return json({ error: 'expired' }, 401);
      return json({ access_token: 'renewed', access_token_expires_at: future(15) });
    }
    expect(req.headers().authorization).toBe(`Bearer ${options.expired ? 'renewed' : 'access'}`);
    if (url.pathname === '/api/ugo_org') return json({ total: 2, items: [{ id: 3, sifra: 'CTKS', naziv: 'Centar za telekomunikacione sisteme' }, { id: 4, sifra: 'CITI', naziv: 'Centar za IT infrastrukturu' }] });
    if (url.pathname.startsWith('/api/ugo_evid/')) {
      if (url.pathname.endsWith('/detalji')) {
        if (options.detailError) return json({ error: 'Detalji nisu dostupni' }, options.detailError);
        return json({ ...contract, ...(options.noContacts ? { ime: '', telefon: '', email: '' } : {}), id_ugo_evid: 2, br_ugovor: '4600099999', lica_dobavljaca: options.noContacts ? [] : [{ id: 7, ime: 'Ana Anić', radno_mesto: 'Menadžer', telefon: '011 999', email: 'ana@example.test', rola_lica: 'Service Level Manager' }] });
      }
      if (options.failContracts) return json({ error: 'pregled ugovora trenutno nije dostupan' }, 500);
      if (url.searchParams.get('id_ugo_org') === '4') return json({ total: 0, items: [] });
      if (url.searchParams.get('page_id') === '2') return json({ total: 21, items: [{ ...contract, id_ugo_evid: 2, br_ugovor: '4600099999' }] });
      return json({ total: 21, items: [{ ...contract, otvoren_ug: url.pathname.endsWith('/zatvoreni') ? '' : 'X' }] });
    }
    return json({ error: 'unknown test route' }, 404);
  });
  return { calls, renewals: () => renewals };
}

async function login(page) {
  await page.goto('/login');
  await page.getByLabel('Korisničko ime', { exact: true }).fill('test.user');
  await page.getByLabel('Lozinka', { exact: true }).fill('test-password');
  await page.getByRole('button', { name: 'Prijavi se', exact: true }).click();
  await expect(page.getByRole('heading', { name: 'Aplikacije', exact: true })).toBeVisible();
}

test('login, portal, both lists, organization filter, pagination and details', async ({ page }) => {
  const { calls } = await mockAPI(page);
  await login(page);
  await page.screenshot({ path: 'test-results/portal.png', fullPage: true });
  await page.getByRole('link', { name: /Evidencija ugovora/ }).click();
  await expect(page).toHaveURL(/\/ugovori\/otvoreni/);
  await expect(page.getByRole('link', { name: '4600012345', exact: true })).toBeVisible();
  await expect(page.getByRole('columnheader', { name: 'Odgovorna lica', exact: true })).toBeVisible();
  await expect(page.getByRole('columnheader', { name: 'Vrednost', exact: true })).toHaveCount(0);
  const contactCell = page.locator('.contact-cell');
  await expect(contactCell).toContainText('011 123 4567');
  const emailBox = await contactCell.getByText('petar@example.test', { exact: true }).boundingBox();
  const phoneBox = await contactCell.getByText('011 123 4567', { exact: true }).boundingBox();
  expect(phoneBox.y).toBeGreaterThan(emailBox.y);
  const people = page.getByRole('list', { name: 'Odgovorna lica' }).getByRole('listitem');
  await expect(people).toHaveCount(6);
  await expect(people.first()).toHaveText('001 — Prvo odgovorno lice');
  await expect(people.last()).toHaveText('006 — Šesto odgovorno lice');
  const first = await people.first().boundingBox();
  const second = await people.nth(1).boundingBox();
  expect(second.y).toBeGreaterThan(first.y);
  await page.screenshot({ path: 'test-results/otvoreni.png', fullPage: true });
  await page.getByLabel('Organizaciona jedinica', { exact: true }).selectOption('3');
  await expect.poll(() => calls.some(url => url.pathname.endsWith('/otvoreni') && url.searchParams.get('id_ugo_org') === '3')).toBeTruthy();
  await page.getByRole('button', { name: 'Sledeća stranica' }).click();
  await expect(page.getByRole('link', { name: '4600099999', exact: true })).toBeVisible();
  await page.getByRole('link', { name: '4600099999', exact: true }).click();
  await expect(page).toHaveURL(/\/ugovori\/detalji\/2$/);
  await expect(page.getByRole('heading', { name: /4600099999.*Održavanje/ })).toBeVisible();
  await expect(page.getByRole('list', { name: 'Odgovorna lica' }).getByText(/Šesto odgovorno lice/)).toBeVisible();
  await expect(page.getByText('Service Level Manager', { exact: true })).toBeVisible();
  const supplierRows = page.getByRole('list', { name: 'Lica dobavljača' }).getByRole('listitem');
  await expect(supplierRows).toHaveCount(2);
  await expect(supplierRows.first()).toContainText('Kontakt za ovaj ugovor');
  await expect(supplierRows.first()).toContainText('Petar Petrović');
  await expect(page.getByText('011 999', { exact: true })).toBeVisible();
  await page.screenshot({ path: 'test-results/detalji.png', fullPage: true });
  await page.reload();
  await expect(page.getByText('Ana Anić', { exact: true })).toBeVisible();
  await page.getByRole('link', { name: 'Povratak na pregled' }).click();
  await expect(page).toHaveURL(/page_id=2.*id_ugo_org=3/);
  await page.getByRole('link', { name: 'Zatvoreni ugovori', exact: true }).click();
  await expect(page.getByRole('heading', { name: 'Zatvoreni ugovori', exact: true })).toBeVisible();
  await expect(page.getByRole('link', { name: '4600012345', exact: true })).toBeVisible();
  await expect.poll(() => calls.some(url => url.pathname.endsWith('/zatvoreni'))).toBeTruthy();
  await page.getByLabel('Organizaciona jedinica', { exact: true }).selectOption('4');
  await expect(page.getByRole('heading', { name: 'Nema ugovora za prikaz' })).toBeVisible();
  await page.getByLabel('Organizaciona jedinica', { exact: true }).selectOption('0');
  await expect(page.getByRole('link', { name: '4600012345', exact: true })).toBeVisible();
});

test('direct details handle empty contacts, missing contracts and retry', async ({ page }) => {
  const options = { noContacts: true };
  await mockAPI(page, options);
  await login(page);
  await page.goto('/ugovori/detalji/2');
  await expect(page.getByText('Nema evidentiranih lica za ovog dobavljača.')).toBeVisible();
  options.detailError = 404;
  await page.reload();
  await expect(page.getByRole('heading', { name: 'Ugovor nije pronađen' })).toBeVisible();
  options.detailError = 500;
  await page.reload();
  await expect(page.getByRole('heading', { name: 'Detalji nisu dostupni' })).toBeVisible();
  options.detailError = 0;
  await page.getByRole('button', { name: 'Pokušaj ponovo' }).click();
  await expect(page.getByRole('heading', { name: /4600099999.*Održavanje/ })).toBeVisible();
});

test('protected routes, login error, session restoration and logout', async ({ page }) => {
  await mockAPI(page);
  await page.goto('/ugovori/zatvoreni');
  await expect(page).toHaveURL(/\/login$/);
  await page.getByLabel('Korisničko ime', { exact: true }).fill('test.user');
  await page.getByLabel('Lozinka', { exact: true }).fill('wrong');
  await page.getByRole('button', { name: 'Prijavi se', exact: true }).click();
  await expect(page.getByRole('alert')).toContainText('nisu ispravni');
  await expect(page.getByLabel('Lozinka', { exact: true })).toHaveValue('');
  await login(page);
  await page.reload();
  await expect(page.getByRole('heading', { name: 'Aplikacije', exact: true })).toBeVisible();
  await page.getByRole('button', { name: 'Odjavi se' }).click();
  await expect(page).toHaveURL(/\/login$/);
  await page.goto('/ugovori/otvoreni');
  await expect(page).toHaveURL(/\/login$/);
  expect(await page.evaluate(() => sessionStorage.getItem('ugo-evid-session'))).toBeNull();
});

test('expired access token is renewed once for concurrent requests', async ({ page }) => {
  const mock = await mockAPI(page, { expired: true });
  await login(page);
  await page.getByRole('link', { name: /Evidencija ugovora/ }).click();
  await expect(page.getByRole('link', { name: '4600012345', exact: true })).toBeVisible();
  expect(mock.renewals()).toBe(1);
});

test('expired refresh returns to login', async ({ page }) => {
  await mockAPI(page, { expired: true, refreshFails: true });
  await login(page);
  await page.getByRole('link', { name: /Evidencija ugovora/ }).click();
  await expect(page).toHaveURL(/\/login$/);
});

test('server error can be retried', async ({ page }) => {
  const options = { failContracts: true };
  await mockAPI(page, options);
  await login(page);
  await page.getByRole('link', { name: /Evidencija ugovora/ }).click();
  await expect(page.getByRole('heading', { name: 'Pregled trenutno nije dostupan' })).toBeVisible();
  options.failContracts = false;
  await page.getByRole('button', { name: 'Pokušaj ponovo', exact: true }).click();
  await expect(page.getByRole('link', { name: '4600012345', exact: true })).toBeVisible();
});

test('mobile navigation and table stay inside viewport', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await mockAPI(page);
  await login(page);
  await page.getByRole('link', { name: /Evidencija ugovora/ }).click();
  await expect(page.getByRole('link', { name: '4600012345', exact: true })).toBeVisible();
  await page.getByRole('button', { name: 'Otvori meni' }).click();
  await page.getByRole('link', { name: 'Zatvoreni ugovori', exact: true }).click();
  await expect(page.getByRole('heading', { name: 'Zatvoreni ugovori', exact: true })).toBeVisible();
  await expect(page.getByRole('link', { name: '4600012345', exact: true })).toBeVisible();
  const overflow = await page.evaluate(() => ({
    width: document.documentElement.scrollWidth,
    viewport: window.innerWidth,
    elements: [...document.querySelectorAll('body *')].filter(el => !el.closest('.table-scroll') && el.getBoundingClientRect().right > window.innerWidth).map(el => `${el.tagName}.${el.className}: ${el.getBoundingClientRect().right}`),
  }));
  expect(overflow.width, JSON.stringify(overflow)).toBeLessThanOrEqual(overflow.viewport);
  await page.screenshot({ path: 'test-results/mobile.png', fullPage: true });
});
