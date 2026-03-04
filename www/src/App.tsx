import { Routes, Route } from 'react-router-dom'
import AppLayout from './components/layout/AppLayout'
import LandingPage from './components/layout/LandingPage'
import ChooseRepositoryDialog from './components/repository/ChooseRepositoryDialog'

function App() {
  return (
    <div className="h-screen flex flex-col">
      <Routes>
        <Route path="/" element={<LandingPage />} />
        <Route path="/repo/:repoId" element={<AppLayout />} />
      </Routes>
      <ChooseRepositoryDialog />
    </div>
  )
}

export default App
