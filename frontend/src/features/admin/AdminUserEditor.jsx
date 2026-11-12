import { useEffect, useRef, useState } from "react";
import toast from "react-hot-toast";
import { useAuth } from "../../context/useAuth";
import DocumentDialog from "../documents/DocumentDialog";
import ConfirmAction from "../../ui/ConfirmAction";
import { useAdminQuery } from "./useAdmin";
import RoleAssignments, { EffectiveAccess } from "./RoleAssignments";

export default function AdminUserEditor({ id, onClose, onSaved }) {
  const { api } = useAuth();
  const catalog = useAdminQuery(["catalog"], "/admin/sifarnici");
  const query = useAdminQuery(["user", id], `/admin/korisnici/${id}`, id > 0);
  const [draft, setDraft] = useState(
    id ? null : { ad_sifra: "", ime: "", status: "A", version: "0", roles: [] },
  );
  const [dirty, setDirty] = useState(false);
  const [busy, setBusy] = useState(false);
  const [blocked, setBlocked] = useState(false);
  const [confirm, setConfirm] = useState(false);
  const pending = useRef(false);
  useEffect(() => {
    if (query.data && !draft && !query.isFetching) setDraft(query.data);
  }, [query.data, draft, query.isFetching]);
  function change(changes) {
    setDirty(true);
    setDraft((v) => ({ ...v, ...changes }));
  }
  function close() {
    if (busy) return;
    if (dirty) setConfirm(true);
    else onClose();
  }
  async function save(e) {
    e.preventDefault();
    if (pending.current || blocked) return;
    const keys = draft.roles.map(
      (g) => `${g.id_ugo_rola}:${g.opseg}:${g.id_ugo_org}`,
    );
    if (new Set(keys).size !== keys.length) {
      toast.error("Isto pravo je uneto više puta.");
      return;
    }
    pending.current = true;
    setBusy(true);
    try {
      await api(id ? `/admin/korisnici/${id}` : "/admin/korisnici", {
        method: id ? "PUT" : "POST",
        body: draft,
      });
      toast.success("Korisnik i prava su sačuvani.");
      onSaved();
    } catch (error) {
      toast.error(error.message);
      if (error.status === 409 || error.status === 0 || error.status >= 500)
        setBlocked(true);
    } finally {
      pending.current = false;
      setBusy(false);
    }
  }
  return (
    <DocumentDialog
      title={id ? "Korisnik i prava" : "Novi korisnik"}
      onClose={close}
      busy={busy}
      wide
    >
      {confirm && (
        <ConfirmAction
          modal
          title="Odustati od izmene korisnika?"
          onCancel={() => setConfirm(false)}
          onConfirm={onClose}
        >
          <p>Nesačuvane izmene biće izgubljene.</p>
        </ConfirmAction>
      )}
      <div className="admin-editor">
        {(catalog.error || query.error) && (
          <p role="alert">{catalog.error?.message || query.error?.message}</p>
        )}
        {draft && catalog.data ? (
          <form onSubmit={save}>
            <fieldset disabled={busy}>
              <label>
                AD korisničko ime
                <input
                  required
                  maxLength={200}
                  disabled={id > 0}
                  value={draft.ad_sifra}
                  onChange={(e) => change({ ad_sifra: e.target.value })}
                />
              </label>
              <label>
                Ime i prezime
                <input
                  maxLength={100}
                  value={draft.ime || ""}
                  onChange={(e) => change({ ime: e.target.value })}
                />
              </label>
              <label>
                Status korisnika
                <select
                  value={draft.status}
                  onChange={(e) => change({ status: e.target.value })}
                >
                  <option value="A">Aktivan</option>
                  <option value="N">Neaktivan</option>
                </select>
              </label>
              <p>
                Unesite postojeći AD nalog. Identitet i lozinka proveravaju se
                pri prijavi kroz AD.
              </p>
              <RoleAssignments
                roles={draft.roles || []}
                onChange={(roles) => change({ roles })}
                catalog={catalog.data}
              />
              <EffectiveAccess
                roles={draft.roles || []}
                status={draft.status}
                organizations={catalog.data.organizations}
              />
            </fieldset>
            {blocked && (
              <p role="alert">
                Proverite aktuelne podatke pre novog pokušaja. Uneti podaci su
                ostali u formi.
              </p>
            )}
            <div className="admin-actions">
              <button
                type="button"
                className="button"
                disabled={busy}
                onClick={close}
              >
                Odustani
              </button>
              {blocked && (
                <button
                  type="button"
                  className="button"
                  disabled={busy}
                  onClick={async () => {
                    if (!id) {
                      onClose();
                      return;
                    }
                    const result = await query.refetch();
                    if (!result.error) {
                      setDraft(result.data);
                      setDirty(false);
                      setBlocked(false);
                    }
                  }}
                >
                  {id ? "Učitaj aktuelne podatke" : "Zatvori i proveri listu"}
                </button>
              )}
              <button
                className="button primary"
                disabled={busy || blocked || !!query.error || !!catalog.error}
              >
                Sačuvaj korisnika
              </button>
            </div>
          </form>
        ) : (
          <p>Učitavanje…</p>
        )}
      </div>
    </DocumentDialog>
  );
}
