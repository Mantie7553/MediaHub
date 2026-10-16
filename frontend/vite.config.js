import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import daisyui from 'daisyui'

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    react(),
    tailwindcss(),
  ],
  // in dev, forward /api to the Go backend (Caddy does this in production)
  server: {
    proxy: {
      '/api': 'http://localhost:9090',
    },
  },
})
