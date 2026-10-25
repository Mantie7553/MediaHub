import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    react(),
    tailwindcss(),
  ],
  // the ASS renderer loads its worker and wasm relative to its own files, which pre-bundling breaks
  optimizeDeps: {
    exclude: ['jassub'],
  },
  // in dev, forward /api to the Go backend (Caddy does this in production).
  // API_PROXY points it elsewhere, e.g. API_PROXY=http://172.16.1.50 for the server's Caddy
  server: {
    proxy: {
      '/api': process.env.API_PROXY ?? 'http://localhost:9090',
    },
  },
})
