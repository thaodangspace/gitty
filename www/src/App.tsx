import { Routes, Route } from 'react-router-dom'
import AppLayout from './components/layout/AppLayout'

function App() {
  return (
    <div className="h-screen flex flex-col">
      <Routes>
        <Route path="/*" element={<AppLayout />} />
      </Routes>
    </div>
  )
}

export default App
