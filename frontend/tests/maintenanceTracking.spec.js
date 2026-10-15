import { test, expect } from '@playwright/test';

async function setup(context,extra={}){
 const state={token:null,periodi:[],zahtevi:[],writes:[],error:null,closed:false,...extra};
 const maintenance={id:1,version:'1',pocetak_odrzavanja:'',ima_razvojnu_podrsku:true,teh_period_meseci:1,id_teh_tip_doc:5,teh_podsetnik_dana:7,raz_podsetnik_dana:3,fondovi:[{id_vrsta_rada:1,ugovoreno_casova:100},{id_vrsta_rada:3,ugovoreno_casova:0}]};
 await context.addInitScript(()=>sessionStorage.setItem('ugo-evid-session',JSON.stringify({access_token:'access',refresh_token:'refresh',access_token_expires_at:new Date(Date.now()+3600000).toISOString(),refresh_token_expires_at:new Date(Date.now()+7200000).toISOString(),user:{username:'test.user'}})));
 await context.route('**/api/**',async route=>{
  const req=route.request(),path=new URL(req.url()).pathname,body=req.postData()?req.postDataJSON():{};
  const json=(data,status=200)=>route.fulfill({status,json:data});
  if(path.endsWith('/detalji'))return json({id_ugo_evid:9,id_ugo_org:3,br_ugovor:'460',predmet_ugovora:'Održavanje',otvoren_ug:state.closed?'':'X',lica_dobavljaca:[]});
  if(path.endsWith('/ugo_tip_ugovora'))return json({items:[{id:1,sifra:'ODRZAVANJE',naziv:'Održavanje softvera'}]});
  if(path.endsWith('/dokumenti'))return json({items:[{id:20,naziv:'Potvrda januar',id_tip_doc:5}],types:[{id:5,sifra:'POTVRDA_TEH',naziv:'Potvrda'}]});
  if(path.endsWith('/podesavanja'))return json({settings:{id_tip_ugovora:1,version:'1',odrzavanje:maintenance,sap_pocetak:'2026-01-15',sap_kraj:'2026-02-28',vrste_rada:[{id:1,naziv:'Inženjer/programer'},{id:3,naziv:'Konsultant'}]},document_types:[{id:5,sifra:'POTVRDA_TEH',naziv:'Potvrda'}]});
  if(path.endsWith('/podsetnici'))return json({items:[{id_ugo_evid:9,vrsta:'TEH',id:1,opis:'Januar',rok:'2026-01-31',kasni:false}]});
  if(path.endsWith('/pracenje'))return json({periodi:state.periodi,zahtevi:state.zahtevi,fondovi:[{id_vrsta_rada:1,naziv:'Inženjer/programer',ugovoreno:100,zauzeto:20,preostalo:80}]});
  if(path.endsWith('/lock')){
   if(req.method()==='POST'){if(state.token)return json({error:'Održavanje menja drugi korisnik ili tab.'},423);state.token='a'.repeat(64);}
   if(req.method()==='DELETE')state.token=null;
   if(req.method()==='PUT'&&body.lock_token!==state.token)return json({error:'Pravo izmene je isteklo.'},409);
   return json({lock_token:state.token});
  }
  if(path.endsWith('/preview'))return json({items:[{period_od:'2026-01-15',period_do:'2026-01-31',status:'PLANIRAN',id_tip_doc:5},{period_od:'2026-02-01',period_do:'2026-02-28',status:'PLANIRAN',id_tip_doc:5}]});
  if(path.includes('/pracenje/')){
   if(state.error)return json({error:state.error},400);
   if(body.lock_token!==state.token)return json({error:'Zaključavanje nije važeće.'},409);
   const kind=path.split('/').at(-1);state.writes.push(body);
   if(req.method()==='DELETE'){const p=state.periodi.find(p=>p.id===body.items[0].id);if(!p||p.status!=='PLANIRAN'||p.version!==body.items[0].version)return json({error:'Period nije vise planiran ili je promenjen.'},409);state.periodi=state.periodi.filter(x=>x.id!==p.id);state.token=null;return json({status:'ok'});}
   for(const p of body.items){const current=state[kind].find(x=>x.id===p.id);if(current)Object.assign(current,p,{version:String(Number(p.version)+1)});else state[kind].push({...p,id:state[kind].length+1,version:'1',rok:p.rok||'2026-02-10'});}
   state.token=null;return json({status:'ok'});
  }
  return json({error:'Unknown route'},404);
 });return state;
}
async function open(page){await page.goto('/ugovori/detalji/9');await expect(page.getByRole('heading',{name:'Praćenje održavanja'})).toBeVisible();}

test('generation previews before saving and displays reminders and contracted work only',async({page,context})=>{
 const state=await setup(context);await open(page);
 await expect(page.locator('.maintenance-summary')).not.toContainText('Konsultant');
 await expect(page.getByRole('region',{name:'Podsetnici za održavanje'})).toContainText('Januar');
 await page.getByRole('button',{name:'Generiši periode'}).click();
 await expect(page.getByRole('button',{name:'Sačuvaj',exact:true})).toBeDisabled();
 await page.getByRole('button',{name:'Prikaži predlog'}).click();
 await expect(page.locator('.tracking-preview')).toContainText('2026-01-15 — 2026-01-31');expect(state.writes).toHaveLength(0);
 await page.getByRole('button',{name:'Sačuvaj',exact:true}).click();
 await expect(page.getByRole('dialog')).toHaveCount(0);expect(state.periodi).toHaveLength(2);
 await expect(page.getByRole('button',{name:'Generiši periode'})).toBeDisabled();
 await page.screenshot({path:'test-results/maintenance-tracking.png'});
});
test('manual period form locks other tab and cancellation keeps draft until confirmed',async({page,context})=>{
 const state=await setup(context);const other=await context.newPage();await open(page);await open(other);
 await page.getByRole('button',{name:'Dodaj period'}).click();await page.getByLabel('Period od',{exact:true}).fill('2026-01-15');
 await other.getByRole('button',{name:'Dodaj razvojni zahtev'}).click();await expect(other.getByRole('dialog')).toHaveCount(0);await expect(other.getByRole('alert')).toContainText('drugi');
 await page.getByRole('button',{name:'Odustani',exact:true}).click();await page.getByRole('button',{name:'Otkaži',exact:true}).click();
 await expect(page.getByLabel('Period od',{exact:true})).toHaveValue('2026-01-15');
 await page.getByLabel('Period do',{exact:true}).fill('2026-01-31');await page.getByLabel('Potvrda / zapisnik').selectOption('20');
 await page.getByRole('button',{name:'Sačuvaj',exact:true}).click();await expect(page.getByRole('dialog')).toHaveCount(0);expect(state.writes[0].items[0].id_ugo_doc).toBe(20);
});
test('development request selects one funded work type and preserves form after rejected hours',async({page,context})=>{
 const state=await setup(context,{error:'Zahtev prekoračuje preostale časove.'});await open(page);
 await page.getByRole('button',{name:'Dodaj razvojni zahtev'}).click();await page.getByLabel('Broj zahteva',{exact:true}).fill('R-1');
 await page.getByLabel('Datum zahteva',{exact:true}).fill('2026-01-20');await page.getByLabel('Vrsta rada',{exact:true}).selectOption('1');
 await expect(page.getByLabel('Vrsta rada',{exact:true}).locator('option')).toHaveCount(2);
 await page.getByLabel('Broj časova',{exact:true}).fill('90');await page.getByLabel('Rok u radnim danima').fill('15');await page.getByLabel('Status',{exact:true}).selectOption('PODNET');
 await page.getByRole('button',{name:'Sačuvaj',exact:true}).click();await expect(page.getByRole('alert')).toContainText('prekoračuje');
 await expect(page.getByLabel('Broj časova',{exact:true})).toHaveValue('90');state.error=null;
 await page.getByLabel('Broj časova',{exact:true}).fill('20');await page.getByRole('button',{name:'Sačuvaj',exact:true}).click();await expect(page.getByRole('dialog')).toHaveCount(0);
 expect(state.writes[0].items[0]).toMatchObject({id_vrsta_rada:1,broj_casova:20,rok_radnih_dana:15,status:'PODNET'});
});
test('closed maintenance is read only and home shows reminders',async({page,context})=>{
 await setup(context,{closed:true});await open(page);await expect(page.getByRole('button',{name:'Dodaj period'})).toHaveCount(0);
 await page.goto('/');await expect(page.getByRole('region',{name:'Podsetnici za održavanje'})).toContainText('Januar');
});

 test('period menu deletes only planned periods, confirms and locks other tabs',async({page,context})=>{
 const state=await setup(context,{periodi:[{id:1,version:'1',period_od:'2026-01-15',period_do:'2026-01-31',status:'PLANIRAN',id_tip_doc:5},{id:2,version:'1',period_od:'2026-02-01',period_do:'2026-02-28',status:'PRIPREMA',id_tip_doc:5}]});
 await open(page);const other=await context.newPage();await open(other);
 await page.getByRole('button',{name:'Akcije za period 2026-02-01 - 2026-02-28'}).click();
 await expect(page.getByRole('menuitem',{name:'Obriši period'})).toHaveCount(0);
 await page.keyboard.press('Escape');
 await page.getByRole('button',{name:'Akcije za period 2026-01-15 - 2026-01-31'}).click();
 await page.getByRole('menuitem',{name:'Obriši period'}).click();
 await expect(page.getByRole('dialog')).toBeVisible();
 await other.getByRole('button',{name:'Akcije za period 2026-01-15 - 2026-01-31'}).click();
 await other.getByRole('menuitem',{name:'Izmeni period'}).click();await expect(other.getByRole('alert')).toContainText('drugi');await expect(other.getByRole('dialog')).toHaveCount(0);
 await page.getByRole('button',{name:'Odustani',exact:true}).click();await expect(page.getByRole('dialog')).toHaveCount(0);expect(state.periodi).toHaveLength(2);expect(state.token).toBeNull();
 await page.getByRole('button',{name:'Akcije za period 2026-01-15 - 2026-01-31'}).click();await page.getByRole('menuitem',{name:'Obriši period'}).click();
 await page.getByRole('button',{name:'Obriši period',exact:true}).click();await expect(page.getByRole('dialog')).toHaveCount(0);
 await expect(page.getByRole('button',{name:'Akcije za period 2026-01-15 - 2026-01-31'})).toHaveCount(0);
 expect(state.periodi).toHaveLength(1);expect(state.token).toBeNull();
 });
