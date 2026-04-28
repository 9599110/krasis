import axios from 'axios'
import type { AxiosInstance, InternalAxiosRequestConfig, AxiosError } from 'axios'
import type { ApiResponse } from './types'

const apiClient: AxiosInstance = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '',
  timeout: 30000,
  headers: { 'Content-Type': 'application/json' },
})

// Request interceptor: attach JWT
apiClient.interceptors.request.use((config: InternalAxiosRequestConfig) => {
  const token = localStorage.getItem('auth_token')
  if (token && config.headers) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// Response interceptor: unwrap envelope, handle errors
apiClient.interceptors.response.use(
  (response) => {
    // Return the full response; stores will unwrap .data.data
    return response
  },
  (error: AxiosError<ApiResponse>) => {
    const status = error.response?.status

    if (status === 401) {
      localStorage.removeItem('auth_token')
      localStorage.removeItem('auth_user')
      if (!window.location.pathname.startsWith('/login') &&
          !window.location.pathname.startsWith('/register') &&
          !window.location.pathname.startsWith('/share/')) {
        window.location.href = `/login?redirect=${encodeURIComponent(window.location.pathname)}`
      }
    }

    if (status === 409) {
      // Version conflict — surface to caller
      const envelope = error.response?.data as unknown as ApiResponse<{ version?: number }>
      return Promise.reject({
        ...error,
        isVersionConflict: true,
        serverVersion: envelope?.data?.version,
      })
    }

    return Promise.reject(error)
  },
)

export default apiClient
