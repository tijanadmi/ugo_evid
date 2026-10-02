import { useEffect } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { useAuth } from '../context/AuthContext';

const eventKey = 'ugo-evid-data-changed';
export function invalidateBusinessData(client) {
  client.invalidateQueries({ queryKey: ['contract-preparation'], refetchType: 'none' });
  return Promise.all(['partners', 'contracts', 'contract', 'unregistered-contracts'].map(key => client.invalidateQueries({ queryKey: [key] })));
}
export function announceBusinessChange(client) {
  invalidateBusinessData(client);
  // No personal data or tokens are written to shared browser storage.
  try { localStorage.setItem(eventKey, JSON.stringify({ nonce: `${Date.now()}-${Math.random()}` })); } catch { /* Focus/polling still refresh other sessions. */ }
}
export default function DataSync() {
  const client = useQueryClient();
  const { user } = useAuth();
  useEffect(() => {
    if (!user) return;
    const changed = event => { if (event.key === eventKey) invalidateBusinessData(client); };
    window.addEventListener('storage', changed);
    return () => window.removeEventListener('storage', changed);
  }, [client, user?.username]);
  return null;
}
