import { ref, computed } from 'vue'
import { defineStore } from 'pinia'
import apiClient from '../api/client'
import type { ApiResponse, User, LoginRequest, RegisterRequest, AuthResponse, Session } from '../api/types'

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string | null>(localStorage.getItem('auth_token'))
  const user = ref<User | null>(
    (() => {
      const stored = localStorage.getItem('auth_user')
      return stored ? JSON.parse(stored) : null
    })(),
  )
  const loading = ref(false)

  const isAuthenticated = computed(() => !!token.value)

  async function login(req: LoginRequest) {
    loading.value = true
    try {
      const res = await apiClient.post<ApiResponse<AuthResponse>>('/auth/login', req)
      const data = res.data.data
      token.value = data.token
      user.value = data.user
      localStorage.setItem('auth_token', data.token)
      localStorage.setItem('auth_user', JSON.stringify(data.user))
      return data
    } finally {
      loading.value = false
    }
  }

  async function register(req: RegisterRequest) {
    loading.value = true
    try {
      const res = await apiClient.post<ApiResponse<AuthResponse>>('/auth/register', req)
      const data = res.data.data
      token.value = data.token
      user.value = data.user
      localStorage.setItem('auth_token', data.token)
      localStorage.setItem('auth_user', JSON.stringify(data.user))
      return data
    } finally {
      loading.value = false
    }
  }

  async function logout() {
    try {
      await apiClient.post('/auth/logout')
    } finally {
      token.value = null
      user.value = null
      localStorage.removeItem('auth_token')
      localStorage.removeItem('auth_user')
    }
  }

  async function me() {
    try {
      const res = await apiClient.get<ApiResponse<User>>('/auth/me')
      user.value = res.data.data
      localStorage.setItem('auth_user', JSON.stringify(res.data.data))
      return res.data.data
    } catch {
      token.value = null
      user.value = null
      localStorage.removeItem('auth_token')
      localStorage.removeItem('auth_user')
      throw new Error('Not authenticated')
    }
  }

  function getOAuthUrl(provider: string) {
    return `/auth/oauth?provider=${provider}`
  }

  async function updateProfile(updates: Partial<Pick<User, 'name'>>) {
    const res = await apiClient.put<ApiResponse<User>>('/user/profile', updates)
    user.value = res.data.data
    localStorage.setItem('auth_user', JSON.stringify(res.data.data))
    return res.data.data
  }

  async function getSessions(): Promise<Session[]> {
    const res = await apiClient.get<ApiResponse<{ sessions: Session[] }>>('/user/sessions')
    return res.data.data.sessions
  }

  async function revokeSession(id: string) {
    await apiClient.delete(`/user/sessions/${id}`)
  }

  async function revokeAllSessions() {
    await apiClient.delete('/user/sessions')
  }

  return {
    token,
    user,
    loading,
    isAuthenticated,
    login,
    register,
    logout,
    me,
    getOAuthUrl,
    updateProfile,
    getSessions,
    revokeSession,
    revokeAllSessions,
  }
})
