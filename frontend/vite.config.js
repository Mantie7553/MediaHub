import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    react(),
    tailwindcss(),
  ],
  // in dev, forward /api to the Go backend (Caddy does this in production).
  // API_PROXY points it elsewhere, e.g. API_PROXY=http://172.16.1.50 for the server's Caddy
  server: {
    proxy: {
      '/api': process.env.API_PROXY ?? 'http://localhost:9090',
    },
  },
})
