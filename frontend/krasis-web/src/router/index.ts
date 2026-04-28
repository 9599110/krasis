import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      name: 'splash',
      component: () => import('../views/SplashView.vue'),
    },
    {
      path: '/login',
      name: 'login',
      component: () => import('../views/LoginView.vue'),
      meta: { guest: true },
    },
    {
      path: '/register',
      name: 'register',
      component: () => import('../views/RegisterView.vue'),
      meta: { guest: true },
    },
    {
      path: '/share/:token',
      name: 'share',
      component: () => import('../views/ShareView.vue'),
      meta: { guest: true },
    },
    {
      path: '/admin',
      component: () => import('../views/admin/AdminLayout.vue'),
      meta: { requiresAuth: true },
      children: [
        {
          path: '',
          name: 'admin-dashboard',
          component: () => import('../views/admin/DashboardView.vue'),
        },
        {
          path: 'users',
          name: 'admin-users',
          component: () => import('../views/admin/UsersView.vue'),
        },
        {
          path: 'shares',
          name: 'admin-shares',
          component: () => import('../views/admin/SharesView.vue'),
        },
        {
          path: 'ai-models',
          name: 'admin-ai-models',
          component: () => import('../views/admin/AIModelsView.vue'),
        },
        {
          path: 'ai-config',
          name: 'admin-ai-config',
          component: () => import('../views/admin/AIConfigView.vue'),
        },
        {
          path: 'system-config',
          name: 'admin-system-config',
          component: () => import('../views/admin/SystemConfigView.vue'),
        },
        {
          path: 'oauth-config',
          name: 'admin-oauth-config',
          component: () => import('../views/admin/OAuthConfigView.vue'),
        },
        {
          path: 'groups',
          name: 'admin-groups',
          component: () => import('../views/admin/GroupsView.vue'),
        },
        {
          path: 'logs',
          name: 'admin-logs',
          component: () => import('../views/admin/LogsView.vue'),
        },
      ],
    },
    {
      path: '/app',
      component: () => import('../views/LayoutView.vue'),
      meta: { requiresAuth: true },
      children: [
        {
          path: 'notes',
          name: 'notes',
          component: () => import('../views/NoteListView.vue'),
        },
        {
          path: 'notes/:id',
          name: 'note-edit',
          component: () => import('../views/NoteEditorView.vue'),
        },
        {
          path: 'notes/:id/versions',
          name: 'note-versions',
          component: () => import('../views/NoteVersionsView.vue'),
        },
        {
          path: 'ai',
          name: 'ai-chat',
          component: () => import('../views/AIChatView.vue'),
        },
        {
          path: 'search',
          name: 'search',
          component: () => import('../views/SearchView.vue'),
        },
        {
          path: 'folders',
          name: 'folders',
          component: () => import('../views/FoldersView.vue'),
        },
        {
          path: 'profile',
          name: 'profile',
          component: () => import('../views/ProfileView.vue'),
        },
        {
          path: '',
          redirect: { name: 'notes' },
        },
      ],
    },
  ],
})

router.beforeEach(async (to) => {
  const token = localStorage.getItem('auth_token')
  const authStore = useAuthStore()

  if (to.meta.requiresAuth && !token) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }

  // If has token but no user loaded, validate token via /auth/me
  if (to.meta.requiresAuth && token) {
    try {
      await authStore.me()
    } catch (e: any) {
      if (e?.response?.status === 401) {
        authStore.$patch({ token: null, user: null })
        return { name: 'login', query: { redirect: to.fullPath } }
      }
      // Network/server error: let navigation proceed, don't block the user
      console.warn('Auth check failed:', e)
    }
  }

  if (to.meta.guest && token && to.name !== 'share') {
    return { name: 'notes' }
  }
})

router.afterEach((to) => {
  if (to.path.startsWith('/admin')) {
    document.body.classList.add('admin-mode')
  } else {
    document.body.classList.remove('admin-mode')
  }
})

router.isReady().then(() => {
  if (router.currentRoute.value.path.startsWith('/admin')) {
    document.body.classList.add('admin-mode')
  }
})

export default router
