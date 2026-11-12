import { NavLink, Outlet } from "react-router-dom";
import { usePermissions } from "../../context/usePermissions";
import "./admin.css";

export default function AdminLayout() {
  const permissions = usePermissions();
  if (permissions.isPending)
    return (
      <main id="main-content" className="admin-page">
        <p>Provera prava…</p>
      </main>
    );
  if (permissions.error)
    return (
      <main id="main-content" className="admin-page">
        <p role="alert">{permissions.error.message}</p>
        <button className="button" onClick={() => permissions.refetch()}>
          Pokušaj ponovo
        </button>
      </main>
    );
  if (!permissions.admin)
    return (
      <main id="main-content" className="admin-page">
        <p role="alert">Nemate administratorska prava.</p>
      </main>
    );
  return (
    <main id="main-content" className="admin-page">
      <h1>Administracija</h1>
      <nav className="admin-tabs" aria-label="Administracija">
        <NavLink to="/administracija/korisnici">Korisnici i prava</NavLink>
        <NavLink to="/administracija/obavestenja">Obaveštenja</NavLink>
        <NavLink to="/administracija/istorija">Istorija promena</NavLink>
        <NavLink to="/ugovori/radni-kalendar">Radni kalendar</NavLink>
      </nav>
      <Outlet />
    </main>
  );
}
