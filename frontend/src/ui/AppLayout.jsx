import { useState } from 'react';
import { Link, NavLink, Outlet } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import Icon from './Icon';

export function Brand() {
  return <Link to="/" className="brand" aria-label="Portal aplikacija"><span className="brand-mark"><Icon name="file"/></span><span>UGOVORI<small>Informacioni sistem</small></span></Link>;
}
export function Header({ onMenu }) {
  const { user, logout } = useAuth();
  return <header className="header">
    <div className="header-start">{onMenu && <button className="icon-button mobile-menu" onClick={onMenu} aria-label="Otvori meni"><Icon name="menu"/></button>}<span>Poslovne aplikacije <span className="header-divider">/</span> <strong>Ugovori</strong></span></div>
    <div className="user-menu"><span className="avatar">{user.username.slice(0, 2).toUpperCase()}</span><span className="username">{user.username}<small>AD nalog</small></span><button className="icon-button" onClick={logout} title="Odjavi se" aria-label="Odjavi se"><Icon name="logout"/></button></div>
  </header>;
}
export function PortalLayout() {
  return <div className="portal-layout"><div className="portal-top"><Brand/><Header/></div><Outlet/></div>;
}
export function WorkspaceLayout() {
  const [menuOpen, setMenuOpen] = useState(false);
  return <div className="workspace">
    {menuOpen && <button className="nav-backdrop" aria-label="Zatvori meni" onClick={() => setMenuOpen(false)}/>}
    <aside className={`sidebar ${menuOpen ? 'is-open' : ''}`}>
      <Brand/>
      <div className="sidebar-label">EVIDENCIJA UGOVORA</div>
      <nav aria-label="Pregledi ugovora">
        <NavLink to="/ugovori/otvoreni" onClick={() => setMenuOpen(false)}><Icon name="open"/>Otvoreni ugovori</NavLink>
        <NavLink to="/ugovori/zatvoreni" onClick={() => setMenuOpen(false)}><Icon name="closed"/>Zatvoreni ugovori</NavLink>
      </nav>
      <div className="sidebar-bottom"><Link to="/" onClick={() => setMenuOpen(false)}><Icon name="grid"/>Portal aplikacija</Link><small>Evidencija ugovora · 1.0</small></div>
    </aside>
    <div className="workspace-body"><Header onMenu={() => setMenuOpen(!menuOpen)}/><main className="workspace-main" id="main-content"><Outlet/></main></div>
  </div>;
}
