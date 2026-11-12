import { useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useAdminQuery, AdminPagination } from "./useAdmin";
import AdminUserEditor from "./AdminUserEditor";
import { announceBusinessChange } from "../../services/DataSync";

export default function AdminUsers() {
  const [filter, setFilter] = useState("");
  const [draft, setDraft] = useState("");
  const [page, setPage] = useState(1);
  const [editing, setEditing] = useState(null);
  const client = useQueryClient();
  const query = useAdminQuery(
    ["users", page, filter],
    `/admin/korisnici?${new URLSearchParams({ page_id: page, page_size: 20, filter })}`,
  );
  return (
    <section className="admin-panel" aria-label="Korisnici i prava">
      <div className="admin-toolbar">
        <form
          onSubmit={(e) => {
            e.preventDefault();
            setFilter(draft.trim());
            setPage(1);
          }}
        >
          <label>
            Pretraga korisnika
            <input
              type="search"
              maxLength={200}
              value={draft}
              onChange={(e) => setDraft(e.target.value)}
            />
          </label>
          <button className="button">Pretraži</button>
        </form>
        <button className="button primary" onClick={() => setEditing(0)}>
          Novi korisnik
        </button>
      </div>
      <p>
        Pristup portalu dodeljuje se postojećem AD nalogu. Lozinke se ne unose
        niti menjaju ovde.
      </p>
      {query.error && <p role="alert">{query.error.message}</p>}
      {query.isPending ? (
        <p>Učitavanje…</p>
      ) : (
        <div className="admin-table-scroll">
          <table>
            <thead>
              <tr>
                <th>AD nalog</th>
                <th>Ime i prezime</th>
                <th>Status</th>
                <th>Akcije</th>
              </tr>
            </thead>
            <tbody>
              {query.data?.items?.map((u) => (
                <tr key={u.id}>
                  <td>{u.ad_sifra}</td>
                  <td>{u.ime}</td>
                  <td>{u.status === "A" ? "Aktivan" : "Neaktivan"}</td>
                  <td>
                    <button className="button" onClick={() => setEditing(u.id)}>
                      Korisnik i prava
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      {!query.isPending && !query.error && !query.data?.items?.length && (
        <p>Nema korisnika za izabrani filter.</p>
      )}
      <AdminPagination
        page={page}
        total={query.data?.total || 0}
        busy={query.isFetching}
        onChange={setPage}
      />
      {editing !== null && (
        <AdminUserEditor
          key={editing}
          id={editing}
          onClose={() => setEditing(null)}
          onSaved={() => {
            setEditing(null);
            client.invalidateQueries({ queryKey: ["admin"] });
            announceBusinessChange(client);
          }}
        />
      )}
    </section>
  );
}
