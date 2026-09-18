import { useQueryClient } from '@tanstack/react-query';
import { createContext, useCallback, useContext, useRef, useState } from 'react';
import { ApiError, request } from '../services/apiClient';

const AuthContext = createContext(null);
const storageKey = 'ugo-evid-session';

function readSession() {
  try {
    const value = JSON.parse(sessionStorage.getItem(storageKey));
    if (value?.access_token && value?.refresh_token && value?.user?.username &&
        Date.parse(value.refresh_token_expires_at) > Date.now()) return value;
    sessionStorage.removeItem(storageKey);
  } catch { /* A disabled/unavailable browser store falls back to memory. */ }
  return null;
}

export function AuthProvider({ children }) {
  const queryClient = useQueryClient();
  const [session, setSession] = useState(readSession);
  const current = useRef(session);
  const refreshPending = useRef(null);
  const save = useCallback((value) => {
    if (!value || current.current?.user?.username !== value.user?.username) queryClient.clear();
    current.current = value;
    setSession(value);
    try {
      if (value) sessionStorage.setItem(storageKey, JSON.stringify(value));
      else sessionStorage.removeItem(storageKey);
    } catch { /* Session still works in this tab without persistence. */ }
  }, [queryClient]);
  const logout = useCallback(() => { save(null); }, [save]);
  const login = useCallback(async (username, password) => {
    const data = await request('/users/login', { method: 'POST', body: { username, password } });
    if (!data.access_token || !data.refresh_token || !data.user?.username ||
        !Number.isFinite(Date.parse(data.access_token_expires_at)) ||
        !Number.isFinite(Date.parse(data.refresh_token_expires_at))) {
      throw new ApiError('Odgovor prijave nije ispravan.', 500);
    }
    save(data);
  }, [save]);

  const refresh = useCallback(async () => {
    if (refreshPending.current) return refreshPending.current;
    const original = current.current;
    if (!original) throw new ApiError('Prijavite se da nastavite.', 401);
    const pending = (async () => {
      try {
        if (Date.parse(original.refresh_token_expires_at) <= Date.now()) throw new ApiError('Sesija je istekla.', 401);
        const data = await request('/tokens/renew_access', {
          method: 'POST', body: { refresh_token: original.refresh_token },
        });
        if (!data.access_token || !Number.isFinite(Date.parse(data.access_token_expires_at))) {
          throw new ApiError('Obnova sesije nije uspela.', 401);
        }
        // Never restore a session after logout or a different login.
        if (current.current !== original) throw new ApiError('Sesija je promenjena.', 401);
        const updated = { ...original, ...data };
        save(updated);
        return updated;
      } catch (error) {
        if ((error.status === 401 || error.status === 403) && current.current === original) save(null);
        throw error;
      }
    })();
    refreshPending.current = pending;
    try { return await pending; }
    finally { if (refreshPending.current === pending) refreshPending.current = null; }
  }, [save]);

  const api = useCallback(async (path, options = {}) => {
    let active = current.current;
    if (!active) throw new ApiError('Prijavite se da nastavite.', 401);
    if (Date.parse(active.access_token_expires_at) <= Date.now() + 5000) active = await refresh();
    try { return await request(path, { ...options, token: active.access_token }); }
    catch (error) {
      if (error.status !== 401) throw error;
      if (!current.current) throw error;
      const updated = current.current.access_token !== active.access_token ? current.current : await refresh();
      try { return await request(path, { ...options, token: updated.access_token }); }
      catch (retryError) {
        if (retryError.status === 401 && current.current === updated) save(null);
        throw retryError;
      }
    }
  }, [refresh, save]);

  return <AuthContext.Provider value={{ user: session?.user, login, logout, api }}>{children}</AuthContext.Provider>;
}

export function useAuth() { return useContext(AuthContext); }
