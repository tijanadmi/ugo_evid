import { createContext } from 'react';

// Keep one context identity across provider and consumer module refreshes.
export const AuthContext = import.meta.hot?.data.authContext ?? createContext(null);
if (import.meta.hot) import.meta.hot.data.authContext = AuthContext;
