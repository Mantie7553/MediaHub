import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

// stamps each build with an ID, inlined as import.meta.env.VITE_BUILD_ID and published as
// /version.json, so a tab left open across a deploy can tell it's out of date and reload
function buildVersion() {
  const id = process.env.GIT_SHA || Date.now().toString(36)
  return {
    name: 'build-version',
    config: () => ({ define: { 'import.meta.env.VITE_BUILD_ID': JSON.stringify(id) } }),
    generateBundle() {
      this.emitFile({ type: 'asset', fileName: 'version.json', source: JSON.stringify({ build: id }) })
    },
  }
}

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    react(),
    tailwindcss(),
    buildVersion(),
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
