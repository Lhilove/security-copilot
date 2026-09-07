import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import type { ReactNode } from 'react'
import { useAuth } from './context/AuthContext'
import LandingPage from './pages/LandingPage'
import CallbackPage from './pages/CallbackPage'
import DashboardPage from './pages/DashboardPage'
import RepositoryPage from './pages/RepositoryPage'
import FindingPage from './pages/FindingPage'
import RemediationsPage from './pages/RemediationsPage'

function ProtectedRoute({ children }: { children: ReactNode }) {
  const { isAuthenticated } = useAuth()
  return isAuthenticated ? <>{children}</> : <Navigate to="/" replace />
}

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<LandingPage />} />
        <Route path="/callback" element={<CallbackPage />} />
        <Route path="/dashboard" element={<ProtectedRoute><DashboardPage /></ProtectedRoute>} />
        <Route path="/repositories/:id" element={<ProtectedRoute><RepositoryPage /></ProtectedRoute>} />
        <Route path="/findings/:id" element={<ProtectedRoute><FindingPage /></ProtectedRoute>} />
        <Route path="/remediations" element={<ProtectedRoute><RemediationsPage /></ProtectedRoute>} />
      </Routes>
    </BrowserRouter>
  )
}