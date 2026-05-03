import { BrowserRouter, Routes, Route, Outlet } from 'react-router-dom'
import Layout from './components/layout/Layout'
import ProtectedRoute from './components/layout/ProtectedRoute'
import DownloadsPage from './pages/downloads/DownloadsPage'
import LoginPage from './pages/login/LoginPage'

function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route element={<ProtectedRoute><Layout><Outlet /></Layout></ProtectedRoute>}>
          <Route path="/" element={<div>Dashboard</div>} />
          <Route path="/downloads" element={<DownloadsPage />} />
          <Route path="/media" element={<div>Media</div>} />
          <Route path="/settings" element={<div>Settings</div>} />
        </Route>
      </Routes>
    </BrowserRouter>
  )
}

export default App