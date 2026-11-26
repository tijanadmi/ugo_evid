import { useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import toast from "react-hot-toast";
import { useAuth } from "../../context/useAuth";
import { usePermissions } from "../../context/usePermissions";
import { announceBusinessChange } from "../../services/DataSync";
import { getContactRoles } from "../../services/apiContacts";
import AddContactDialog from "../partners/AddContactDialog";
import { useContactActions } from "../partners/useContactActions";
import DocumentDialog from "../documents/DocumentDialog";
import ConfirmAction from "../../ui/ConfirmAction";
import RowMenu from "../../ui/RowMenu";

export default function ContractPeople({ item }) {
  const { api, user } = useAuth();
  const permissions = usePermissions();
  const client = useQueryClient();
  const writable =
    permissions.canWrite(item.id_ugo_org) && item.otvoren_ug === "X";
  const url = `/ugo_evid/${item.id_ugo_evid}/lica`;
  const query = useQuery({
    queryKey: ["contract-people", user.username, item.id_ugo_evid],
    queryFn: ({ signal }) => api(url, { signal }),
  });
  const [editor, setEditor] = useState(null);
  const roles = useQuery({
    queryKey: ["contact-roles", user.username],
    queryFn: () => getContactRoles(api),
    enabled: writable && !!editor,
  });
  const contacts = useContactActions(item.id_sap_dobavljac);
  const [adding, setAdding] = useState(false);
  const [ending, setEnding] = useState(null);
  const [busy, setBusy] = useState(false);
  const [blocked, setBlocked] = useState(false);
  const items = query.data?.items || [];
  const currentSLM = items.find(
    (p) => p.status === "A" && p.id_ugo_dob_lica_rola === 1,
  );
  function open(role) {
    setBlocked(false);
    setEditor({
      role: String(role || ""),
      person: "",
      previous: currentSLM?.id || 0,
      version: currentSLM?.version || "0",
    });
  }
  async function save(remove = false, createdPerson = null) {
    if (busy || blocked) return;
    setBusy(true);
    try {
      await api(url, {
        method: remove ? "DELETE" : "POST",
        body: remove
          ? { id: ending.id, version: ending.version }
          : {
              id_ugo_dob_lica: Number(createdPerson?.id || editor.person),
              id_ugo_dob_lica_rola: Number(editor.role),
              prethodni_slm: editor.previous,
              prethodna_verzija: editor.version,
            },
      });
      setEditor(null);
      setEnding(null);
      announceBusinessChange(client);
      toast.success(
        remove ? "Angažovanje je završeno." : "Lice je povezano sa ugovorom.",
      );
    } catch (e) {
      toast.error(
        createdPerson
          ? `Lice je sačuvano u imeniku, ali nije povezano sa ugovorom. ${e.message}`
          : e.message,
      );
      setBlocked(true);
      await query.refetch();
    } finally {
      setBusy(false);
    }
  }
  return (
    <>
      <h3>Lica angažovana na ugovoru</h3>
      {query.isPending && <p>Učitavanje lica…</p>}
      {query.error && (
        <p role="alert">
          {query.error.message}{" "}
          <button className="button" onClick={() => query.refetch()}>
            Pokušaj ponovo
          </button>
        </p>
      )}
      {!query.isPending && !query.error && !currentSLM && (
        <p role="status">Za ovaj ugovor nije izabran aktivni SLM.</p>
      )}
      {writable && (
        <div className="contract-people-actions">
          <button
            className="button primary"
            disabled={query.isFetching || !!query.error || contacts.busy}
            onClick={() => open(1)}
          >
            {currentSLM ? "Promeni SLM-a" : "Izaberi SLM-a"}
          </button>{" "}
          <button
            className="button"
            disabled={query.isFetching || !!query.error || contacts.busy}
            onClick={() => open()}
          >
            Dodaj lice na ugovor
          </button>
        </div>
      )}
      <ul className="partner-people" aria-label="Lica ugovora">
        {items
          .filter((p) => p.status === "A")
          .map((p) => (
            <li
              key={p.id}
              className={
                p.id_ugo_dob_lica_rola === 1
                  ? "partner-person partner-person-slm"
                  : "partner-person"
              }
            >
              <div className="partner-person-identity">
                <strong>{p.ime}</strong>
                <span>{p.rola}</span>
              </div>
              <div className="partner-person-contact">
                <span>{p.email || "—"}</span>
                <span>{p.telefon || "—"}</span>
              </div>
              {p.status_lica !== "A" && (
                <p role="status">
                  Lice je neaktivno kod dobavljača.
                  {writable && " Izaberite zamenu ili završite angažovanje."}
                </p>
              )}
              {writable && (
                <RowMenu
                  label={`Akcije za lice ${p.ime}`}
                  actions={[
                    {
                      label: "Izmeni podatke lica",
                      icon: "edit",
                      onClick: () =>
                        contacts.open({ id: p.id_ugo_dob_lica }, "edit"),
                    },
                    {
                      label: "Završi angažovanje",
                      icon: "trash",
                      onClick: () => {
                        setBlocked(false);
                        setEnding(p);
                      },
                    },
                  ]}
                />
              )}
            </li>
          ))}
      </ul>
      {items.some((p) => p.status === "P") && (
        <details>
          <summary>Prethodna angažovanja</summary>
          <ul>
            {items
              .filter((p) => p.status === "P")
              .map((p) => (
                <li key={p.id}>
                  {p.ime} — {p.rola}
                </li>
              ))}
          </ul>
        </details>
      )}
      {contacts.dialog}
      {editor && (
        <DocumentDialog
          title="Lice na ugovoru"
          busy={busy}
          onClose={() => !busy && setEditor(null)}
        >
          <form
            className="contact-editor"
            onSubmit={(e) => {
              e.preventDefault();
              save();
            }}
          >
            <p>
              Izaberite lice iz imenika dobavljača i njegovu ulogu na ovom
              ugovoru. Promena ne utiče na druge ugovore.
            </p>
            <label>
              Uloga na ugovoru
              <select
                required
                disabled={busy || blocked}
                value={editor.role}
                onChange={(e) => setEditor({ ...editor, role: e.target.value })}
              >
                <option value="">Izaberite ulogu</option>
                {(roles.data || []).map((r) => (
                  <option key={r.id} value={r.id}>
                    {r.naziv}
                  </option>
                ))}
              </select>
            </label>
            <label>
              Lice dobavljača
              <select
                required
                disabled={busy || blocked}
                value={editor.person}
                onChange={(e) =>
                  setEditor({ ...editor, person: e.target.value })
                }
              >
                <option value="">Izaberite lice</option>
                {(query.data?.candidates || []).map((p) => (
                  <option key={p.id_ugo_dob_lica} value={p.id_ugo_dob_lica}>
                    {p.ime} — {p.email}
                  </option>
                ))}
              </select>
            </label>
            {roles.error && <p role="alert">{roles.error.message}</p>}
            <button
              type="button"
              className="button"
              disabled={
                busy ||
                blocked ||
                !editor.role ||
                roles.isPending ||
                !!roles.error
              }
              onClick={() => setAdding(true)}
            >
              Dodaj novo lice u imenik
            </button>
            <p>
              Za novo lice prvo izaberite ulogu na ugovoru. Čuvanjem novog lica
              ono se dodaje u imenik i odmah povezuje sa ovim ugovorom.
            </p>
            {editor.role === "1" && currentSLM && (
              <p>
                Čuvanjem novog SLM-a završava se prethodno SLM angažovanje na
                ovom ugovoru.
              </p>
            )}
            {blocked && (
              <p role="alert">
                Zatvorite dijalog i ponovo učitajte lica pre novog pokušaja.
              </p>
            )}
            <div className="contact-actions">
              <button
                type="button"
                className="button"
                disabled={busy}
                onClick={() => setEditor(null)}
              >
                Odustani
              </button>
              <button
                className="button primary"
                disabled={busy || blocked || roles.isPending || !!roles.error}
              >
                Sačuvaj angažovanje
              </button>
            </div>
          </form>
        </DocumentDialog>
      )}
      {adding && (
        <AddContactDialog
          supplierID={item.id_sap_dobavljac}
          onCreated={async (person) => {
            setAdding(false);
            if (!person?.id) {
              toast.error(
                "Lice je sačuvano u imeniku, ali server nije vratio njegov ID. Osvežite listu i izaberite lice za povezivanje.",
              );
              await query.refetch();
              return;
            }
            setEditor((current) => ({ ...current, person: String(person.id) }));
            await save(false, person);
          }}
          onClose={() => {
            setAdding(false);
            query.refetch();
          }}
        />
      )}
      {ending && (
        <ConfirmAction
          modal
          title="Završiti angažovanje?"
          disabled={busy}
          blocked={blocked}
          onCancel={() => setEnding(null)}
          onConfirm={() => save(true)}
        >
          <p>
            {ending.ime} više neće biti aktivno lice ovog ugovora. Podaci u
            imeniku i drugi ugovori ostaju nepromenjeni.
          </p>
          {blocked && (
            <p>Zatvorite potvrdu i osvežite podatke pre ponavljanja.</p>
          )}
        </ConfirmAction>
      )}
    </>
  );
}
