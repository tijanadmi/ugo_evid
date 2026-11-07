import DocumentDialog from "../documents/DocumentDialog";

export default function HolidayImportDialog({
  preview,
  setPreview,
  year,
  busy,
  onSubmit,
}) {
  return (
    <>
      {" "}
      {preview && (
        <DocumentDialog
          title={`Praznici Srbije — ${year}.`}
          busy={busy}
          onClose={() => setPreview(null)}
        >
          <div className="calendar-import">
            <p>
              Predlog: Nager.Date, uz dopunu vaskršnjeg perioda i pravila
              prenosa državnih praznika. Proverite prema{" "}
              <a
                href="https://www.nbs.rs/sr/o_nbs/radno-vreme-i-neradni-dani/"
                target="_blank"
                rel="noreferrer"
              >
                zvaničnoj objavi
              </a>
              . Uvoz ne potvrđuje godinu.
            </p>
            <p>
              Ručne korekcije nisu označene za uvoz. Označi ih samo ako želiš da
              ih zameniš. Uvoze se samo izabrani datumi.
            </p>
            <ul className="calendar-holiday-list">
              {preview.rows.map((h) => (
                <li key={h.datum}>
                  <label>
                    <input
                      type="checkbox"
                      disabled={busy}
                      checked={h.selected}
                      onChange={(e) =>
                        setPreview((p) => ({
                          ...p,
                          rows: p.rows.map((x) =>
                            x.datum === h.datum
                              ? { ...x, selected: e.target.checked }
                              : x,
                          ),
                        }))
                      }
                    />
                    <span>
                      <strong>
                        {h.datum} — {h.naziv}
                      </strong>
                      <small>
                        {h.protectedDay
                          ? "Ručna korekcija — označavanjem odobravaš zamenu"
                          : h.poreklo}
                      </small>
                    </span>
                  </label>
                </li>
              ))}
            </ul>
            <div className="contact-actions">
              <button
                className="button"
                disabled={busy}
                onClick={() => setPreview(null)}
              >
                Odustani
              </button>
              <button
                className="button primary"
                disabled={busy || !preview.rows.some((h) => h.selected)}
                onClick={() =>
                  onSubmit({
                    action: "import",
                    version: preview.version,
                    praznici: preview.rows
                      .filter((h) => h.selected)
                      .map(({ datum, naziv, protectedDay }) => ({
                        datum,
                        naziv,
                        zameni_rucno: protectedDay,
                      })),
                  })
                }
              >
                Uvezi izabrane praznike
              </button>
            </div>
          </div>
        </DocumentDialog>
      )}
    </>
  );
}
