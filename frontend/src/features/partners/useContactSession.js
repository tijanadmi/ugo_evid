import { useCallback, useEffect, useRef, useState } from 'react';
import toast from 'react-hot-toast';
import { acquireContact, renewContact, releaseContact, getContact } from '../../services/apiContacts';

// One owner for the whole edit/delete session. Child form remounts or query
// refreshes must never release a lease that is still being used.
export function useContactSession(api) {
  const current = useRef(null);
  const mounted = useRef(true);
  const pending = useRef(false);
  const renewal = useRef(null);
  const [session, setSession] = useState(null);
  const [lost, setLost] = useState(false);
  const release = useCallback(async () => {
    const previous = current.current;
    current.current = null;
    setSession(null);
    setLost(false);
    if (renewal.current) await renewal.current;
    if (previous) await releaseContact(api, previous.person.id, previous.token).catch(() => {});
  }, [api]);
  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
      const previous = current.current;
      current.current = null;
      if (previous) Promise.resolve(renewal.current).then(() => releaseContact(api, previous.person.id, previous.token)).catch(() => {});
    };
  }, [api]);
  const acquire = useCallback(async (id, mode) => {
    if (pending.current) return null;
    pending.current = true;
    let token;
    try {
      const lease = await acquireContact(api, id);
      token = lease.lock_token;
      const person = await getContact(api, id);
      if (!mounted.current) { await releaseContact(api, id, token).catch(() => {}); return null; }
      const next = { person, token, mode };
      current.current = next;
      setSession(next);
      setLost(false);
      return next;
    } catch (error) {
      if (token) await releaseContact(api, id, token).catch(() => {});
      if (mounted.current) toast.error(error.message, { id: 'operation-error' });
      return null;
    } finally { pending.current = false; }
  }, [api]);
  const renew = useCallback(async () => {
    const owner = current.current;
    if (!owner) return false;
    if (renewal.current) return renewal.current;
    const task = (async () => {
      try {
        await renewContact(api, owner.person.id, owner.token);
        return current.current === owner;
      } catch (error) {
        if (mounted.current && current.current === owner) {
          setLost(true);
          toast.error(`${error.message} Ponovo učitajte podatke pre nastavka.`, { id: 'contact-lease-lost' });
        }
        return false;
      }
    })();
    renewal.current = task;
    try { return await task; } finally { renewal.current = null; }
  }, [api]);
  useEffect(() => {
    if (!session || lost) return;
    const timer = setInterval(renew, 30000);
    const focus = () => { if (document.visibilityState === 'visible') renew(); };
    window.addEventListener('focus', focus);
    document.addEventListener('visibilitychange', focus);
    return () => { clearInterval(timer); window.removeEventListener('focus', focus); document.removeEventListener('visibilitychange', focus); };
  }, [session, lost, renew]);
  function committed() { current.current = null; setSession(null); setLost(false); }
  return { session, lost, acquire, release, renew, committed };
}
