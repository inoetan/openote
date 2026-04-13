import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { useAuthStore } from './store/auth'
import LoginPage from './pages/LoginPage'
import DashboardPage from './pages/DashboardPage'
import JobBuilderPage from './pages/JobBuilderPage'
import JobSchedulePage from './pages/JobSchedulePage'
import ExecutionsPage from './pages/ExecutionsPage'

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const token = useAuthStore((s) => s.token)
  if (!token) return <Navigate to="/login" replace />
  return <>{children}</>
}

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route path="/" element={<Navigate to="/dashboard" replace />} />
        <Route
          path="/dashboard"
          element={
            <ProtectedRoute>
              <DashboardPage />
            </ProtectedRoute>
          }
        />
        <Route
          path="/projects/:pid/jobs/new"
          element={
            <ProtectedRoute>
              <JobBuilderPage />
            </ProtectedRoute>
          }
        />
        <Route
          path="/projects/:pid/jobs/:jid/edit"
          element={
            <ProtectedRoute>
              <JobBuilderPage />
            </ProtectedRoute>
          }
        />
        <Route
          path="/projects/:pid/jobs/:jid/schedule"
          element={
            <ProtectedRoute>
              <JobSchedulePage />
            </ProtectedRoute>
          }
        />
        <Route
          path="/projects/:pid/executions"
          element={
            <ProtectedRoute>
              <ExecutionsPage />
            </ProtectedRoute>
          }
        />
      </Routes>
    </BrowserRouter>
  )
}
