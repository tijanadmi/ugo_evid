import TechnicalPeriodsTable from "./TechnicalPeriodsTable";
import DevelopmentRequestsTable from "./DevelopmentRequestsTable";
import MaintenanceTrackingEditor from "./MaintenanceTrackingEditor";
import { emptyTrackingItem as empty } from "./trackingConstants";
import Documents from "../documents/Documents";
import { formatDate } from "../../utils/contractFormatting";
import { useRef, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import toast from "react-hot-toast";
import { useAuth } from "../../context/useAuth";
import { useContactSession } from "../partners/useContactSession";
import { maintenanceService } from "./maintenanceAPI";
import { announceBusinessChange } from "../../services/DataSync";
import DocumentDialog from "../documents/DocumentDialog";
import Reminders from "./Reminders";
import "./maintenanceTracking.css";

export default function MaintenanceTracking({ item, settings, documentTypes }) {
  const { api, user } = useAuth();
  const client = useQueryClient();
  const owner = useContactSession(api, maintenanceService);
  const [editor, setEditor] = useState(null),
    [busy, setBusy] = useState(false),
    [confirm, setConfirm] = useState(false),
    [blocked, setBlocked] = useState(false);
  const [requestDocuments, setRequestDocuments] = useState(null);
  const [periodDocuments, setPeriodDocuments] = useState(null);
  const pending = useRef(false),
    id = item.id_ugo_evid;
  const query = useQuery({
    queryKey: ["maintenance-tracking", user.username, id],
    queryFn: ({ signal }) => api(`/ugo_evid/${id}/pracenje`, { signal }),
    retry: false,
    refetchInterval: 30000,
    refetchOnWindowFocus: true,
  });
  const data = query.data,
    canWrite = item.otvoren_ug === "X" && !!data && !query.error;
  async function open(
    kind,
    row,
    generate = false,
    remove = false,
    completed = false,
  ) {
    if (pending.current || editor) return;
    pending.current = true;
    setBusy(true);
    setBlocked(false);
    try {
      const session = await owner.acquire(id, "edit");
      if (!session) return;
      const fresh = await api(`/ugo_evid/${id}/pracenje`);
      client.setQueryData(["maintenance-tracking", user.username, id], fresh);
      const source = row ? fresh[kind].find((x) => x.id === row.id) : null;
      if (row && !source) throw new Error("Slog više nije dostupan.");
      if (remove && kind === "periodi" && source?.status !== "PLANIRAN")
        throw new Error(
          "Status više ne dozvoljava brisanje. Osvežite podatke.",
        );
      if (remove && kind === "zahtevi" && source?.status === "ZAVRSEN")
        throw new Error("Završen zahtev nije moguće obrisati.");
      const current = session.person.settings;
      setEditor({
        kind,
        generate,
        remove,
        mode: [1, 3, 6, 12].includes(current.odrzavanje.teh_period_meseci)
          ? "calendar"
          : "anchor",
        preview: null,
        settings: current,
        row: source || {
          ...empty,
          status:
            kind === "periodi" ? "PLANIRAN" : completed ? "ZAVRSEN" : "NACRT",
          id_tip_doc: current.odrzavanje.id_teh_tip_doc,
        },
      });
    } catch (e) {
      toast.error(e.message);
      await owner.release();
    } finally {
      pending.current = false;
      setBusy(false);
    }
  }
  function change(key, value) {
    setEditor((s) => ({ ...s, row: { ...s.row, [key]: value } }));
  }
  async function discard() {
    setBusy(true);
    await owner.release();
    setEditor(null);
    setConfirm(false);
    setBusy(false);
  }
  async function preview() {
    if (pending.current) return;
    pending.current = true;
    setBusy(true);
    try {
      if (!(await owner.renew())) return;
      const result = await api(`/ugo_evid/${id}/pracenje/preview`, {
        method: "POST",
        body: { mode: editor.mode },
      });
      setEditor((s) => ({ ...s, preview: result.items }));
    } catch (e) {
      toast.error(e.message);
    } finally {
      pending.current = false;
      setBusy(false);
    }
  }
  async function save(event) {
    event?.preventDefault();
    const uploadAfterSave = event?.nativeEvent?.submitter?.value === "upload";
    if (pending.current || owner.lost || blocked) return;
    pending.current = true;
    setBusy(true);
    try {
      if (!(await owner.renew())) return;
      const response = await api(`/ugo_evid/${id}/pracenje/${editor.kind}`, {
        method: editor.remove ? "DELETE" : "POST",
        body: {
          lock_token: owner.session.token,
          items: editor.generate ? editor.preview : [editor.row],
        },
      });
      owner.committed();
      setEditor(null);
      toast.success("Podaci su sačuvani.");
      announceBusinessChange(client);
      if (uploadAfterSave && !editor.generate && !editor.remove) {
        const savedID = response.id || editor.row.id;
        if (!savedID) {
          toast.error(
            "Podaci su sačuvani, ali dokument dodajte iz menija reda nakon osvežavanja liste.",
          );
          return;
        }
        const documents = { row: { ...editor.row, id: savedID }, upload: true };
        if (editor.kind === "periodi") setPeriodDocuments(documents);
        else setRequestDocuments(documents);
      }
    } catch (e) {
      toast.error(e.message);
      if ([0, 404, 409, 423].includes(e.status) || e.status >= 500)
        setBlocked(true);
    } finally {
      pending.current = false;
      setBusy(false);
    }
  }
  const periods = data?.periodi || [];
  const active = periods
    .filter((p) => p.status !== "OTKAZAN")
    .sort((a, b) => a.period_od.localeCompare(b.period_od));
  const start = settings.odrzavanje.pocetak_odrzavanja || settings.sap_pocetak;
  const hasGap =
    active.length > 0 &&
    (active[0].period_od > start ||
      active.at(-1).period_do < settings.sap_kraj ||
      active.some(
        (p, i) =>
          i > 0 &&
          new Date(p.period_od + "T00:00:00Z") -
            new Date(active[i - 1].period_do + "T00:00:00Z") >
            86400000,
      ));
  return (
    <section className="maintenance-tracking" aria-label="Praćenje održavanja">
      <h2>Praćenje održavanja</h2>
      {query.isPending ? (
        <p>Učitavanje perioda i zahteva...</p>
      ) : query.error ? (
        <p role="alert">
          Praćenje nije dostupno. Proverite da li je izvršen
          maintenance_tracking.sql.{" "}
          <button className="button" onClick={() => query.refetch()}>
            Pokušaj ponovo
          </button>
        </p>
      ) : (
        <>
          <h3>Tehnički periodi</h3>
          {canWrite && (
            <div className="tracking-actions">
              <button
                className="button"
                disabled={busy}
                onClick={() => open("periodi")}
              >
                Dodaj period
              </button>
              <button
                className="button"
                disabled={
                  busy ||
                  periods.length > 0 ||
                  !settings.odrzavanje.teh_period_meseci
                }
                onClick={() => open("periodi", null, true)}
              >
                Generiši periode
              </button>
            </div>
          )}
          <p className="muted">
            Generisanje je dostupno dok lista nema perioda. Postojeći periodi se
            menjaju pojedinačno. Otkazani periodi ostaju u evidenciji.
          </p>
          {hasGap && (
            <p role="status">
              Postoje nepokriveni delovi trajanja održavanja. Proverite datume
              perioda.
            </p>
          )}
          <TechnicalPeriodsTable
            periods={periods}
            canWrite={canWrite}
            busy={busy}
            open={open}
            onDocuments={setPeriodDocuments}
          />
          {!periods.length && <p>Nema evidentiranih tehničkih perioda.</p>}
          {settings.odrzavanje.ima_razvojnu_podrsku && (
            <>
              <h3>Razvojna podrška</h3>
              <div className="tracking-table">
                <table>
                  <thead>
                    <tr>
                      <th>Vrsta rada</th>
                      <th>Ugovoreno</th>
                      <th>Zauzeto / utrošeno</th>
                      <th>Preostalo</th>
                    </tr>
                  </thead>
                  <tbody>
                    {data.fondovi.map((f) => (
                      <tr key={f.id_vrsta_rada}>
                        <td>{f.naziv}</td>
                        <td>{f.ugovoreno}</td>
                        <td>{f.zauzeto}</td>
                        <td>
                          <strong>{f.preostalo}</strong>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
              <p className="muted">
                Nacrt i otkazan zahtev ne zauzimaju časove. Završeni zahtevi
                ostaju u obračunu.
              </p>
              {canWrite && (
                <div className="tracking-actions">
                  <button
                    className="button"
                    disabled={busy || !data.fondovi.length}
                    onClick={() => open("zahtevi")}
                  >
                    Dodaj razvojni zahtev
                  </button>
                  <button
                    className="button"
                    disabled={busy || !data.fondovi.length}
                    onClick={() => open("zahtevi", null, false, false, true)}
                  >
                    Unesi završen zahtev
                  </button>
                </div>
              )}
              <DevelopmentRequestsTable
                requests={data.zahtevi}
                workTypes={settings.vrste_rada}
                canWrite={canWrite}
                busy={busy}
                open={open}
                onDocuments={setRequestDocuments}
              />
              {!data.zahtevi.length && (
                <p>Nema evidentiranih razvojnih zahteva.</p>
              )}
            </>
          )}
        </>
      )}
      <Reminders contractID={id} />
      {periodDocuments && (
        <DocumentDialog
          wide
          title={`Dokumenta perioda ${formatDate(periodDocuments.row.period_od)} — ${formatDate(periodDocuments.row.period_do)}`}
          onClose={() => setPeriodDocuments(null)}
        >
          <Documents
            item={item}
            period={periodDocuments.row}
            startUpload={periodDocuments.upload}
          />
        </DocumentDialog>
      )}
      {requestDocuments && (
        <DocumentDialog
          wide
          title={`Dokumenta zahteva ${requestDocuments.row.broj_zahteva}`}
          onClose={() => setRequestDocuments(null)}
        >
          <Documents
            item={item}
            request={requestDocuments.row}
            startUpload={requestDocuments.upload}
          />
        </DocumentDialog>
      )}
      <MaintenanceTrackingEditor
        editor={editor}
        setEditor={setEditor}
        busy={busy}
        blocked={blocked}
        lost={owner.lost}
        save={save}
        discard={discard}
        confirm={confirm}
        setConfirm={setConfirm}
        preview={preview}
        change={change}
        documentTypes={documentTypes}
        funds={data?.fondovi || []}
      />
    </section>
  );
}
