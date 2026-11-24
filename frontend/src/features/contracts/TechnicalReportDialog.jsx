import { useRef, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import toast from "react-hot-toast";
import { useAuth } from "../../context/useAuth";
import { formatDate } from "../../utils/contractFormatting";
import DocumentDialog from "../documents/DocumentDialog";
import { announceBusinessChange } from "../../services/DataSync";
import "./technicalReports.css";

export default function TechnicalReportDialog({
  contractID,
  period,
  periods,
  types,
  canWrite,
  onClose,
}) {
  const { api, user } = useAuth();
  const client = useQueryClient();
  const base = `/ugo_evid/${contractID}`;
  const templates = useQuery({
    queryKey: ["report-templates", user.username, contractID],
    queryFn: ({ signal }) => api(`${base}/sabloni`, { signal }),
    retry: false,
  });
  const tracking = useQuery({
    queryKey: ["report-periods", user.username, contractID],
    queryFn: ({ signal }) => api(`${base}/pracenje`, { signal }),
    enabled: !!period,
    retry: false,
  });
  const reports = useQuery({
    queryKey: ["technical-reports", user.username, contractID, period?.id],
    queryFn: ({ signal }) =>
      api(`${base}/teh-period/${period.id}/izvestaji`, { signal }),
    enabled: !!period,
    retry: false,
  });
  const row = tracking.data?.periodi?.find((p) => p.id === period?.id);
  const currentPeriods = tracking.data?.periodi || periods || [];
  const suggestion =
    row?.broj_kvartala ||
    1 +
      currentPeriods.filter(
        (p) =>
          p.id !== row?.id &&
          p.status === "ZAVRSEN" &&
          p.period_do < row?.period_od,
      ).length;
  const [quarter, setQuarter] = useState("");
  const [type, setType] = useState(period?.id_tip_doc || types[0]?.id || "");
  const [file, setFile] = useState(null);
  const [pdf, setPDF] = useState(true);
  const [busy, setBusy] = useState(false);
  const [blocked, setBlocked] = useState(false);
  const pending = useRef(false);
  const activeType = period ? row?.id_tip_doc : Number(type);
  const template = templates.data?.items?.find(
    (t) => t.id_tip_doc === activeType,
  );
  async function download(url, name) {
    try {
      const blob = await api(url, { responseType: "blob" });
      const href = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = href;
      a.download = name;
      a.click();
      setTimeout(() => URL.revokeObjectURL(href), 10000);
    } catch (e) {
      toast.error(e.message);
    }
  }
  async function submit(e) {
    e.preventDefault();
    if (pending.current || blocked) return;
    pending.current = true;
    setBusy(true);
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), 90000);
    try {
      if (!period) {
        if (!file) throw new Error("Izaberite DOCX šablon.");
        const body = new FormData();
        body.append("file", file);
        body.append("id_tip_doc", type);
        body.append("verzija", template?.verzija || 0);
        await api(`${base}/sabloni`, {
          method: "POST",
          body,
          signal: controller.signal,
        });
        toast.success("Nova verzija šablona je sačuvana.");
        setFile(null);
        e.target.reset();
        await templates.refetch();
      } else {
        await api(`${base}/teh-period/${period.id}/izvestaji`, {
          method: "POST",
          signal: controller.signal,
          body: {
            id_template: template.id,
            ...(template.koristi_kvartal ? {broj_kvartala: Number(quarter || suggestion)} : {}),
            version: row.version,
            pdf: pdf && templates.data.pdf_available,
          },
        });
        toast.success("Izveštaj je generisan. Status perioda nije promenjen.");
        await Promise.all([tracking.refetch(), reports.refetch()]);
        client.invalidateQueries({ queryKey: ["maintenance-tracking"] });
        announceBusinessChange();
      }
    } catch (e) {
      if (controller.signal.aborted) {
        toast.error(
          "Server nije potvrdio obradu. Osvežite podatke pre ponavljanja.",
        );
        setBlocked(true);
        return;
      }
      toast.error(e.message);
      if (
        e.status === 409 ||
        e.status === 423 ||
        e.status === 0 ||
        e.status >= 500
      )
        setBlocked(true);
    } finally {
      clearTimeout(timer);
      pending.current = false;
      setBusy(false);
    }
  }
  const error = templates.error || tracking.error || reports.error;
  return (
    <DocumentDialog
      title={
        period ? "Izveštaji tehničkog perioda" : "Šabloni tehničke podrške"
      }
      className="tracking-dialog"
      busy={busy}
      onClose={() => {
        if (!busy) onClose();
      }}
    >
      <form className="technical-report-form" onSubmit={submit}>
        <div className="technical-report-scroll">
          {error && <p role="alert">{error.message}</p>}
          {period ? (
            <>
              <p>
                Period:{" "}
                <strong>
                  {formatDate(row?.period_od || period.period_od)} —{" "}
                  {formatDate(row?.period_do || period.period_do)}
                </strong>
              </p>
              <p>
                Datum potpisa: <strong>{formatDate(row?.datum_potpisa)}</strong>
              </p>
              {!row?.datum_potpisa && (
                <p role="status">
                  Pre generisanja unesite datum potpisa kroz „Izmeni period“.
                </p>
              )}
              {template?.koristi_kvartal && <><label>
                Broj kvartala
                <input
                  type="number"
                  min="1"
                  max="9999"
                  step="1"
                  required
                  value={quarter || suggestion}
                  disabled={busy || !canWrite}
                  onChange={(e) => setQuarter(e.target.value)}
                />
              </label>
              <p>
                Predlog je broj prethodnih završenih perioda + 1. Potvrđena
                vrednost čuva se uz period. Generisani dokument nije dokaz da je
                potpisan.
              </p></>}
              <p>
                Šablon:{" "}
                {template
                  ? `${template.naziv} — verzija ${template.verzija}`
                  : "Nema šablona za tip dokumenta ovog perioda. Učitajte ga kroz Šabloni tehničke podrške."}
              </p>
              {canWrite && (
                <label className="report-checkbox">
                  <input
                    type="checkbox"
                    checked={pdf && !!templates.data?.pdf_available}
                    disabled={busy || !templates.data?.pdf_available}
                    onChange={(e) => setPDF(e.target.checked)}
                  />
                  Generiši i PDF (DOCX se uvek čuva)
                </label>
              )}
              {canWrite && !templates.data?.pdf_available && (
                <p>
                  PDF zahteva LibreOffice na serveru i podešen LIBREOFFICE_PATH.
                  DOCX je dostupan.
                </p>
              )}
              <h3>Generisani izveštaji</h3>
              {reports.isPending ? (
                <p>Učitavanje…</p>
              ) : !reports.data?.items?.length ? (
                <p>Nema generisanih izveštaja.</p>
              ) : (
                reports.data.items.map((r) => (
                  <div className="report-history" key={r.id}>
                    <p>
                      {r.broj_kvartala > 0 ? `Kvartal ${r.broj_kvartala} · ` : ""}Potpis{" "}
                      {formatDate(r.datum_potpisa)} · Generisano {r.kreiran}
                    </p>
                    <button
                      type="button"
                      className="button"
                      onClick={() =>
                        download(
                          `${base}/teh-period/${period.id}/izvestaji/${r.id}?format=docx`,
                          `izvestaj-${r.id}.docx`,
                        )
                      }
                    >
                      Preuzmi DOCX
                    </button>
                    {r.ima_pdf && (
                      <button
                        type="button"
                        className="button"
                        onClick={() =>
                          download(
                            `${base}/teh-period/${period.id}/izvestaji/${r.id}?format=pdf`,
                            `izvestaj-${r.id}.pdf`,
                          )
                        }
                      >
                        Preuzmi PDF
                      </button>
                    )}
                  </div>
                ))
              )}
            </>
          ) : (
            <>
              <p>
                U Word šablonu ostavite stalni tekst ugovora, RUN broj i
                potpisnika. Aplikacija popunjava oznake u tekstu, tabelama,
                zaglavlju i podnožju.
              </p>
              <p>
                Obavezne oznake:{" "}
                <code>{"{{datum_od}}, {{datum_do}}, {{datum_potpisa}}"}</code>.
                Za kvartal dodajte <code>{"{{broj_kvartala}}"}</code>. Stara
                oznaka <code>{"{{sysdate}}"}</code> takođe koristi datum
                potpisa, ne današnji datum.
              </p>
              <label>
                Tip dokumenta
                <select
                  value={type}
                  disabled={busy}
                  required
                  onChange={(e) => setType(Number(e.target.value))}
                >
                  {types.map((t) => (
                    <option value={t.id} key={t.id}>
                      {t.naziv}
                    </option>
                  ))}
                </select>
              </label>
              {canWrite && (
                <label>
                  DOCX šablon (do 10 MB)
                  <input
                    type="file"
                    accept=".docx"
                    required
                    disabled={busy}
                    onChange={(e) => setFile(e.target.files[0] || null)}
                  />
                </label>
              )}
              <p>
                Nova verzija zamenjuje aktivni šablon za ovaj tip dokumenta.
                Prethodne verzije i već generisani izveštaji ostaju sačuvani.
              </p>
              {(templates.data?.items || []).map((t) => (
                <p key={t.id}>
                  <button
                    type="button"
                    className="button"
                    onClick={() => download(`${base}/sabloni/${t.id}`, t.naziv)}
                  >
                    {t.naziv} — verzija {t.verzija}
                  </button>
                </p>
              ))}
            </>
          )}
          {blocked && (
            <p role="alert">
              Proverite aktuelne podatke pre ponavljanja.{" "}
              <button
                type="button"
                className="button"
                disabled={busy}
                onClick={async () => {
                  const results = await Promise.all([
                    templates.refetch(),
                    ...(period ? [tracking.refetch(), reports.refetch()] : []),
                  ]);
                  if (results.every((r) => !r.error)) setBlocked(false);
                }}
              >
                Osveži podatke
              </button>
            </p>
          )}
        </div>
        <div className="contact-actions">
          <button
            type="button"
            className="button"
            disabled={busy}
            onClick={onClose}
          >
            Zatvori
          </button>
          {canWrite && (
            <button
              className="button primary"
              disabled={
                busy ||
                blocked ||
                !!error ||
                templates.isFetching ||
                (period
                  ? !row ||
                    tracking.isFetching ||
                    !row.datum_potpisa ||
                    !template ||
                    row.status === "OTKAZAN"
                  : !file || !type)
              }
            >
              {busy
                ? "Obrada…"
                : period
                  ? "Generiši izveštaj"
                  : "Sačuvaj šablon"}
            </button>
          )}
        </div>
      </form>
    </DocumentDialog>
  );
}
