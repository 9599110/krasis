import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// Set KRASIS_BACKEND_URL to override (e.g. http://localhost:9091)
const backend = process.env.KRASIS_BACKEND_URL || 'http://localhost:9091'

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
      '/keys': { target: backend, bypass: spaBypass },
      '/share': { target: backend, bypass: spaBypass },
      '/health': { target: backend, bypass: spaBypass },
    },
    // 加速开发时页面热更新
    watch: {
      usePolling: false,
      ignored: ['**/node_modules/**', '**/dist/**'],
    },
  },
  // 预构建依赖缓存加速
  optimizeDeps: {
    include: [
      'vue',
      'vue-router',
      'pinia',
      'axios',
      'tdesign-vue-next',
      'tdesign-icons-vue-next',
      '@tiptap/vue-3',
      '@tiptap/starter-kit',
      'tiptap-markdown',
      'marked',
    ],
  },
  build: {
    // 生产构建时启用 CSS 代码分割，提升首屏加载速度
    cssCodeSplit: true,
    rollupOptions: {
      output: {
        manualChunks(id: string) {
          if (id.includes('node_modules/vue') || id.includes('node_modules/@vue')) return 'vendor-vue'
          if (id.includes('node_modules/tdesign')) return 'vendor-tdesign'
          if (id.includes('node_modules/@tiptap') || id.includes('node_modules/tiptap-markdown')) return 'vendor-tiptap'
        },
      },
    },
  },
})
