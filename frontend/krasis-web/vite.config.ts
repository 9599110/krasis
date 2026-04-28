import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// Set KRASIS_BACKEND_URL to override (e.g. http://localhost:9091)
const backend = process.env.KRASIS_BACKEND_URL || 'http://localhost:18081'

// Skip proxy for browser navigations (Accept: text/html)
// so Vite serves index.html for SPA client-side routing.
const spaBypass = (req: any, _proxy: any) => {
  const accept = req.headers.accept || ''
  if (accept.includes('text/html')) return '/'
}

// https://vite.dev/config/
export default defineConfig({
  plugins: [vue()],
  server: {
    proxy: {
      // Unified admin proxy — matches all /admin/* requests.
      // API calls (Accept: application/json) are proxied to backend.
      // Browser refresh (Accept: text/html) rewrites to / for SPA.
      '^/admin': { target: backend, bypass: spaBypass },

      // App APIs
      '/auth': { target: backend, bypass: spaBypass },
      '/user': { target: backend, bypass: spaBypass },
      '/folders': { target: backend, bypass: spaBypass },
      '/notes': { target: backend, bypass: spaBypass },
      '/search': { target: backend, bypass: spaBypass },
      '/ai': { target: backend, bypass: spaBypass },
      '/files': { target: backend, bypass: spaBypass },
      '/share': { target: backend, bypass: spaBypass },
      '/health': { target: backend, bypass: spaBypass },
    },
  },
})
