import { test, expect } from '@playwright/test';

async function setup(context, overrides = {}) {
 const state = { type: 0, version: 1, maintenance: null, token:null, expired:false, writes: [], closed: false, conflict: false, ...overrides };
 await context.addInitScript(() => sessionStorage.setItem('ugo-evid-session', JSON.stringify({ access_token:'access',refresh_token:'refresh',access_token_expires_at:new Date(Date.now()+3600000).toISOString(),refresh_token_expires_at:new Date(Date.now()+7200000).toISOString(),user:{username:'test.user'} })));
 await context.route('**/api/**', async route => {
  const req = route.request(), path = new URL(req.url()).pathname;
  const json = (body,status=200) => route.fulfill({status,contentType:'application/json',body:JSON.stringify(body)});
    if(new URL(route.request().url()).pathname.endsWith('/podsetnici')) return json({items:[]});
    if(new URL(route.request().url()).pathname.endsWith('/pracenje')) return json({periodi:[],zahtevi:[],fondovi:[]});
  if(path.endsWith('/detalji')) return json({id_ugo_evid:9,id_ugo_org:3,br_ugovor:'460',predmet_ugovora:'Odrzavanje',otvoren_ug:state.closed?'':'X',lica_dobavljaca:[]});
  if(path.endsWith('/ugo_tip_ugovora')) return json({items:[{id:1,sifra:'ODRZAVANJE',naziv:'Odrzavanje softvera'},{id:2,sifra:'LICENCE',naziv:'Licence'}]});
  if(path.endsWith('/dokumenti')) return json({items:[],types:[]});
  if(path.endsWith('/podesavanja')) return json({settings:{id_tip_ugovora:state.type,version:String(state.version),odrzavanje:state.maintenance,sap_pocetak:'2026-01-01',sap_kraj:'2026-12-31',vrste_rada:[{id:1,sifra:'INZENJER',naziv:'Inzenjer/programer'},{id:2,sifra:'TEHNICAR',naziv:'Tehnicar'},{id:3,sifra:'KONSULTANT',naziv:'Konsultant'}]},document_types:[{id:5,sifra:'POTVRDA_TEH',naziv:'Potvrda tehnicke podrske'},{id:6,sifra:'ZAPISNIK_TEH',naziv:'Zapisnik tehnicke podrske'}]});
  const body=req.postData()?req.postDataJSON():{};
  if(path.endsWith('/lock')){
   if(req.method()==='POST'){if(state.token)return json({error:'Odrzavanje menja drugi korisnik.'},423);state.token='a'.repeat(64);}
   else if(req.method()==='DELETE'){if(body.lock_token===state.token)state.token=null;}
   else if(state.expired || body.lock_token!==state.token)return json({error:'Pravo izmene je isteklo.'},409);
   return json({lock_token:state.token,expires_at:new Date(Date.now()+120000).toISOString()});
  }
  if(path.endsWith('/tip')) {if(body.version!==String(state.version))return json({error:'Podaci su promenjeni.'},409);state.type=body.id_tip_ugovora;state.version++;state.writes.push(body);return json({status:'ok'});}
  if(path.endsWith('/odrzavanje') && req.method()==='PUT'){if(body.lock_token!==state.token || body.version!==state.maintenance.version)return json({error:'Podaci su promenjeni.'},409);state.maintenance={...state.maintenance,...body,version:String(Number(body.version)+1)};state.token=null;state.writes.push(body);return json({status:'ok'});}
  if(path.endsWith('/odrzavanje')) {if(state.maintenance || state.conflict)return json({error:'Odrzavanje vec postoji.'},409);state.maintenance={...body,id:1,version:'1'};state.writes.push(body);return json({status:'ok'},201);}
  return json({error:'Unknown route'},404);
 });
 return state;
}
async function open(page){await page.goto('/ugovori/detalji/9');await expect(page.getByRole('region',{name:'Tip ugovora i održavanje'})).toBeVisible();}
async function fill(page){
 await page.getByRole('button',{name:'Unesi održavanje'}).click();
 await expect(page.getByText(/Važenje ugovora iz SAP-a:/)).toContainText('01.01.2026');

 await page.getByLabel('Dokument tehničke podrške',{exact:true}).selectOption('5');
}
test('existing contract gets a type and basic maintenance settings',async({context,page})=>{
 const state=await setup(context);await open(page);
 await page.getByRole('button',{name:'Odredi tip ugovora'}).click();
 await page.getByRole('combobox',{name:'Tip ugovora',exact:true}).selectOption('1');
 await page.getByRole('button',{name:'Sačuvaj podešavanja'}).click();
 await expect(page.getByRole('dialog')).toHaveCount(0);
 await fill(page);
 await page.getByLabel('Razvojna podrška',{exact:true}).selectOption('true');
 await page.getByLabel('Period potvrđivanja tehničke podrške',{exact:true}).selectOption('3');
 await page.getByLabel('Inzenjer/programer — ugovoreni časovi',{exact:true}).fill('100');
 await page.screenshot({path:'test-results/maintenance-form.png'});
 await page.getByRole('button',{name:'Sačuvaj podešavanja'}).click();
 await expect(page.getByRole('dialog')).toHaveCount(0);
 await expect(page.getByRole('region',{name:'Tip ugovora i održavanje'})).toContainText('Tehnička i razvojna');
 expect(state.maintenance).toMatchObject({teh_period_meseci:3,teh_podsetnik_dana:7,raz_podsetnik_dana:3,id_teh_tip_doc:5});
 await expect(page.getByRole('button',{name:'Unesi održavanje'})).toHaveCount(0);
 await expect(page.getByRole('button',{name:'Odredi tip ugovora'})).toHaveCount(0);
});
test('custom periods and technical-only maintenance are saved',async({context,page})=>{
 const state=await setup(context,{type:1});await open(page);await fill(page);
 await page.getByLabel('Period potvrđivanja tehničke podrške',{exact:true}).selectOption('custom');
 await page.getByRole('button',{name:'Sačuvaj podešavanja'}).click();
 await expect(page.getByRole('dialog')).toHaveCount(0);
 expect(state.maintenance.teh_period_meseci).toBeNull();expect(state.maintenance.ima_razvojnu_podrsku).toBe(false);
});
test('concurrent creation conflict preserves input and blocks retry',async({context,page})=>{
 const state=await setup(context,{type:1,conflict:true});await open(page);await fill(page);
 await page.getByRole('button',{name:'Sačuvaj podešavanja'}).click();
 await expect(page.getByRole('alert')).toContainText('vec postoji');
 await expect(page.getByRole('button',{name:'Sačuvaj podešavanja'})).toBeDisabled();
 await expect(page.getByLabel('Dokument tehničke podrške',{exact:true})).toHaveValue('5');
 expect(state.writes).toHaveLength(0);
});
test('closed contract has no settings write actions',async({context,page})=>{
 await setup(context,{type:1,closed:true});await open(page);
 await expect(page.getByRole('button',{name:'Unesi održavanje'})).toHaveCount(0);
 await expect(page.getByRole('button',{name:'Odredi tip ugovora'})).toHaveCount(0);
});

test('stale type editor cannot overwrite a change from another session',async({context,page})=>{
 const state=await setup(context);await open(page);
 await page.getByRole('button',{name:'Odredi tip ugovora'}).click();
 await page.getByRole('combobox',{name:'Tip ugovora',exact:true}).selectOption('1');
 state.type=2;state.version++;
 await page.getByRole('button',{name:'Sačuvaj podešavanja'}).click();
 await expect(page.getByRole('alert')).toContainText('promenjeni');
 await expect(page.getByRole('button',{name:'Sačuvaj podešavanja'})).toBeDisabled();
 expect(state.type).toBe(2);expect(state.writes).toHaveLength(0);
});

const existing={id:1,version:'1',pocetak_odrzavanja:'',datum_potpisa:'2025-12-20',datum_kraja:'2026-12-31',ima_razvojnu_podrsku:true,teh_period_meseci:3,id_teh_tip_doc:5,teh_podsetnik_dana:7,raz_podsetnik_dana:3,fondovi:[{id_vrsta_rada:1,ugovoreno_casova:100}]};
test('maintenance editing locks other tabs, preserves draft on cancel and saves funds with start override',async({context,page})=>{
 const state=await setup(context,{type:1,maintenance:{...existing}});await open(page);
 await page.getByRole('button',{name:'Izmeni održavanje'}).click();
 await page.getByLabel('Početak održavanja',{exact:true}).selectOption('true');
 await page.getByLabel('Drugi datum početka',{exact:true}).fill('2026-02-01');
 await page.getByLabel('Tehnicar — ugovoreni časovi',{exact:true}).fill('40');
 await page.getByRole('button',{name:'Odustani',exact:true}).click();
 await page.getByRole('button',{name:'Otkaži',exact:true}).click();
 await expect(page.getByLabel('Drugi datum početka',{exact:true})).toHaveValue('01.02.2026');
 const other=await context.newPage();await open(other);
 await other.getByRole('button',{name:'Izmeni održavanje'}).click();
 await expect(other.getByRole('alert')).toContainText('drugi korisnik');
 await expect(other.getByRole('dialog')).toHaveCount(0);
 await page.getByRole('button',{name:'Sačuvaj podešavanja'}).click();
 await expect(page.getByRole('dialog')).toHaveCount(0);
 expect(state.maintenance.pocetak_odrzavanja).toBe('2026-02-01');
 expect(state.maintenance.fondovi.find(f=>f.id_vrsta_rada===2).ugovoreno_casova).toBe(40);
 expect(state.maintenance.datum_potpisa).toBe('2025-12-20');expect(state.token).toBeNull();
 await other.getByRole('button',{name:'Izmeni održavanje'}).click();
 await expect(other.getByLabel('Drugi datum početka',{exact:true})).toHaveValue('01.02.2026');
 await other.getByRole('button',{name:'Odustani',exact:true}).click();
 await other.getByRole('button',{name:'OK',exact:true}).click();
 await expect.poll(()=>state.token).toBeNull();
});
test('expired maintenance lease disables save and retains hours',async({context,page})=>{
 const state=await setup(context,{type:1,maintenance:{...existing}});await page.clock.install();await open(page);
 await page.getByRole('button',{name:'Izmeni održavanje'}).click();
 await page.getByLabel('Inzenjer/programer — ugovoreni časovi',{exact:true}).fill('120');
 state.expired=true;await page.clock.fastForward(31000);
 await expect(page.getByRole('button',{name:'Sačuvaj podešavanja'})).toBeDisabled();
 await expect(page.getByLabel('Inzenjer/programer — ugovoreni časovi',{exact:true})).toHaveValue('120');expect(state.writes).toHaveLength(0);
});

test('saved custom month period is retained and override can revert to SAP',async({context,page})=>{
 const state=await setup(context,{type:1,maintenance:{...existing,teh_period_meseci:2,pocetak_odrzavanja:'2026-02-01'}});await open(page);
 await page.getByRole('button',{name:'Izmeni održavanje'}).click();
 await expect(page.getByLabel('Period potvrđivanja tehničke podrške',{exact:true})).toHaveValue('other');
 await expect(page.getByLabel('Broj meseci',{exact:true})).toHaveValue('2');
 await page.getByLabel('Početak održavanja',{exact:true}).selectOption('false');
 await page.getByRole('button',{name:'Sačuvaj podešavanja'}).click();
 await expect(page.getByRole('dialog')).toHaveCount(0);
 expect(state.maintenance.pocetak_odrzavanja).toBe('');expect(state.maintenance.teh_period_meseci).toBe(2);
 expect(state.maintenance.datum_potpisa).toBe('2025-12-20');
});
