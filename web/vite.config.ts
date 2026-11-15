import tailwindcss from '@tailwindcss/vite'
import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

// The Go server embeds web/dist. In development, run the Go server on :8789
// and `pnpm dev`; Vite proxies API calls to it.
export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    proxy: {
      '/api': 'http://localhost:8789',
    },
  },
})
