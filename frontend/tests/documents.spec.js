import { test, expect } from '@playwright/test';

function samplePDF() {
 const stream = 'BT /F1 18 Tf 30 100 Td (UGO document preview) Tj ET';
 const objects = ['<< /Type /Catalog /Pages 2 0 R >>', '<< /Type /Pages /Kids [3 0 R] /Count 1 >>', '<< /Type /Page /Parent 2 0 R /MediaBox [0 0 400 200] /Resources << /Font << /F1 4 0 R >> >> /Contents 5 0 R >>', '<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>', `<< /Length ${stream.length} >>\nstream\n${stream}\nendstream`];
 let pdf = '%PDF-1.4\n'; const offsets = [0];
 objects.forEach((object, index) => { offsets.push(pdf.length); pdf += `${index + 1} 0 obj\n${object}\nendobj\n`; });
 const xref = pdf.length;
 pdf += `xref\n0 ${objects.length + 1}\n0000000000 65535 f \n`;
 pdf += offsets.slice(1).map(offset => `${String(offset).padStart(10, '0')} 00000 n \n`).join('');
 return pdf + `trailer\n<< /Size ${objects.length + 1} /Root 1 0 R >>\nstartxref\n${xref}\n%%EOF`;
}


const types = [{ id: 1, sifra: 'UGOVOR', naziv: 'Ugovor' }, { id: 2, sifra: 'RESENJE', naziv: 'Rešenje za operativno praćenje' }, { id: 3, sifra: 'DODATNO_RESENJE', naziv: 'Dodatno rešenje' }, { id: 4, sifra: 'ZAPISNIK', naziv: 'Zapisnik' }];
async function setup(context, overrides = {}) {
 const state = { docs: [{ id: 11, id_ugo_evid: 9, id_tip_doc: 1, tip_naziv: 'Ugovor', naziv: 'Potpisani ugovor', originalni_naziv: 'ugovor.pdf', mime_tip: 'application/pdf', velicina_bajt: 100, version: '1' }], token: null, uploads: [], writes: [], closed: false, forbidden: false, expired: false, types, ...overrides };
 await context.addInitScript(() => sessionStorage.setItem('ugo-evid-session', JSON.stringify({ access_token: 'access', refresh_token: 'refresh', access_token_expires_at: new Date(Date.now() + 3600000).toISOString(), refresh_token_expires_at: new Date(Date.now() + 7200000).toISOString(), user: { username: 'test.user' } })));
 await context.route('**/api/**', async route => {
  const req = route.request(), url = new URL(req.url()), method = req.method();
  const json = (body, status = 200) => route.fulfill({ status, contentType: 'application/json', body: JSON.stringify(body) });
    if (new URL(route.request().url()).pathname === '/api/ugo_tip_ugovora') return json({ items: [{id:1,sifra:'ODRZAVANJE',naziv:'Odrzavanje softvera'}] });
    if (new URL(route.request().url()).pathname.endsWith('/podesavanja')) return json({ settings: {id_tip_ugovora:0,version:'1',odrzavanje:null}, document_types:[] });
  if (url.pathname.endsWith('/detalji')) return json({ id_ugo_evid: 9, id_ugo_org: 3, id_sap_dobavljac: 20, br_ugovor: '460', predmet_ugovora: 'Oprema', naziv: 'Partner', otvoren_ug: state.closed ? '' : 'X', lica_dobavljaca: [] });
  if (state.forbidden) return json({ error: 'Zabranjen pristup' }, 403);
  if (url.pathname.endsWith('/dokumenti')) {
   if (method === 'GET') return json({ items: state.docs, types: state.types, max_bytes: 1024 });
   expect(req.headers()['content-type']).toContain('multipart/form-data; boundary=');
   const raw = req.postDataBuffer().toString();
   const metadata = JSON.parse(raw.match(/name="metadata"\r\n\r\n([^\r]+)/)[1]);
   state.uploads.push(metadata);
   state.docs.push({ ...metadata, id: 12, id_ugo_evid: 9, originalni_naziv: 'novi.pdf', mime_tip: 'application/pdf', velicina_bajt: 100, version: '1', tip_naziv: types.find(type => type.id === metadata.id_tip_doc).naziv });
   return json({ id: 12 }, 201);
  }
  if (url.pathname.endsWith('/sadrzaj')) {
   expect(req.headers().authorization).toBe('Bearer access');
   return route.fulfill({ contentType: 'application/pdf', body: samplePDF() });
  }
  const body = req.postData() ? req.postDataJSON() : {};
  if (url.pathname.endsWith('/lock')) {
   if (method === 'POST') { if (state.token) return json({ error: 'Dokument trenutno menja drugi korisnik ili drugi tab.' }, 423); state.token = 'a'.repeat(64); }
   else if (method === 'DELETE') { if (body.lock_token === state.token) state.token = null; }
   else if (state.expired || body.lock_token !== state.token) return json({ error: 'Pravo izmene dokumenta je isteklo.' }, 409);
   return json({ lock_token: state.token, expires_at: new Date(Date.now() + 120000).toISOString() });
  }
  const id = Number(url.pathname.split('/').at(-1));
  const doc = state.docs.find(doc => doc.id === id);
  if (!doc) return json({ error: 'Dokument nije pronadjen.' }, 404);
  if (method === 'GET') return json(doc);
  if (body.lock_token !== state.token || body.version !== doc.version) return json({ error: 'Dokument je promenjen.' }, 409);
  state.writes.push(body); state.token = null;
  if (method === 'DELETE') state.docs = state.docs.filter(doc => doc.id !== id);
  else Object.assign(doc, body, { version: String(Number(doc.version) + 1) });
  return json({ status: 'ok' });
 });
 return state;
}
async function open(page) { await page.goto('/ugovori/detalji/9'); await expect(page.getByRole('region', { name: 'Dokumenta ugovora' })).toBeVisible(); }
async function edit(page) { await page.getByRole('button', { name: 'Akcije za dokument Potpisani ugovor' }).click(); await page.getByRole('menuitem', { name: 'Izmeni', exact: true }).click(); }

test('upload entry points preselect document types and refresh the list', async ({ context, page }) => {
 const state = await setup(context); await open(page);
 await page.getByRole('button', { name: 'Dodaj ugovor', exact: true }).click();
 await expect(page.getByLabel('Tip dokumenta')).toHaveValue('Ugovor');
 await page.getByRole('button', { name: 'Odustani', exact: true }).click();
 await page.getByRole('button', { name: 'OK', exact: true }).click();
 await page.getByRole('button', { name: 'Dodaj rešenje', exact: true }).click();
 await expect(page.getByLabel('Tip dokumenta')).toHaveValue('Rešenje za operativno praćenje');
 await page.getByLabel('Fajl', { exact: true }).setInputFiles({ name: 'novi.pdf', mimeType: 'application/pdf', buffer: Buffer.from('%PDF-1.4\n%%EOF') });
 await page.getByLabel('Naziv dokumenta', { exact: true }).fill('Resenje 2026');
 await expect(page.getByLabel('Datum dokumenta', { exact: true })).toHaveCount(0);
 await page.getByRole('button', { name: 'Sačuvaj dokument', exact: true }).click();
 await expect(page.getByRole('dialog')).toHaveCount(0);
 await expect(page.getByRole('region', { name: 'Dokumenta ugovora' })).toContainText('Resenje 2026');
 expect(state.uploads[0]).toMatchObject({ id_tip_doc: 2, naziv: 'Resenje 2026', datum_dokumenta: '' });
 await page.getByRole('button', { name: 'Dodaj zapisnik', exact: true }).click();
 await expect(page.getByLabel('Tip dokumenta')).toHaveValue('Zapisnik');
});

test('editing holds lease across cancellation; another tab cannot delete until save', async ({ context, page }) => {
 const state = await setup(context); await open(page); await edit(page);
 await page.getByLabel('Naziv dokumenta', { exact: true }).fill('Izmenjen naziv');
 await page.getByRole('button', { name: 'Odustani', exact: true }).click();
 await page.getByRole('button', { name: 'Otkaži', exact: true }).click();
 await expect(page.getByLabel('Naziv dokumenta', { exact: true })).toHaveValue('Izmenjen naziv');
 const second = await context.newPage(); await open(second);
 await second.getByRole('button', { name: 'Akcije za dokument Potpisani ugovor' }).click();
 await second.getByRole('menuitem', { name: 'Obriši', exact: true }).click();
 await expect(second.getByRole('alert')).toContainText('drugi korisnik');
 await expect(second.getByRole('dialog')).toHaveCount(0);
 await page.getByRole('button', { name: 'Sačuvaj dokument', exact: true }).click();
 await expect(page.getByRole('dialog')).toHaveCount(0);
 await expect(second.getByRole('button', { name: 'Akcije za dokument Izmenjen naziv' })).toBeVisible();
 expect(state.writes[0]).toMatchObject({ version: '1', lock_token: 'a'.repeat(64) });
 await second.getByRole('button', { name: 'Akcije za dokument Izmenjen naziv' }).click();
 await second.getByRole('menuitem', { name: 'Obriši', exact: true }).click();
 await second.getByRole('button', { name: 'Potvrdi brisanje' }).click();
 await expect(page.getByText('Nema dodatih dokumenata.')).toBeVisible();
 expect(state.docs).toHaveLength(0);
});

test('expired lease preserves metadata and prevents writing', async ({ context, page }) => {
 const state = await setup(context); await page.clock.install(); await open(page); await edit(page);
 await page.getByLabel('Naziv dokumenta', { exact: true }).fill('Nesacuvano');
 state.expired = true; await page.clock.fastForward(31000);
 await expect(page.getByRole('button', { name: 'Sačuvaj dokument', exact: true })).toBeDisabled();
 await expect(page.getByLabel('Naziv dokumenta', { exact: true })).toHaveValue('Nesacuvano');
 expect(state.writes).toHaveLength(0);
});

test('authenticated download and inline preview work for closed contracts without edit controls', async ({ context, page }) => {
 await setup(context, { closed: true }); await open(page);
 await expect(page.getByRole('button', { name: 'Dodaj ugovor', exact: true })).toHaveCount(0);
 await expect(page.getByRole('button', { name: 'Akcije za dokument Potpisani ugovor' })).toHaveCount(0);
 const downloading = page.waitForEvent('download');
 await page.getByRole('button', { name: 'Preuzmi', exact: true }).click();
 expect((await downloading).suggestedFilename()).toBe('ugovor.pdf');
 await page.getByRole('button', { name: 'Pregled', exact: true }).click();
 await expect(page.getByRole('dialog', { name: 'Potpisani ugovor' })).toBeVisible();
 await expect(page.locator('iframe')).toHaveAttribute('src', /^blob:/);
 const previewDialog = page.getByRole('dialog', { name: 'Potpisani ugovor' });
 const original = await previewDialog.boundingBox();
 await page.getByRole('button', { name: 'Uvećaj prozor', exact: true }).click();
 await expect(previewDialog).toHaveClass(/is-expanded/);
 expect((await previewDialog.boundingBox()).width).toBeGreaterThan(original.width);
 await page.getByRole('button', { name: 'Vrati veličinu', exact: true }).click();
 await expect(previewDialog).not.toHaveClass(/is-expanded/);
 expect(await previewDialog.evaluate(node => getComputedStyle(node).resize)).toBe('both');

 await expect(page.frameLocator('iframe').locator('embed')).toBeVisible();
 await page.screenshot({ path: 'test-results/document-preview.png', fullPage: true });
 await page.getByRole('button', { name: 'Zatvori dokument' }).click();
 await expect(page.getByRole('dialog')).toHaveCount(0);
});

test('foreign organization has no document actions', async ({ context, page }) => {
 await setup(context, { forbidden: true }); await open(page);
 await expect(page.getByText('Dokumenta su dostupna korisnicima organizacione jedinice ugovora.')).toBeVisible();
 await expect(page.getByRole('button', { name: 'Dodaj ugovor', exact: true })).toHaveCount(0);
 await expect(page.getByRole('button', { name: 'Preuzmi', exact: true })).toHaveCount(0);
});

test('oversize upload remains in the form without sending file', async ({ context, page }) => {
 const state = await setup(context); await open(page);
 await page.getByRole('button', { name: 'Dodaj ugovor', exact: true }).click();
 await page.getByLabel('Fajl', { exact: true }).setInputFiles({ name: 'large.pdf', mimeType: 'application/pdf', buffer: Buffer.alloc(2048) });
 await page.getByRole('button', { name: 'Sačuvaj dokument', exact: true }).click();
 await expect(page.getByRole('alert')).toContainText('Izaberite fajl');
 expect(state.uploads).toHaveLength(0);
 await expect(page.getByRole('form', { name: 'Dodavanje dokumenta' })).toBeVisible();
});

test('missing contract type shows a toast instead of uploading under another type', async ({ context, page }) => {
 const state = await setup(context, { types: types.slice(1) }); await open(page);
 await page.getByRole('button', { name: 'Dodaj ugovor', exact: true }).click();
 await expect(page.getByRole('alert')).toContainText('Tip dokumenta UGOVOR nije dostupan');
 await expect(page.getByRole('dialog')).toHaveCount(0);
 expect(state.uploads).toHaveLength(0);
});

async function dropFiles(page, names) {
 const transfer = await page.evaluateHandle(names => {
  const transfer = new DataTransfer();
  for (const name of names) transfer.items.add(new File(['%PDF-1.4\n%%EOF'], name, { type: 'application/pdf' }));
  return transfer;
 }, names);
 const header = page.getByRole('dialog').getByRole('heading', { name: 'Dodaj dokument' });
 await header.dispatchEvent('dragenter', { dataTransfer: transfer });
 await header.dispatchEvent('dragover', { dataTransfer: transfer });
 await header.dispatchEvent('drop', { dataTransfer: transfer });
 await transfer.dispose();
}

test('dropping on dialog selects a file and waits for save; chooser remains available', async ({ context, page }) => {
 const state = await setup(context); await open(page);
 await page.getByRole('button', { name: 'Dodaj ugovor', exact: true }).click();
 await dropFiles(page, ['prevucen.pdf']);
 await expect(page.getByLabel('Naziv dokumenta', { exact: true })).toHaveValue('prevucen.pdf');
 expect(await page.getByLabel('Fajl', { exact: true }).evaluate(input => input.files[0].name)).toBe('prevucen.pdf');
 expect(state.uploads).toHaveLength(0);
 await page.getByLabel('Naziv dokumenta', { exact: true }).fill('Moj naziv');
 await dropFiles(page, ['jedan.pdf', 'dva.pdf']);
 await expect(page.getByRole('alert')).toContainText('Prevucite jedan dokument');
 expect(await page.getByLabel('Fajl', { exact: true }).evaluate(input => input.files[0].name)).toBe('prevucen.pdf');
 await page.getByLabel('Fajl', { exact: true }).setInputFiles({ name: 'izabran.pdf', mimeType: 'application/pdf', buffer: Buffer.from('%PDF-1.4') });
 await expect(page.getByLabel('Naziv dokumenta', { exact: true })).toHaveValue('Moj naziv');
 await dropFiles(page, ['poslednji.pdf']);
 await expect(page.getByLabel('Naziv dokumenta', { exact: true })).toHaveValue('Moj naziv');
 await page.getByRole('button', { name: 'Sačuvaj dokument', exact: true }).click();
 await expect(page.getByRole('dialog')).toHaveCount(0);
 expect(state.uploads).toHaveLength(1);
 expect(state.uploads[0].naziv).toBe('Moj naziv');
});

test('image preview supports zoom and fit to width', async ({ context, page }) => {
 await setup(context, { docs: [{ id: 11, naziv: 'Slika ugovora', originalni_naziv: 'ugovor.png', mime_tip: 'image/png', velicina_bajt: 100, version: '1' }] });
 await context.route('**/api/ugo_doc/11/sadrzaj?preview=1', route => route.fulfill({ contentType: 'image/png', body: Buffer.from('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+jRZkAAAAASUVORK5CYII=', 'base64') }));
 await open(page);
 await page.getByRole('button', { name: 'Pregled', exact: true }).click();
 const picture = page.getByRole('img', { name: 'Slika ugovora' });
 await expect(picture).toBeVisible();
 const originalWidth = (await picture.boundingBox()).width;
 await page.getByRole('button', { name: 'Uvećaj sliku', exact: true }).click();
 expect((await picture.boundingBox()).width).toBeGreaterThan(originalWidth);
 await expect(page.getByRole('group', { name: 'Zum slike' })).toContainText('125%');
 await page.getByRole('button', { name: 'Prilagodi širini', exact: true }).click();
 await expect(page.getByRole('group', { name: 'Zum slike' })).toContainText('100%');
 await page.getByRole('button', { name: 'Zatvori dokument', exact: true }).click();
 await expect(page.getByRole('dialog')).toHaveCount(0);
});
