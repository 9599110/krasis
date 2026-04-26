import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// Set KRASIS_BACKEND_URL to override (e.g. http://localhost:9091)
const backend = process.env.KRASIS_BACKEND_URL || 'http://localhost:18081'

// https://vite.dev/config/
export default defineConfig({
  plugins: [vue()],
  server: {
    proxy: {
      // App APIs
      '/auth': backend,
      '/user': backend,
      '/folders': backend,
      '/notes': backend,
      '/search': backend,
      '/ai': backend,
      '/files': backend,
      '/share': backend,
      '/health': backend,

      // Admin APIs only. Do NOT proxy '/admin' itself, otherwise SPA routing breaks.
      '/admin/users': backend,
      '/admin/stats': backend,
      '/admin/shares': backend,
      // Note: use trailing slash so '/admin/ai-models' (SPA route) is NOT proxied.
      '/admin/ai/': backend,
      '/admin/config': backend,
      '/admin/auth': backend,
      '/admin/groups': backend,
      '/admin/logs': backend,
    },
  },
})
