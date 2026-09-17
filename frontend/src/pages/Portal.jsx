import { Link } from 'react-router-dom';
import Icon from '../components/Icon';

export default function Portal() {
  return <main className="portal-main" id="main-content">
    <section className="portal-hero"><div><span className="eyebrow">PORTAL APLIKACIJA</span><h1>Dobro došli u<br/>vaš radni prostor.</h1><p>Izaberite aplikaciju i nastavite sa radom.</p></div><div className="hero-symbol" aria-hidden="true"><Icon name="file" width="120" height="120"/></div></section>
    <div className="section-heading"><div><h2>Aplikacije</h2><p className="muted">Vaše poslovne evidencije i pregledi.</p></div><span className="subtle-badge">1 aplikacija</span></div>
    <div className="application-grid"><Link className="application-card" to="/ugovori/otvoreni"><span className="app-icon"><Icon name="file" width="28" height="28"/></span><span className="app-card-copy"><h3>Evidencija ugovora</h3><p>Otvoreni i zatvoreni ugovori, dobavljači i kontakt podaci.</p><span className="app-card-action">Otvori aplikaciju <Icon name="arrow"/></span></span></Link></div>
  </main>;
}
