import {test,expect} from '@playwright/test';
async function setup(context){
 const state={items:[],version:1,writes:[]};
 await context.addInitScript(()=>sessionStorage.setItem('ugo-evid-session',JSON.stringify({access_token:'access',refresh_token:'refresh',access_token_expires_at:new Date(Date.now()+3600000).toISOString(),refresh_token_expires_at:new Date(Date.now()+7200000).toISOString(),user:{username:'test.user'}})));
 await context.route('**/api/**',async route=>{
  const req=route.request();if(!req.url().includes('/radni_kalendar/'))return route.fulfill({status:404,json:{error:'Unknown route'}});
  if(req.method()==='GET')return route.fulfill({json:{items:state.items,version:String(state.version)}});
  const body=req.postDataJSON();if(body.version!==String(state.version))return route.fulfill({status:409,json:{error:'Kalendar je izmenjen u drugoj sesiji.'}});
  state.writes.push(body);
  if(body.action==='generate'){for(let i=0;i<365;i++){const d=new Date(Date.UTC(2026,0,1+i)),datum=d.toISOString().slice(0,10);if(!state.items.some(x=>x.datum===datum))state.items.push({datum,radni:[0,6].includes(d.getUTCDay())?'N':'D',potvrdjen:'N',opis:''});}}
  if(body.action==='change')for(const d of state.items){if(d.datum>=body.od&&d.datum<=body.do)Object.assign(d,{radni:body.radni,opis:body.opis,potvrdjen:'N'});}
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
 await page.getByRole('button',{name:'Izmeni 2026-01-03',exact:true}).click();await page.getByLabel('Radni dan',{exact:true}).selectOption('D');await page.getByRole('button',{name:'Sačuvaj dane'}).click();
 await expect(page.getByRole('status').filter({hasText:'Generisano'})).toContainText('Potvrđeno 364');expect(state.items[2].radni).toBe('D');
});
test('stale approval cannot confirm changes from another session',async({page,context})=>{
 const state=await setup(context);await page.goto('/ugovori/radni-kalendar?godina=2026');await page.getByRole('button',{name:'Generiši kalendar'}).click();
 await page.getByRole('button',{name:'Potvrdi godinu'}).click();state.version++;
 await page.getByRole('button',{name:'Potvrdi',exact:true}).click();await expect(page.getByRole('alert')).toContainText('drugoj sesiji');expect(state.items.every(d=>d.potvrdjen==='N')).toBe(true);
});
