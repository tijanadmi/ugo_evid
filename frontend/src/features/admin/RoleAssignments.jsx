export default function RoleAssignments({ roles, onChange, catalog }) {
  function update(index, changes) {
    onChange(
      roles.map((role, i) => (i === index ? { ...role, ...changes } : role)),
    );
  }
  return (
    <section className="admin-roles" aria-label="Dodeljena prava">
      <div className="admin-roles-heading">
        <h3>Dodeljena prava</h3>
        <button
          className="button"
          type="button"
          disabled={roles.length >= 50}
          onClick={() =>
            onChange([...roles, { id_ugo_rola: 1, opseg: "OJ", id_ugo_org: 0 }])
          }
        >
          Dodaj pravo
        </button>
      </div>
      {roles.length >= 50 && (
        <p role="status">
          Dostignut je maksimalan broj od 50 dodeljenih prava.
        </p>
      )}
      {roles.map((grant, i) => (
        <div className="admin-role" key={i}>
          <label>
            Rola {i + 1}
            <select
              value={grant.id_ugo_rola}
              onChange={(e) => {
                const role = Number(e.target.value);
                update(i, {
                  id_ugo_rola: role,
                  opseg: role >= 3 ? "GLOBALNO" : "OJ",
                  id_ugo_org: 0,
                });
              }}
            >
              {catalog.roles.map((r) => (
                <option key={r.id} value={r.id}>
                  {r.sifra} — {r.naziv}
                </option>
              ))}
            </select>
          </label>
          <label>
            Opseg {i + 1}
            <select
              disabled={grant.id_ugo_rola >= 3}
              value={grant.opseg}
              onChange={(e) =>
                update(i, { opseg: e.target.value, id_ugo_org: 0 })
              }
            >
              {grant.id_ugo_rola >= 3 ? (
                <option value="GLOBALNO">Globalno</option>
              ) : (
                <>
                  <option value="OJ">Organizaciona jedinica</option>
                  <option value="SVE_OJ">Sve OJ</option>
                </>
              )}
            </select>
          </label>
          {grant.opseg === "OJ" && (
            <label>
              OJ {i + 1}
              <select
                required
                value={grant.id_ugo_org || ""}
                onChange={(e) =>
                  update(i, { id_ugo_org: Number(e.target.value) })
                }
              >
                <option value="">Izaberite OJ</option>
                {catalog.organizations.map((o) => (
                  <option key={o.id} value={o.id}>
                    {o.sifra} — {o.naziv}
                  </option>
                ))}
              </select>
            </label>
          )}
          <button
            className="button"
            type="button"
            onClick={() => onChange(roles.filter((_, index) => i !== index))}
          >
            Ukloni pravo {i + 1}
          </button>
        </div>
      ))}
    </section>
  );
}

export function EffectiveAccess({ roles, status, organizations }) {
  const admin = roles.some((g) => g.id_ugo_rola === 4);
  const visible = organizations.flatMap((o) => {
    const grants = roles.filter(
      (g) =>
        [1, 2].includes(g.id_ugo_rola) &&
        (g.opseg === "SVE_OJ" || g.id_ugo_org === o.id),
    );
    return grants.length
      ? [{ ...o, write: grants.some((g) => g.id_ugo_rola === 2) }]
      : [];
  });
  return (
    <details className="admin-effective">
      <summary>Efektivna prava po trenutnom izboru</summary>
      {status !== "A" ? (
        <p>Neaktivan korisnik nema pristup portalu.</p>
      ) : (
        <>
          <p>
            Administracija: {admin ? "Da" : "Ne"}. Uređivanje kalendara:{" "}
            {admin || roles.some((g) => g.id_ugo_rola === 3) ? "Da" : "Ne"}.
          </p>
          {visible.length ? (
            <ul>
              {visible.map((o) => (
                <li key={o.id}>
                  {o.sifra} — {o.naziv}:{" "}
                  {o.write
                    ? "pregled, unos, izmena i brisanje"
                    : "samo pregled"}
                </li>
              ))}
            </ul>
          ) : (
            <p>Nema prava pristupa ugovorima.</p>
          )}
        </>
      )}
    </details>
  );
}
