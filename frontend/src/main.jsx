import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'
import App from './App.jsx'
import { installModalGestures } from './utils/modalGestures'

// every dialog closes on a click outside it, and on phones on a swipe down
installModalGestures()

createRoot(document.getElementById('root')).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
