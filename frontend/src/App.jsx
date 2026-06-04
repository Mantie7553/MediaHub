import { BrowserRouter, Routes, Route, Outlet } from 'react-router-dom'
import DashboardPage from './pages/dashboard/DashboardPage'
import DownloadsPage from './pages/downloads/DownloadsPage'
import Layout from './components/layout/Layout'
import LoginPage from './pages/login/LoginPage'
import ProtectedRoute from './components/layout/ProtectedRoute'
import SettingsPage from './pages/settings/SettingsPage'
import { MangaViewPage, MangaReader } from './pages/manga'
import { LightNovelViewPage, LightNovelReader } from './pages/light-novel'
import Discover from './pages/discover/Discover'
import PlayerPage from './pages/player/PlayerPage'
import AnimeViewPage from './pages/anime/AnimeViewPage'
import Library from './pages/library/Library'

function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route element={<ProtectedRoute><Outlet/></ProtectedRoute>}>
          <Route path='/watch/:type/:id' element={<PlayerPage/>} />
        </Route>
        <Route element={<ProtectedRoute><Layout><Outlet /></Layout></ProtectedRoute>}>
          <Route path="/" element={<DashboardPage/>} />
          <Route path="/downloads" element={<DownloadsPage />} />
          <Route path="/discover" element={<Discover />} />
          <Route path="/library" element={<Library/>}/>
          <Route path="/light-novels/:id" element={<LightNovelViewPage />} />
          <Route path="/light-novels/:id/volumes/:volumeId/read" element={<LightNovelReader />} />
          <Route path="/anime/:id" element={<AnimeViewPage/>} />
          <Route path="/manga/:id" element={<MangaViewPage/>}/>
          <Route path='manga/:id/chapters/:chapterId/read' element={<MangaReader/>} />
          <Route path="/settings" element={<SettingsPage/>} />
        </Route>
      </Routes>
    </BrowserRouter>
  )
}

export default App