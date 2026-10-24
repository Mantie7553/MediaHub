import { BrowserRouter, Routes, Route, Outlet } from 'react-router-dom'
import DashboardPage from './pages/dashboard/DashboardPage'
import DownloadsPage from './pages/downloads/DownloadsPage'
import Layout from './components/layout/Layout'
import LoginPage from './pages/login/LoginPage'
import ProtectedRoute from './components/layout/ProtectedRoute'
import SettingsPage from './pages/settings/SettingsPage'
import Discover from './pages/discover/Discover'
import { 
  AlbumDetailsPage, CollectionPage,
  AnimeDetailsPage, LightNovelDetailsPage, 
  MangaDetailsPage, MovieDetailsPage 
} from './pages/details'
import { LightNovelReader, MangaReader, PlayerPage } from './pages/viewer'
import UserManagementPage from './pages/Management/UserManagementPage'
import InsightsPage from './pages/insights/InsightsPage'
import AudioPlayer from './pages/viewer/AudioPlayer'
import Toaster from './components/Toaster'
import NotFound from './components/states/NotFound'
import ErrorBoundary, { PageBoundary } from './components/states/ErrorBoundary'

function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route element={<ProtectedRoute><PageBoundary><Outlet/></PageBoundary></ProtectedRoute>}>
          {/* viewers fill the screen, without the sidebar or tab bar */}
          <Route path='/watch/:type/:id' element={<PlayerPage/>} />
          <Route path="/light-novels/:id/volumes/:volumeId/read" element={<LightNovelReader/>} />
          <Route path='/manga/:id/chapters/:chapterId/read' element={<MangaReader/>} />
        </Route>
        <Route element={<ProtectedRoute><Layout><PageBoundary><Outlet /></PageBoundary></Layout></ProtectedRoute>}>
          <Route path="/" element={<DashboardPage/>} />
          <Route path="/downloads" element={<DownloadsPage />} />
          <Route path='/users' element={<UserManagementPage/>} />
          <Route path='/insights' element={<InsightsPage/>} />
          <Route path="/discover" element={<Discover />} />
          <Route path="/settings" element={<SettingsPage/>} />
          <Route path="/anime/:id" element={<AnimeDetailsPage/>} />
          <Route path='/albums/:id' element={<AlbumDetailsPage/>}/>
          <Route path='/playlists/:id' element={<CollectionPage/>}/>
          <Route path='/collections/:id' element={<CollectionPage/>}/>
          <Route path="/light-novels/:id" element={<LightNovelDetailsPage />} />
          <Route path="/manga/:id" element={<MangaDetailsPage/>}/>
          <Route path='/movies/:id' element={<MovieDetailsPage/>}/>
          <Route path="*" element={<NotFound/>} />
        </Route>
      </Routes>
      <ErrorBoundary><AudioPlayer/></ErrorBoundary>
      <Toaster/>
    </BrowserRouter>
  )
}

export default App