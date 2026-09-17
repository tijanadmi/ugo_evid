import React from 'react';
import { createRoot } from 'react-dom/client';
import { BrowserRouter, Link, Navigate, Outlet, Route, Routes } from 'react-router-dom';
import { AuthProvider, useAuth } from './auth';
import { PortalLayout, WorkspaceLayout } from './components/Layout';
import Login from './pages/Login';
import Portal from './pages/Portal';
import Contracts from './pages/Contracts';
import './styles.css';

function ProtectedRoute() { const { user } = useAuth(); return user ? <Outlet/> : <Navigate to="/login" replace/>; }
function NotFound() { return <main className="not-found"><h1>Stranica nije pronađena</h1><Link className="button primary" to="/">Povratak na portal</Link></main>; }

createRoot(document.getElementById('root')).render(<React.StrictMode><BrowserRouter><AuthProvider>
  <a className="skip-link" href="#main-content">Pređi na sadržaj</a>
  <Routes>
    <Route path="/login" element={<Login/>}/>
    <Route element={<ProtectedRoute/>}>
      <Route element={<PortalLayout/>}><Route index element={<Portal/>}/></Route>
      <Route path="/ugovori" element={<WorkspaceLayout/>}>
        <Route index element={<Navigate to="otvoreni" replace/>}/>
        <Route path="otvoreni" element={<Contracts key="otvoreni" status="otvoreni"/>}/>
        <Route path="zatvoreni" element={<Contracts key="zatvoreni" status="zatvoreni"/>}/>
      </Route>
    </Route>
    <Route path="*" element={<NotFound/>}/>
  </Routes>
</AuthProvider></BrowserRouter></React.StrictMode>);
