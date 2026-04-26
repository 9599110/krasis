import { create } from 'zustand'
import request from '../utils/request'

interface AuthState {
  user: { id: string; email: string; username: string; role: string } | null
  login: (email: string, password: string) => Promise<void>
  logout: () => void
  fetchMe: () => Promise<void>
}

export const useAuth = create<AuthState>((set) => ({
  user: null,

  login: async (email: string, password: string) => {
    const res = await request.post('/auth/login', { email, password })
    const token = res.data.data.token
    localStorage.setItem('admin_token', token)
    const meRes = await request.get('/auth/me')
    set({ user: meRes.data.data })
  },

  logout: () => {
    localStorage.removeItem('admin_token')
    set({ user: null })
    window.location.href = '/login'
  },

  fetchMe: async () => {
    try {
      const res = await request.get('/auth/me')
      set({ user: res.data.data })
    } catch {
      set({ user: null })
    }
  },
}))
