import { test, expect } from '@playwright/test';

async function setup(context, overrides = {}) {
 const state = { type: 0, version: 1, maintenance: null, writes: [], closed: false, conflict: false, ...overrides };
 await context.addInitScript(() => sessionStorage.setItem('ugo-evid-session', JSON.stringify({ access_token:'access',refresh_token:'refresh',access_token_expires_at:new Date(Date.now()+3600000).toISOString(),refresh_token_expires_at:new Date(Date.now()+7200000).toISOString(),user:{username:'test.user'} })));
 await context.route('**/api/**', async route => {
  const req = route.request(), path = new URL(req.url()).pathname;
  const json = (body,status=200) => route.fulfill({status,contentType:'application/json',body:JSON.stringify(body)});
  if(path.endsWith('/detalji')) return json({id_ugo_evid:9,id_ugo_org:3,br_ugovor:'460',predmet_ugovora:'Odrzavanje',otvoren_ug:state.closed?'':'X',lica_dobavljaca:[]});
  if(path.endsWith('/ugo_tip_ugovora')) return json({items:[{id:1,sifra:'ODRZAVANJE',naziv:'Odrzavanje softvera'},{id:2,sifra:'LICENCE',naziv:'Licence'}]});
  if(path.endsWith('/dokumenti')) return json({items:[],types:[]});
  if(path.endsWith('/podesavanja')) return json({settings:{id_tip_ugovora:state.type,version:String(state.version),odrzavanje:state.maintenance},document_types:[{id:5,sifra:'POTVRDA_TEH',naziv:'Potvrda tehnicke podrske'},{id:6,sifra:'ZAPISNIK_TEH',naziv:'Zapisnik tehnicke podrske'}]});
  const body=req.postDataJSON();
  if(path.endsWith('/tip')) {if(body.version!==String(state.version))return json({error:'Podaci su promenjeni.'},409);state.type=body.id_tip_ugovora;state.version++;state.writes.push(body);return json({status:'ok'});}
  if(path.endsWith('/odrzavanje')) {if(state.maintenance || state.conflict)return json({error:'Odrzavanje vec postoji.'},409);state.maintenance={...body,id:1};state.writes.push(body);return json({status:'ok'},201);}
  return json({error:'Unknown route'},404);
 });
 return state;
}
async function open(page){await page.goto('/ugovori/detalji/9');await expect(page.getByRole('region',{name:'Tip ugovora i održavanje'})).toBeVisible();}
async function fill(page){
 await page.getByRole('button',{name:'Unesi održavanje'}).click();
 await page.getByLabel('Datum potpisa ugovora',{exact:true}).fill('2026-01-01');
 await page.getByLabel('Datum kraja održavanja',{exact:true}).fill('2026-12-31');
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
 await page.getByLabel('Period tehničke podrške (meseci)',{exact:true}).fill('3');
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
 await page.getByLabel('Period tehničke podrške (meseci)',{exact:true}).fill('');
 await page.getByRole('button',{name:'Sačuvaj podešavanja'}).click();
 await expect(page.getByRole('dialog')).toHaveCount(0);
 expect(state.maintenance.teh_period_meseci).toBeNull();expect(state.maintenance.ima_razvojnu_podrsku).toBe(false);
});
test('concurrent creation conflict preserves input and blocks retry',async({context,page})=>{
 const state=await setup(context,{type:1,conflict:true});await open(page);await fill(page);
 await page.getByRole('button',{name:'Sačuvaj podešavanja'}).click();
 await expect(page.getByRole('alert')).toContainText('vec postoji');
 await expect(page.getByRole('button',{name:'Sačuvaj podešavanja'})).toBeDisabled();
 await expect(page.getByLabel('Datum potpisa ugovora',{exact:true})).toHaveValue('2026-01-01');
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
