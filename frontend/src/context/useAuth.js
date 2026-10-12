import { useContext } from 'react';
import { AuthContext } from './authContextStore';

export function useAuth() {
  const value = useContext(AuthContext);
  if (value === null) throw new Error('useAuth mora biti unutar AuthProvider komponente.');
  return value;
}
