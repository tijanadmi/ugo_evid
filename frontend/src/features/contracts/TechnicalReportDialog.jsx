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
  const [format, setFormat] = useState("docx");
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
        const blob = await api(`${base}/teh-period/${period.id}/izvestaj`, {
          responseType: "blob",
          method: "POST",
          signal: controller.signal,
          body: {
            id_template: template.id,
            ...(template.koristi_kvartal
              ? { broj_kvartala: Number(quarter || suggestion) }
              : {}),
            version: row.version,
            format,
          },
        });
        const href = URL.createObjectURL(blob);
        const link = document.createElement("a");
        link.href = href;
        link.download = `tehnicka-podrska-${period.id}.${format}`;
        link.click();
        setTimeout(() => URL.revokeObjectURL(href), 10000);
        toast.success("Dokument je spreman za preuzimanje.");
        await tracking.refetch();
        announceBusinessChange(client);
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
  const error = templates.error || tracking.error;
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
              {template?.koristi_kvartal && (
                <>
                  <label>
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
                    vrednost čuva se uz period. Generisani dokument nije dokaz
                    da je potpisan.
                  </p>
                </>
              )}
              <p>
                Šablon:{" "}
                {template
                  ? `${template.naziv} — verzija ${template.verzija}`
                  : "Nema šablona za tip dokumenta ovog perioda. Učitajte ga kroz Šabloni tehničke podrške."}
              </p>
              <label>
                Format dokumenta
                <select
                  value={format}
                  disabled={busy}
                  onChange={(e) => setFormat(e.target.value)}
                >
                  <option value="docx">DOCX</option>
                  <option value="pdf" disabled={!templates.data?.pdf_available}>
                    PDF
                  </option>
                </select>
              </label>
              <p>Dokument se preuzima na računar i ne čuva se na serveru.</p>
              {canWrite && !templates.data?.pdf_available && (
                <p>
                  PDF zahteva LibreOffice na serveru i podešen LIBREOFFICE_PATH.
                  DOCX je dostupan.
                </p>
              )}
            </>
          ) : (
            <>
              <p>
                Sami pripremite šablon u Wordu za ovaj ugovor i sačuvajte ga u
                DOCX formatu. Unesite stalni tekst, RUN broj, potpisnika i ostale
                podatke koji se ne menjaju između perioda.
              </p>
              <p>
                Na mesta promenljivih podataka unesite sledeće obavezne oznake,
                zajedno sa dvostrukim vitičastim zagradama:
              </p>
              <ul>
                <li><code>{"{{datum_od}}"}</code> — početak tehničkog perioda.</li>
                <li><code>{"{{datum_do}}"}</code> — kraj tehničkog perioda.</li>
                <li><code>{"{{datum_potpisa}}"}</code> — datum potpisa evidentiran uz period.</li>
              </ul>
              <p>
                Ako dokument treba da sadrži broj kvartala, dodajte i oznaku{" "}
                <code>{"{{broj_kvartala}}"}</code>. Ona nije obavezna; ako je
                navedete, aplikacija će ponuditi izbor broja kvartala pre
                generisanja dokumenta.
              </p>
              <p>
                Izaberite tip dokumenta, učitajte pripremljeni DOCX fajl i
                kliknite „Sačuvaj šablon“. Zatim u meniju tehničkog perioda
                izaberite „Generiši / preuzmi izveštaj“. Aplikacija će zameniti
                oznake podacima izabranog perioda i omogućiti preuzimanje
                dokumenta u DOCX ili PDF formatu. Datumi se ispisuju u formatu
                dd.mm.yyyy.
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
                Prethodne verzije šablona ostaju sačuvane.
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
                    ...(period ? [tracking.refetch()] : []),
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
                  ? "Generiši i preuzmi"
                  : "Sačuvaj šablon"}
            </button>
          )}
        </div>
      </form>
    </DocumentDialog>
  );
}
