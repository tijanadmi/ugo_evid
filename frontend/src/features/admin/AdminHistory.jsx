import { useState } from "react";
import { useAdminQuery, adminDate, AdminPagination } from "./useAdmin";

function snapshot(value) {
  try {
    return JSON.stringify(JSON.parse(value), null, 2);
  } catch {
    return value || "—";
  }
}
export default function AdminHistory() {
  const [page, setPage] = useState(1);
  const query = useAdminQuery(
    ["history", page],
    `/admin/istorija?page_id=${page}&page_size=20`,
  );
  return (
    <section
      className="admin-panel"
      aria-label="Istorija administrativnih promena"
    >
      <h2>Istorija promena</h2>
      <p>
        Promene korisnika i dodeljenih prava, kao i administratorsko preuzimanje
        i zatvaranje problema.
      </p>
      {query.error && <p role="alert">{query.error.message}</p>}
      <div className="admin-table-scroll">
        <table>
          <thead>
            <tr>
              <th>Vreme</th>
              <th>Administrator</th>
              <th>Akcija</th>
              <th>Podatak</th>
              <th>Promena</th>
            </tr>
          </thead>
          <tbody>
            {query.data?.items?.map((item) => (
              <tr key={item.id}>
                <td>{adminDate(item.vreme)}</td>
                <td>{item.korisnik || item.id_korisnika}</td>
                <td>{item.akcija}</td>
                <td>
                  {item.vrsta} #{item.id_sloga}
                </td>
                <td>
                  <details>
                    <summary>Pre i posle</summary>
                    <h3>Pre</h3>
                    <pre>{snapshot(item.pre)}</pre>
                    <h3>Posle</h3>
                    <pre>{snapshot(item.posle)}</pre>
                  </details>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {query.isPending && <p>Učitavanje…</p>}
      {!query.isPending && !query.error && !query.data?.items?.length && (
        <p>Nema evidentiranih promena.</p>
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
