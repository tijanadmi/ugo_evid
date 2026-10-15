import {test,expect} from '@playwright/test';
async function setup(context){
 const state={items:[],version:1,writes:[],holidayError:false,previews:0};
 await context.addInitScript(()=>sessionStorage.setItem('ugo-evid-session',JSON.stringify({access_token:'access',refresh_token:'refresh',access_token_expires_at:new Date(Date.now()+3600000).toISOString(),refresh_token_expires_at:new Date(Date.now()+7200000).toISOString(),user:{username:'test.user'}})));
 await context.route('**/api/**',async route=>{
  const req=route.request();if(!req.url().includes('/radni_kalendar/'))return route.fulfill({status:404,json:{error:'Unknown route'}});
  if(req.url().endsWith('/praznici')){state.previews++;return route.fulfill(state.holidayError?{status:502,json:{error:'Preuzimanje praznika nije uspelo.'}}:{json:{items:[{datum:'2026-01-01',naziv:'Nova godina',poreklo:'Nager.Date'},{datum:'2026-01-02',naziv:'Nova godina',poreklo:'Nager.Date'}]}});}
  if(req.method()==='GET')return route.fulfill({json:{items:state.items,version:String(state.version)}});
  const body=req.postDataJSON();if(body.version!==String(state.version))return route.fulfill({status:409,json:{error:'Kalendar je izmenjen u drugoj sesiji.'}});
  state.writes.push(body);
  if(body.action==='generate'){for(let i=0;i<365;i++){const d=new Date(Date.UTC(2026,0,1+i)),datum=d.toISOString().slice(0,10);if(!state.items.some(x=>x.datum===datum))state.items.push({datum,radni:[0,6].includes(d.getUTCDay())?'N':'D',potvrdjen:'N',opis:'',izvor:'GENERISAN'});}}
  if(body.action==='change')for(const d of state.items){if(d.datum>=body.od&&d.datum<=body.do)Object.assign(d,{radni:body.radni,opis:body.opis,potvrdjen:'N',izvor:'RUCNO'});}
  if(body.action==='import')for(const h of body.praznici){const day=state.items.find(d=>d.datum===h.datum);if(day.izvor!=='RUCNO'||h.zameni_rucno)Object.assign(day,{radni:'N',opis:h.naziv,potvrdjen:'N',izvor:'PRAZNIK'});}
  if(body.action==='approve')for(const d of state.items)d.potvrdjen='D';
  state.version++;return route.fulfill({json:{status:'ok'}});
 });return state;
}
test('generate year, add holiday range, preserve overrides, confirm and change working weekend',async({page,context})=>{
 const state=await setup(context);await page.goto('/ugovori/radni-kalendar?godina=2026');
 await expect(page.getByRole('button',{name:'Potvrdi godinu'})).toBeDisabled();await page.getByRole('button',{name:'Generiši kalendar'}).click();
 await expect(page.getByRole('status').filter({hasText:'Generisano'})).toContainText('365 od 365');
 await page.getByRole('button',{name:'Izmeni dane'}).click();await page.getByLabel('Od datuma').fill('2026-01-01');await page.getByLabel('Do datuma').fill('2026-01-02');await page.getByLabel('Opis',{exact:true}).fill('Praznik');
 await page.getByRole('button',{name:'Sačuvaj dane'}).click();await expect(page.getByRole('dialog')).toHaveCount(0);
 await page.getByRole('button',{name:'Generiši kalendar'}).click();await expect.poll(()=>state.writes.length).toBe(3);expect(state.items[0].opis).toBe('Praznik');
 await page.getByRole('button',{name:'Potvrdi godinu'}).click();await page.getByRole('button',{name:'Potvrdi',exact:true}).click();await expect(page.getByRole('dialog')).toHaveCount(0);
 await expect(page.getByRole('status').filter({hasText:'Generisano'})).toContainText('Potvrđeno 365');
 await page.getByRole('button',{name:'2026-01-03, neradni dan',exact:true}).click();
 await expect(page.getByRole('status').filter({hasText:'Generisano'})).toContainText('Potvrđeno 364');expect(state.items[2].radni).toBe('D');
});
test('stale approval cannot confirm changes from another session',async({page,context})=>{
 const state=await setup(context);await page.goto('/ugovori/radni-kalendar?godina=2026');await page.getByRole('button',{name:'Generiši kalendar'}).click();
 await page.getByRole('button',{name:'Potvrdi godinu'}).click();state.version++;
 await page.getByRole('button',{name:'Potvrdi',exact:true}).click();await expect(page.getByRole('alert')).toContainText('drugoj sesiji');expect(state.items.every(d=>d.potvrdjen==='N')).toBe(true);
});

test('monthly calendar fits viewport, switches months and saves a day with keyboard',async({page,context})=>{
 await setup(context);await page.setViewportSize({width:1440,height:900});await page.goto('/ugovori/radni-kalendar?godina=2026&mesec=1');
 await page.getByRole('button',{name:'Generiši kalendar'}).click();
 await expect(page.getByRole('button',{name:'2026-01-05, radni dan',exact:true})).toHaveClass(/working/);
 const day=page.getByRole('button',{name:'2026-01-05, radni dan',exact:true});await day.focus();await page.keyboard.press('Enter');
 await expect(page.getByRole('button',{name:'2026-01-05, neradni dan',exact:true})).toHaveClass(/nonworking/);
 await page.getByRole('button',{name:'Sledeći mesec'}).click();await expect(page.getByLabel('Mesec',{exact:true})).toHaveValue('2');
 await expect(page.getByRole('button',{name:/2026-01-05,/})).toHaveCount(0);
 await page.getByRole('button',{name:'Prethodni mesec'}).click();await expect(page.getByRole('button',{name:/2026-01-05, neradni/})).toBeVisible();
 const bounds=await page.locator('.work-calendar').boundingBox();expect(bounds.y+bounds.height).toBeLessThanOrEqual(900);
 await page.screenshot({path:'test-results/calendar-month-desktop.png'});
 await page.setViewportSize({width:390,height:844});
 const mobile=await page.locator('.work-calendar').boundingBox();expect(mobile.y+mobile.height).toBeLessThanOrEqual(844);
 expect(await page.evaluate(()=>document.documentElement.scrollWidth)).toBeLessThanOrEqual(390);
 await page.screenshot({path:'test-results/calendar-month-mobile.png'});
});

test('holiday preview is read only, preserves manual days and imports selected dates',async({page,context})=>{
 const state=await setup(context);await page.goto('/ugovori/radni-kalendar?godina=2026');await page.getByRole('button',{name:'Generiši kalendar'}).click();
 // A manual working-day override must survive future imports even without a description.
 await page.getByRole('button',{name:'2026-01-01, radni dan',exact:true}).click();await expect(page.getByRole('button',{name:'2026-01-01, neradni dan',exact:true})).toBeEnabled();
 await page.getByRole('button',{name:'2026-01-01, neradni dan',exact:true}).click();await expect(page.getByRole('button',{name:'2026-01-01, radni dan',exact:true})).toBeEnabled();
 const before=state.writes.length;
 await page.getByRole('button',{name:'Preuzmi praznike za Srbiju'}).click();await expect(page.getByRole('dialog')).toBeVisible();
 expect(state.writes).toHaveLength(before);expect(state.previews).toBe(1);
 const checks=page.getByRole('checkbox');await expect(checks.nth(0)).not.toBeChecked();await expect(checks.nth(1)).toBeChecked();
 await page.getByRole('button',{name:'Uvezi izabrane praznike'}).click();await expect(page.getByRole('dialog')).toHaveCount(0);
 expect(state.items[0]).toMatchObject({radni:'D',izvor:'RUCNO'});expect(state.items[1]).toMatchObject({radni:'N',opis:'Nova godina',izvor:'PRAZNIK',potvrdjen:'N'});
 await page.getByRole('button',{name:/2026-01-02, neradni dan/}).click();await expect(page.getByRole('button',{name:/2026-01-02, radni dan/})).toBeEnabled();
 expect(state.items[1].izvor).toBe('RUCNO');
});

test('failed download and stale click leave the calendar unchanged',async({page,context})=>{
 const state=await setup(context);await page.goto('/ugovori/radni-kalendar?godina=2026');await page.getByRole('button',{name:'Generiši kalendar'}).click();
 await expect(page.getByRole('button',{name:'Preuzmi praznike za Srbiju'})).toBeEnabled();state.holidayError=true;const before=state.writes.length;
 await page.getByRole('button',{name:'Preuzmi praznike za Srbiju'}).click();await expect(page.getByRole('alert')).toContainText('Preuzimanje');expect(state.writes).toHaveLength(before);await expect(page.getByRole('dialog')).toHaveCount(0);
 state.version++;await page.getByRole('button',{name:'2026-01-05, radni dan',exact:true}).click();
 await expect(page.getByRole('button',{name:'2026-01-05, radni dan',exact:true})).toBeEnabled();expect(state.items[4].radni).toBe('D');expect(state.writes).toHaveLength(before);
});
