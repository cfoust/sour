import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [
    react(),
  ],
  publicDir: 'public',
  build: {
    outDir: '../pkg/server/static/site',
    emptyOutDir: true,
  },
  server: {
    port: 5173,
    proxy: {
      '/ws/': { target: 'http://localhost:1337', ws: true },
      '/api/': { target: 'http://localhost:1337' },
      '/assets/': { target: 'http://localhost:1337' },
      '/catalog.json': { target: 'http://localhost:1337' },
      '/catalog/': { target: 'http://localhost:1337' },
      '/service/': { target: 'http://localhost:1337', ws: true },
    },
  },
  worker: { format: 'es' },
})
