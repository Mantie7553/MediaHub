import { BrowserRouter, Routes, Route } from 'react-router-dom'
import Layout from './components/layout/Layout'
import DownloadsPage from './pages/downloads/DownloadsPage'

function App() {
  return (
    <BrowserRouter>
      <Layout>
        <Routes>
          <Route path="/" element={<div>Dashboard</div>} />
          <Route path="/downloads" element={<DownloadsPage/>} />
          <Route path="/media" element={<div>Media</div>} />
          <Route path="/settings" element={<div>Settings</div>} />
        </Routes>
      </Layout>
    </BrowserRouter>
  )
}

export default App