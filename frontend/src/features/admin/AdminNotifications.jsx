import { useRef, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useQuery } from "@tanstack/react-query";
import toast from "react-hot-toast";
import { useAuth } from "../../context/useAuth";
import { adminDate, AdminPagination } from "./useAdmin";

export default function AdminNotifications() {
  const { api, user } = useAuth();
  const client = useQueryClient();
  const [status, setStatus] = useState("");
  const [page, setPage] = useState(1);
  const [busy, setBusy] = useState(false);
  const pending = useRef(false);
  const query = useQuery({
    queryKey: ["admin", user.username, "notifications", status, page],
    queryFn: ({ signal }) =>
      api(
        `/admin/obavestenja?${new URLSearchParams({ status, page_id: page, page_size: 20 })}`,
        { signal },
      ),
    refetchInterval: 30000,
    retry: false,
  });
  async function action(item, value) {
    if (pending.current) return;
    pending.current = true;
    setBusy(true);
    try {
      await api(`/admin/obavestenja/${item.id}`, {
        method: "PUT",
        body: { version: item.version, action: value },
      });
      toast.success("Obaveštenje je ažurirano.");
    } catch (error) {
      toast.error(error.message);
    } finally {
      pending.current = false;
      setBusy(false);
      client.invalidateQueries({ queryKey: ["admin"] });
    }
  }
  return (
    <section className="admin-panel" aria-label="Administrativna obaveštenja">
      <div className="admin-toolbar">
        <label>
          Status obaveštenja
          <select
            value={status}
            onChange={(e) => {
              setStatus(e.target.value);
              setPage(1);
            }}
          >
            <option value="">Sva nerešena</option>
            <option value="OTVORENO">Otvorena</option>
            <option value="PREUZETO">Preuzeta</option>
            <option value="RESENO">Rešena</option>
            <option value="SVE">Sva obaveštenja</option>
          </select>
        </label>
        <button
          className="button"
          onClick={() => query.refetch()}
          disabled={query.isFetching}
        >
          Osveži
        </button>
      </div>
      <p>
        Neuspešno brisanje dokumenta prijavljuje se nakon 15 minuta. Ponovni
        pokušaji ažuriraju isto obaveštenje. Kada brisanje uspe, ono se
        automatski označava kao rešeno.
      </p>
      {query.error && <p role="alert">{query.error.message}</p>}
      {query.isPending ? (
        <p>Učitavanje…</p>
      ) : (
        query.data?.items?.map((item) => (
          <article className="admin-notification" key={item.id}>
            <h2>{item.naslov}</h2>
            {item.originalni_naziv && (
              <p>
                <strong>{item.originalni_naziv}</strong> · Ugovor #
                {item.id_ugo_evid} · Pokušaja brisanja: {item.broj_pokusaja}
              </p>
            )}
            <dl>
              <div>
                <dt>Status</dt>
                <dd>
                  {
                    {
                      OTVORENO: "Otvoreno",
                      PREUZETO: "Preuzeto",
                      RESENO: "Rešeno",
                    }[item.status]
                  }
                </dd>
              </div>
              <div>
                <dt>Dokument / slog</dt>
                <dd>{item.id_sloga}</dd>
              </div>
              <div>
                <dt>Prvi put</dt>
                <dd>{adminDate(item.prvi_put)}</dd>
              </div>
              <div>
                <dt>Poslednji put</dt>
                <dd>{adminDate(item.poslednji_put)}</dd>
              </div>
              <div>
                <dt>Broj prijava</dt>
                <dd>{item.broj_ponavljanja}</dd>
              </div>
              <div>
                <dt>Preuzeo</dt>
                <dd>{item.preuzeo || "—"}</dd>
              </div>
              {item.reseno && (
                <div>
                  <dt>Rešeno</dt>
                  <dd>{adminDate(item.reseno)}</dd>
                </div>
              )}
            </dl>
            <details>
              <summary>Detalji greške</summary>
              <pre>{item.poruka || "Nema dodatnih detalja."}</pre>
            </details>
            <div className="admin-actions">
              {item.status === "OTVORENO" && (
                <button
                  className="button"
                  disabled={busy}
                  onClick={() => action(item, "PREUZMI")}
                >
                  Preuzmi problem
                </button>
              )}
              {item.status === "PREUZETO" &&
                item.preuzeo?.toLowerCase() === user.username.toLowerCase() && (
                  <button
                    className="button"
                    disabled={busy}
                    onClick={() => action(item, "VRATI")}
                  >
                    Vrati u otvorene
                  </button>
                )}
              {item.status !== "RESENO" && item.vrsta !== "DOC_DELETE" && (
                <button
                  className="button"
                  disabled={busy}
                  onClick={() => action(item, "RESI")}
                >
                  Označi kao rešeno
                </button>
              )}
            </div>
          </article>
        ))
      )}
      {!query.isPending && !query.error && !query.data?.items?.length && (
        <p>Nema obaveštenja za izabrani status.</p>
      )}
      <AdminPagination
        page={page}
        total={query.data?.total || 0}
        busy={query.isFetching}
        onChange={setPage}
      />
    </section>
  );
}
