import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom';
import { QueryClientProvider } from '@tanstack/react-query';
import { queryClient } from './services/queryClient';
import { AuthProvider } from './context/AuthContext';
import { PortalLayout, WorkspaceLayout } from './ui/AppLayout';
import ProtectedRoute from './ui/ProtectedRoute';
import NotFound from './pages/PageNotFound';
import Login from './pages/Login';
import Portal from './pages/Portal';
import Contracts from './pages/Contracts';
export default function App() { return (<QueryClientProvider client={queryClient}><BrowserRouter><AuthProvider>
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
</AuthProvider></BrowserRouter></QueryClientProvider>); }
