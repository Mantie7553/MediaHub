import { BrowserRouter, Routes, Route, Outlet } from 'react-router-dom'
import DashboardPage from './pages/dashboard/DashboardPage'
import DownloadsPage from './pages/downloads/DownloadsPage'
import Layout from './components/layout/Layout'
import LoginPage from './pages/login/LoginPage'
import ProtectedRoute from './components/layout/ProtectedRoute'
import SettingsPage from './pages/settings/SettingsPage'
import Discover from './pages/discover/Discover'
import Library from './pages/library/Library'
import { 
  AlbumDetailsPage,
  AnimeDetailsPage, LightNovelDetailsPage, 
  MangaDetailsPage, MovieDetailsPage 
} from './pages/details'
import { LightNovelReader, MangaReader, PlayerPage } from './pages/viewer'

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
          <Route path="/settings" element={<SettingsPage/>} />
          <Route path="/anime/:id" element={<AnimeDetailsPage/>} />
          <Route path='/albums/:id' element={<AlbumDetailsPage/>}/>
          <Route path="/light-novels/:id" element={<LightNovelDetailsPage />} />
          <Route path="/manga/:id" element={<MangaDetailsPage/>}/>
          <Route path='/movies/:id' element={<MovieDetailsPage/>}/>
          <Route path="/light-novels/:id/volumes/:volumeId/read" element={<LightNovelReader/>} />
          <Route path='manga/:id/chapters/:chapterId/read' element={<MangaReader/>} />
        </Route>
      </Routes>
    </BrowserRouter>
  )
}

export default App