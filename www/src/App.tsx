import { Routes, Route, Navigate } from 'react-router-dom'
import { useAtomValue } from 'jotai'
import { isAuthenticatedAtom } from './store/atoms/auth-atoms'
import AppLayout from './components/layout/AppLayout'
import LandingPage from './components/layout/LandingPage'
import ChooseRepositoryDialog from './components/repository/ChooseRepositoryDialog'
import CommitDetailsPage from './pages/CommitDetailsPage'
import LoginPage from './pages/LoginPage'

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const isAuthenticated = useAtomValue(isAuthenticatedAtom)
  return isAuthenticated ? <>{children}</> : <Navigate to="/login" replace />
}

function App() {
  return (
    <div className="h-screen flex flex-col">
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route path="/" element={<ProtectedRoute><LandingPage /></ProtectedRoute>} />
        <Route path="/repo/:repoId" element={<ProtectedRoute><AppLayout /></ProtectedRoute>} />
        <Route path="/repo/:repoId/commit/:commitHash" element={<ProtectedRoute><CommitDetailsPage /></ProtectedRoute>} />
      </Routes>
      <ChooseRepositoryDialog />
    </div>
  )
}

export default App
