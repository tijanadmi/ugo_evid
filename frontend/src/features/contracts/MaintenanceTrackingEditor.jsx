import DateInput from "../../ui/DateInput";
import Icon from "../../ui/Icon";
import { Link } from "react-router-dom";
import { formatDate } from "../../utils/contractFormatting";
import DocumentDialog from "../documents/DocumentDialog";
import ConfirmAction from "../../ui/ConfirmAction";
import { trackingLabels as labels } from "./trackingConstants";

export default function MaintenanceTrackingEditor({
  editor,
  setEditor,
  busy,
  blocked,
  lost,
  save,
  discard,
  confirm,
  setConfirm,
  preview,
  change,
  documentTypes,
  funds,
}) {
  return (
    <>
      {" "}
      {editor?.remove && (
        <ConfirmAction
          modal
          title={
            editor.kind === "periodi"
              ? "Obrisati tehnički period?"
              : "Obrisati razvojni zahtev?"
          }
          confirmLabel={
            editor.kind === "periodi" ? "Obriši period" : "Obriši zahtev"
          }
          cancelLabel="Odustani"
          variation="danger"
          disabled={busy}
          blocked={blocked || lost}
          onConfirm={() => save()}
          onCancel={discard}
        >
          <p>
            {editor.kind === "periodi"
              ? `Period ${formatDate(editor.row.period_od)} — ${formatDate(editor.row.period_do)} biće obrisan. Povezani dokument ostaje sačuvan.`
              : "Zahtev se uklanja iz aktivne liste i obračuna časova. Dokumenti i evidencija ostaju sačuvani."}
          </p>
          {(blocked || lost) && (
            <p>
              Pravo brisanja više nije važeće. Zatvorite prozor i osvežite
              podatke.
            </p>
          )}
        </ConfirmAction>
      )}
      {editor && !editor.remove && (
        <DocumentDialog
          title={
            editor.generate
              ? "Generisanje tehničkih perioda"
              : editor.kind === "periodi"
                ? "Tehnički period"
                : "Razvojni zahtev"
          }
          onClose={() => setConfirm(true)}
          busy={busy}
        >
          <form className="contact-editor tracking-editor" onSubmit={save}>
            <fieldset disabled={busy || lost || blocked}>
              {editor.generate ? (
                <>
                  <label>
                    Način obračuna
                    <select
                      value={editor.mode}
                      onChange={(e) =>
                        setEditor((s) => ({
                          ...s,
                          mode: e.target.value,
                          preview: null,
                        }))
                      }
                    >
                      <option
                        value="calendar"
                        disabled={
                          ![1, 3, 6, 12].includes(
                            editor.settings.odrzavanje.teh_period_meseci,
                          )
                        }
                      >
                        Kalendarski mesec / kvartal / polugodište / godina
                      </option>
                      <option value="anchor">
                        Od datuma početka održavanja
                      </option>
                    </select>
                  </label>
                  <p>
                    Početak:{" "}
                    {formatDate(
                      editor.settings.odrzavanje.pocetak_odrzavanja ||
                        editor.settings.sap_pocetak,
                    )}
                    ; kraj: {formatDate(editor.settings.sap_kraj)}. Poslednji
                    period se završava na kraju ugovora.
                  </p>
                  <button className="button" type="button" onClick={preview}>
                    Prikaži predlog
                  </button>
                  {editor.preview && (
                    <div className="tracking-preview">
                      <ol>
                        {editor.preview.map((p, i) => (
                          <li key={i}>
                            {formatDate(p.period_od)} —{" "}
                            {formatDate(p.period_do)}
                          </li>
                        ))}
                      </ol>
                    </div>
                  )}
                </>
              ) : (
                <>
                  {editor.kind === "periodi" ? (
                    <>
                      <label>
                        Period od
                        <DateInput
                          required
                          value={editor.row.period_od}
                          onChange={(e) => change("period_od", e.target.value)}
                        />
                      </label>
                      <label>
                        Period do
                        <DateInput
                          required
                          value={editor.row.period_do}
                          onChange={(e) => change("period_do", e.target.value)}
                        />
                      </label>
                      <label>
                        Tip dokumenta
                        <select
                          required
                          value={editor.row.id_tip_doc}
                          onChange={(e) =>
                            change("id_tip_doc", Number(e.target.value))
                          }
                        >
                          <option value="0">Izaberite tip</option>
                          {documentTypes
                            .filter((t) =>
                              ["POTVRDA_TEH", "ZAPISNIK_TEH"].includes(t.sifra),
                            )
                            .map((t) => (
                              <option key={t.id} value={t.id}>
                                {t.naziv}
                              </option>
                            ))}
                        </select>
                      </label>
                      <label>
                        Rok dokumenta (opciono)
                        <DateInput
                          value={editor.row.rok}
                          onChange={(e) => change("rok", e.target.value)}
                        />
                      </label>
                    </>
                  ) : (
                    <>
                      <label>
                        Broj zahteva
                        <input
                          required
                          maxLength={100}
                          value={editor.row.broj_zahteva}
                          onChange={(e) =>
                            change("broj_zahteva", e.target.value)
                          }
                        />
                      </label>
                      <label>
                        Datum zahteva
                        <DateInput
                          required
                          value={editor.row.datum_zahteva}
                          onChange={(e) =>
                            change("datum_zahteva", e.target.value)
                          }
                        />
                      </label>
                      <label>
                        Vrsta rada
                        <select
                          aria-label="Vrsta rada"
                          required
                          value={editor.row.id_vrsta_rada || ""}
                          onChange={(e) =>
                            change("id_vrsta_rada", Number(e.target.value))
                          }
                        >
                          <option value="">Izaberite vrstu rada</option>
                          {funds.map((f) => (
                            <option
                              key={f.id_vrsta_rada}
                              value={f.id_vrsta_rada}
                            >
                              {f.naziv} — preostalo {f.preostalo} č.
                            </option>
                          ))}
                        </select>
                      </label>
                      <label>
                        Broj časova
                        <input
                          required
                          type="number"
                          min="1"
                          max="2147483647"
                          step="1"
                          value={editor.row.broj_casova}
                          onChange={(e) =>
                            change("broj_casova", Number(e.target.value))
                          }
                        />
                      </label>
                      <label>
                        Rok u radnim danima
                        <input
                          required
                          type="number"
                          min="1"
                          max="3660"
                          step="1"
                          value={editor.row.rok_radnih_dana}
                          onChange={(e) =>
                            change("rok_radnih_dana", Number(e.target.value))
                          }
                        />
                      </label>
                      <p>
                        Rok se računa pri čuvanju iz kalendara radnih dana,
                        počev od narednog dana. Usluga mora biti realizovana
                        najkasnije na dan roka. Zapisnik može biti potpisan kasnije,
                        ali ne pre realizacije.
                        {editor.row.rok &&
                          ` Sačuvani rok: ${formatDate(editor.row.rok)}.`}
                      </p>
                      <p>
                        <Link
                          target="_blank"
                          rel="noopener noreferrer"
                          to={`/ugovori/radni-kalendar${editor.row.datum_zahteva ? "?godina=" + editor.row.datum_zahteva.slice(0, 4) : ""}`}
                        >
                          Otvori radni kalendar u novom tabu
                        </Link>
                      </p>
                    </>
                  )}
                  <label>
                    Status
                    <select
                      aria-label="Status"
                      value={editor.row.status}
                      onChange={(e) => change("status", e.target.value)}
                    >
                      {(editor.kind === "periodi"
                        ? [
                            "PLANIRAN",
                            "PRIPREMA",
                            "NA_POTPISU",
                            "ZAVRSEN",
                            "OTKAZAN",
                          ]
                        : [
                            "NACRT",
                            "PODNET",
                            "U_RADU",
                            "NA_POTPISU",
                            "ZAVRSEN",
                            "OTKAZAN",
                          ]
                      ).map((s) => (
                        <option key={s} value={s}>
                          {labels[s]}
                        </option>
                      ))}
                    </select>
                  </label>
                  {editor.kind === "zahtevi" && (
                    <label>
                      Datum realizacije
                      <DateInput
                        required={editor.row.status === "ZAVRSEN"}
                        value={editor.row.datum_realizacije || ""}
                        onChange={(e) =>
                          change("datum_realizacije", e.target.value)
                        }
                      />
                    </label>
                  )}
                  <label>
                    {editor.kind === "zahtevi"
                      ? "Datum potpisa zapisnika"
                      : "Datum potpisa"}
                    <DateInput
                      required={editor.row.status === "ZAVRSEN"}
                      value={editor.row.datum_potpisa}
                      onChange={(e) => change("datum_potpisa", e.target.value)}
                    />
                  </label>
                  {editor.kind === "zahtevi" &&
                    editor.row.status === "ZAVRSEN" && (
                      <p>
                        Unesite originalni datum zahteva i datum potpisa
                        završnog zapisnika. Časovi ulaze u utrošeni fond. Za
                        izračunavanje roka potreban je potvrđen radni kalendar
                        za taj period.
                      </p>
                    )}
                  <p className="muted">
                    Dokument možete dodati odmah nakon čuvanja ili kasnije iz
                    menija reda.
                  </p>
                  <label>
                    Napomena
                    <textarea
                      maxLength={2000}
                      value={editor.row.napomena}
                      onChange={(e) => change("napomena", e.target.value)}
                    />
                  </label>
                </>
              )}
            </fieldset>
            {(blocked || lost) && (
              <p role="alert">
                Pravo izmene više nije važeće. Zatvorite prozor i ponovo
                učitajte podatke.
              </p>
            )}
            <div className="contact-actions">
              <button
                className="button"
                type="button"
                disabled={busy}
                onClick={() => setConfirm(true)}
              >
                Odustani
              </button>
              <button
                className="button primary"
                disabled={
                  busy ||
                  blocked ||
                  lost ||
                  (editor.generate && !editor.preview?.length)
                }
              >
                Sačuvaj
              </button>
              {!editor.generate && (
                <button
                  className="button primary"
                  type="submit"
                  name="action"
                  value="upload"
                  disabled={busy || blocked || lost}
                >
                  <Icon name="upload" />
                  Sačuvaj i dodaj dokument
                </button>
              )}
            </div>
          </form>
          {confirm && (
            <ConfirmAction
              modal
              title="Odustati od unosa?"
              disabled={busy}
              onCancel={() => setConfirm(false)}
              onConfirm={discard}
            >
              <p>Nesačuvani podaci biće izgubljeni.</p>
            </ConfirmAction>
          )}
        </DocumentDialog>
      )}
    </>
  );
}
