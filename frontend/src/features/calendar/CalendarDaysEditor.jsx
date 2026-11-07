import DocumentDialog from "../documents/DocumentDialog";
import ConfirmAction from "../../ui/ConfirmAction";

export default function CalendarDaysEditor({
  draft,
  setDraft,
  year,
  busy,
  blocked,
  cancelDraft,
  setCancelDraft,
  onSubmit,
}) {
  return (
    <>
      {" "}
      {draft && (
        <DocumentDialog
          title="Izmena radnih dana"
          busy={busy}
          onClose={() => setCancelDraft(true)}
        >
          <form
            className="contact-editor"
            onSubmit={(e) => {
              e.preventDefault();
              onSubmit(draft);
            }}
          >
            <fieldset disabled={busy || blocked}>
              <label>
                Od datuma
                <input
                  type="date"
                  required
                  min={`${year}-01-01`}
                  max={`${year}-12-31`}
                  value={draft.od}
                  onChange={(e) => setDraft({ ...draft, od: e.target.value })}
                />
              </label>
              <label>
                Do datuma
                <input
                  type="date"
                  required
                  min={draft.od}
                  max={`${year}-12-31`}
                  value={draft.do}
                  onChange={(e) => setDraft({ ...draft, do: e.target.value })}
                />
              </label>
              <label>
                Radni dan
                <select
                  aria-label="Radni dan"
                  value={draft.radni}
                  onChange={(e) =>
                    setDraft({ ...draft, radni: e.target.value })
                  }
                >
                  <option value="N">Ne — neradni dan</option>
                  <option value="D">Da — radni dan</option>
                </select>
              </label>
              <label>
                Opis
                <input
                  maxLength={200}
                  value={draft.opis}
                  onChange={(e) => setDraft({ ...draft, opis: e.target.value })}
                />
              </label>
              <p>
                Promena važi za sve datume u opsegu. Ručne izmene su zaštićene
                od uvoza praznika i traže ponovnu potvrdu.
              </p>
            </fieldset>
            {blocked && (
              <p role="alert">
                Kalendar je promenjen. Zatvorite prozor i ponovo otvorite
                izmenu.
              </p>
            )}
            <div className="contact-actions">
              <button
                type="button"
                className="button"
                disabled={busy}
                onClick={() => setCancelDraft(true)}
              >
                Odustani
              </button>
              <button className="button primary" disabled={busy || blocked}>
                Sačuvaj dane
              </button>
            </div>
          </form>
          {cancelDraft && (
            <ConfirmAction
              modal
              title="Odustati od izmena?"
              disabled={busy}
              onCancel={() => setCancelDraft(false)}
              onConfirm={() => {
                setDraft(null);
                setCancelDraft(false);
              }}
            >
              <p>Nesačuvane izmene biće izgubljene.</p>
            </ConfirmAction>
          )}
        </DocumentDialog>
      )}
    </>
  );
}
