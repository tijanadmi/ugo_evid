import { useState } from 'react';
import { Navigate, useNavigate } from 'react-router-dom';
import { useAuth } from '../auth';
import { Brand } from '../components/Layout';
import Icon from '../components/Icon';

export default function Login() {
  const { user, login } = useAuth();
  const navigate = useNavigate();
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [visible, setVisible] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  if (user) return <Navigate to="/" replace/>;
  async function submit(event) {
    event.preventDefault();
    if (busy) return;
    setBusy(true); setError('');
    try { await login(username.trim(), password); navigate('/', { replace: true }); }
    catch (error) { setError(error.status === 401 || error.status === 403 ? 'Korisničko ime ili lozinka nisu ispravni, ili nalog nema pristup aplikaciji.' : error.message); }
    finally { setBusy(false); setPassword(''); }
  }
  return <main className="login-page">
    <section className="login-intro"><Brand/><div className="login-copy"><span className="eyebrow">POSLOVNE APLIKACIJE</span><h1>Vaši ugovori.<br/>Na jednom mestu.</h1><p>Jedinstven pristup pregledu ugovora,<br className="desktop-break"/> dobavljača i odgovornih lica.</p><div className="intro-note"><Icon name="file"/><span>Evidencija ugovora</span></div></div><small>Informacioni sistem · Interna upotreba</small></section>
    <section className="login-form-section"><div className="login-card"><span className="section-icon"><Icon name="lock"/></span><h2>Dobro došli</h2><p className="muted">Prijavite se svojim AD nalogom.</p>
      <form onSubmit={submit}>
        <label htmlFor="username">Korisničko ime</label><input id="username" autoComplete="username" value={username} onChange={e => setUsername(e.target.value)} required disabled={busy} autoFocus placeholder="ime.prezime"/>
        <label htmlFor="password">Lozinka</label><div className="password-field"><input id="password" type={visible ? 'text' : 'password'} autoComplete="current-password" value={password} onChange={e => setPassword(e.target.value)} required disabled={busy}/><button type="button" className="icon-button" aria-label={visible ? 'Sakrij lozinku' : 'Prikaži lozinku'} aria-pressed={visible} onClick={() => setVisible(!visible)}><Icon name="eye"/></button></div>
        {error && <div className="alert error" role="alert">{error}</div>}
        <button className="button primary login-submit" disabled={busy || !username.trim()}>{busy ? 'Prijavljivanje…' : 'Prijavi se'}{!busy && <Icon name="arrow"/>}</button>
      </form><p className="login-help">Za pristup koristite korisničko ime i lozinku<br/>sa poslovne mreže.</p>
    </div></section>
  </main>;
}
