import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

const apiPrefixes = [
  '/admin/users',
  '/admin/stats',
  '/admin/ai/',
  '/admin/config',
  '/admin/groups',
  '/admin/shares',
  '/admin/logs',
  '/admin/auth/',
  '/auth/',
  '/user/',
  '/notes/',
  '/files/',
  '/search',
  '/ws/',
]

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5180,
    proxy: {
      '/': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        bypass: (req) => {
          const isApi = apiPrefixes.some((p) => req.url?.startsWith(p))
          const isStatic = req.url?.match(/\.(js|css|png|jpg|svg|ico|woff|ttf|eot|map)$/)
          if (!isApi && !isStatic) {
            return '/index.html'
          }
        },
      },
    },
  },
})
