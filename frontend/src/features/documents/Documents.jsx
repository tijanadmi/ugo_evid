import { value } from '../../utils/contractFormatting';
import { useEffect, useRef, useState } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import toast from 'react-hot-toast';
import { useAuth } from '../../context/useAuth';
import { announceBusinessChange } from '../../services/DataSync';
import { useContactSession } from '../partners/useContactSession';
import { documentSessionService, uploadDocument } from './documentAPI';
import DocumentDialog from './DocumentDialog';
import DocumentForm from './DocumentForm';
import ConfirmAction from '../../ui/ConfirmAction';
import RowMenu from '../../ui/RowMenu';
import Icon from '../../ui/Icon';

export default function Documents({ item, children }) {
 const { api, user } = useAuth();
 const client = useQueryClient();
 const owner = useContactSession(api, documentSessionService);
 const [editor, setEditor] = useState(null);
 const [busy, setBusy] = useState(false);
 const [blocked, setBlocked] = useState(false);
 const [confirmClose, setConfirmClose] = useState(false);
 const [preview, setPreview] = useState(null);
 const [imageZoom, setImageZoom] = useState(100);
 const pending = useRef(false);
 const mounted = useRef(true);
 useEffect(() => { mounted.current = true; return () => { mounted.current = false; }; }, []);
 useEffect(() => () => { if (preview) URL.revokeObjectURL(preview.url); }, [preview]);
 const query = useQuery({ queryKey: ['documents', user.username, item.id_ugo_evid],
   queryFn: async ({ signal }) => { try { return await api(`/ugo_evid/${item.id_ugo_evid}/dokumenti`, { signal }); } catch (error) { if (error.status === 403) return { forbidden: true }; throw error; } },
   retry: false, refetchOnWindowFocus: true, refetchInterval: 30000,
 });
 const data = query.data;
 const canWrite = item.otvoren_ug === 'X' && data && !data.forbidden && !query.error;
 function upload(code) {
   if (!canWrite || busy || pending.current || editor) return;
   const type = data.types?.find(type => type.sifra === code);
   if ((code && !type) || !data.types?.length) { toast.error(code ? `Tip dokumenta ${code} nije dostupan. Proverite aktivne tipove dokumenata u bazi.` : 'Nema aktivnih tipova dokumenata u bazi.'); return; }
   setBlocked(false); setEditor({ mode: 'add', fixedType: type, initial: { id_tip_doc: type?.id || data.types[0].id } });
 }
 async function open(doc, mode) {
   if (!canWrite || busy || pending.current || editor) return;
   pending.current = true; setBusy(true);
   try { const session = await owner.acquire(doc.id, mode); if (session && mounted.current) { setBlocked(false); setEditor({ mode, initial: session.person }); } else query.refetch(); }
   finally { pending.current = false; if (mounted.current) setBusy(false); }
 }
 async function cancel() {
   setBusy(true); await owner.release();
   if (mounted.current) { setEditor(null); setConfirmClose(false); setBlocked(false); setBusy(false); }
 }
 function close() {
   if (busy || pending.current) return;
   if (editor?.mode === 'delete') cancel(); else setConfirmClose(true);
 }
 function saved(message) {
   owner.committed(); setEditor(null); setBlocked(false); toast.success(message); announceBusinessChange(client);
 }
 async function save(metadata, file) {
   if (pending.current || busy || blocked || (editor.mode !== 'add' && owner.lost)) return;
   pending.current = true; setBusy(true);
   try {
     if (editor.mode === 'add') await uploadDocument(api, item.id_ugo_evid, metadata, file);
     else {
       if (!await owner.renew()) return;
       await api(`/ugo_doc/${editor.initial.id}`, { method: editor.mode === 'delete' ? 'DELETE' : 'PUT', body: { ...(metadata || {}), version: owner.session.person.version, lock_token: owner.session.token } });
     }
     saved(editor.mode === 'delete' ? 'Dokument je uklonjen iz prikaza.' : 'Dokument je sačuvan.');
   } catch (error) { toast.error(error.message, { id: 'operation-error' }); if ([0, 404, 409, 423].includes(error.status) || error.status >= 500) setBlocked(true); }
   finally { pending.current = false; if (mounted.current) setBusy(false); }
 }
 async function content(doc, showPreview) {
   if (pending.current || busy) return;
   pending.current = true; setBusy(true);
   try {
     const blob = await api(`/ugo_doc/${doc.id}/sadrzaj${showPreview ? '?preview=1' : ''}`, { responseType: 'blob' });
     if (!mounted.current) return;
     if (showPreview && !['application/pdf', 'image/jpeg', 'image/png'].includes(blob.type.split(';')[0])) throw new Error('Format nije podržan za pregled. Preuzmite dokument.');
     const url = URL.createObjectURL(blob);
     if (showPreview) { setImageZoom(100); setPreview({ url, doc }); }
     else { const a = document.createElement('a'); a.href = url; a.download = doc.originalni_naziv; document.body.append(a); a.click(); a.remove(); setTimeout(() => URL.revokeObjectURL(url), 1000); }
   } catch (error) { toast.error(error.message, { id: 'operation-error' }); }
   finally { pending.current = false; if (mounted.current) setBusy(false); }
 }
 return <>
   {children(upload, !!canWrite && !busy)}
   <section className="contract-card documents-panel" aria-label="Dokumenta ugovora">
     <div className="partner-heading"><h2>Dokumenta ugovora</h2><div className="document-buttons">
       {canWrite && <><button className="button" disabled={busy} onClick={() => upload('DODATNO_RESENJE')}>Dodaj dodatno rešenje</button><button className="button" disabled={busy} onClick={() => upload('ZAPISNIK')}>Dodaj zapisnik</button><button className="button primary" disabled={busy} onClick={() => upload()}>Dodaj dokument</button></>}
       <button className="button" disabled={query.isFetching} onClick={() => query.refetch()}>Osveži dokumenta</button>
     </div></div>
     {query.isPending ? <p role="status">Učitavanje dokumenata…</p> : data?.forbidden ? <p>Dokumenta su dostupna korisnicima organizacione jedinice ugovora.</p> : query.error ? <p>Dokumenta trenutno nisu dostupna.</p> : !data?.items?.length ? <p>Nema dodatih dokumenata.</p> :
       <ul className="document-list">{data.items.map(doc => <li key={doc.id}>
         <div><strong>{doc.naziv}</strong><span>{doc.tip_naziv}{doc.broj_dokumenta && ` · ${doc.broj_dokumenta}`}{doc.datpri && ` · Datum unosa: ${value(doc, 'datpri')}`}</span>
           <small>{doc.originalni_naziv} · {Math.ceil(doc.velicina_bajt / 1024)} KB</small>{doc.napomena && <p>{doc.napomena}</p>}</div>
         <div className="document-buttons">
           {['application/pdf', 'image/jpeg', 'image/png'].includes(doc.mime_tip) && <button className="button" disabled={busy} onClick={() => content(doc, true)}>Pregled</button>}
           <button className="button" disabled={busy} onClick={() => content(doc, false)}>Preuzmi</button>
           {canWrite && <RowMenu label={`Akcije za dokument ${doc.naziv}`} actions={[{ label: 'Izmeni', icon: 'edit', onClick: () => open(doc, 'edit') }, { label: 'Obriši', icon: 'trash', danger: true, onClick: () => open(doc, 'delete') }]}/>}
         </div>
       </li>)}</ul>}
   </section>
   {editor && <DocumentDialog title={editor.mode === 'add' ? 'Dodaj dokument' : editor.mode === 'edit' ? 'Izmeni dokument' : 'Obriši dokument'} busy={busy} onClose={close}>
     {editor.mode === 'delete' ? <ConfirmAction title={`Obriši ${editor.initial.naziv}?`} onConfirm={() => save()} onCancel={cancel} disabled={busy} blocked={blocked || owner.lost} variation="danger" confirmLabel="Potvrdi brisanje" cancelLabel="Odustani">
       <p>Dokument će biti uklonjen iz aktivnog prikaza. Fajl i evidencija o unosu ostaju sačuvani.</p>
     </ConfirmAction> : <DocumentForm fixedType={editor.fixedType} types={data?.types || []} initial={editor.initial} upload={editor.mode === 'add'} maxBytes={data?.max_bytes || 25 * 1024 * 1024}
       busy={busy} blocked={blocked || (editor.mode !== 'add' && owner.lost)} onSave={save} onCancel={close}/>}
     {confirmClose && <ConfirmAction modal title="Odustati od unosa dokumenta?" disabled={busy} onConfirm={cancel} onCancel={() => setConfirmClose(false)}><p>Nesačuvane izmene biće izgubljene.</p></ConfirmAction>}
   </DocumentDialog>}
   {preview && <DocumentDialog wide title={preview.doc.naziv} onClose={() => setPreview(null)}>
     <div className="detail-footer document-preview-toolbar">
       {preview.doc.mime_tip.startsWith('image/') && <div className="document-window-actions" role="group" aria-label="Zum slike">
         <button className="button" disabled={imageZoom <= 50} onClick={() => setImageZoom(zoom => Math.max(50, zoom - 25))} aria-label="Umanji sliku">?</button>
         <output aria-live="polite">{imageZoom}%</output>
         <button className="button" disabled={imageZoom >= 300} onClick={() => setImageZoom(zoom => Math.min(300, zoom + 25))} aria-label="Uvećaj sliku">+</button>
         <button className="button" onClick={() => setImageZoom(100)}>Prilagodi širini</button>
       </div>}
       <button className="button" disabled={busy} onClick={() => content(preview.doc, false)}>Preuzmi dokument</button></div>
     {preview.doc.mime_tip.startsWith('image/') ? <div className="document-image-viewport"><img className="document-image" style={{ width: `${imageZoom}%` }} src={preview.url} alt={preview.doc.naziv}/></div> : <iframe className="document-frame" title={preview.doc.naziv} src={preview.url} referrerPolicy="no-referrer"/>}
   </DocumentDialog>}
 </>;
}
