import { BrowserRouter, Routes, Route, Outlet } from 'react-router-dom'
import DashboardPage from './pages/dashboard/DashboardPage'
import DownloadsPage from './pages/downloads/DownloadsPage'
import Layout from './components/layout/Layout'
import LoginPage from './pages/login/LoginPage'
import ProtectedRoute from './components/layout/ProtectedRoute'
import SettingsPage from './pages/settings/SettingsPage'

function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route element={<ProtectedRoute><Layout><Outlet /></Layout></ProtectedRoute>}>
          <Route path="/" element={<DashboardPage/>} />
          <Route path="/downloads" element={<DownloadsPage />} />
          <Route path="/media" element={<div>Media</div>} />
          <Route path="/settings" element={<SettingsPage/>} />
        </Route>
      </Routes>
    </BrowserRouter>
  )
}

export default App