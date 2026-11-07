import Reminders from "../features/contracts/Reminders";
import { Link } from "react-router-dom";
import Icon from "../ui/Icon";

export default function Portal() {
  return (
    <main className="portal-main" id="main-content">
      <section className="portal-hero">
        <div>
          <span className="eyebrow">POSLOVNI PORTAL</span>
          <h1>Dobro došli</h1>
          <p>Izaberite aplikaciju za nastavak rada.</p>
        </div>
        <div className="hero-symbol" aria-hidden="true">
          <Icon name="grid" width="120" height="120" />
        </div>
      </section>
      <div className="section-heading">
        <div>
          <h2>Aplikacije</h2>
          <p className="muted">Nabavke, ugovori i kolaudacioni elaborati.</p>
        </div>
        <span className="subtle-badge">1 dostupna · 2 u pripremi</span>
      </div>
      <div className="application-grid">
        <article className="application-card is-planned">
          <span className="app-icon">
            <Icon name="open" width="28" height="28" />
          </span>
          <div className="app-card-copy">
            <h3>Planiranje i nabavke</h3>
            <p>
              Planovi javnih nabavki, zahtevi za pokretanje nabavke, interne
              narudžbenice i konkursna dokumentacija.
            </p>
            <span className="subtle-badge">U pripremi</span>
          </div>
        </article>
        <Link className="application-card" to="/ugovori/otvoreni">
          <span className="app-icon">
            <Icon name="file" width="28" height="28" />
          </span>
          <span className="app-card-copy">
            <h3>Evidencija ugovora</h3>
            <p>
              Pregled ugovora, praćenje realizacije, dokumentacija i podsetnici.
            </p>
            <span className="app-card-action">
              Otvori aplikaciju <Icon name="arrow" />
            </span>
          </span>
        </Link>
        <article className="application-card is-planned">
          <span className="app-icon">
            <Icon name="edit" width="28" height="28" />
          </span>
          <div className="app-card-copy">
            <h3>Kolaudacija</h3>
            <p>
              Izrada i praćenje kolaudacionih elaborata uz korišćenje podataka o
              osnovnim sredstvima iz SAP-a.
            </p>
            <span className="subtle-badge">U pripremi</span>
          </div>
        </article>
      </div>
      <Reminders />
    </main>
  );
}
