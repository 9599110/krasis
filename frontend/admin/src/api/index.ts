import request from '../utils/request'

// --- Users ---
export const getUsers = (params: { page: number; size: number; keyword?: string; role?: string }) =>
  request.get('/admin/users', { params })

export const getUser = (id: string) => request.get(`/admin/users/${id}`)

export const createUser = (data: { email: string; username: string; password: string; role?: string }) =>
  request.post('/admin/users', data)

export const updateUser = (id: string, data: { username?: string; email?: string; status?: number }) =>
  request.put(`/admin/users/${id}`, data)

export const deleteUser = (id: string) => request.delete(`/admin/users/${id}`)

export const updateUserRole = (id: string, role: string) =>
  request.put(`/admin/users/${id}/role`, { role })

export const updateUserStatus = (id: string, status: number) =>
  request.put(`/admin/users/${id}/status`, { status })

export const batchDisableUsers = (userIds: string[]) =>
  request.post('/admin/users/batch/disable', { user_ids: userIds })

export const exportUsers = () =>
  request.post('/admin/users/export', {}, { responseType: 'blob' })

// --- Stats ---
export const getStatsOverview = () => request.get('/admin/stats/overview')

export const getUsageStats = () => request.get('/admin/stats/usage')

// --- AI Models ---
export const getModels = (type?: string) =>
  request.get('/admin/ai/models', { params: { type } })

export const createModel = (data: Record<string, unknown>) =>
  request.post('/admin/ai/models', data)

export const updateModel = (id: string, data: Record<string, unknown>) =>
  request.put(`/admin/ai/models/${id}`, data)

export const deleteModel = (id: string) =>
  request.delete(`/admin/ai/models/${id}`)

export const testModel = (id: string) =>
  request.post(`/admin/ai/models/${id}/test`)

export const getAIConfig = () => request.get('/admin/ai/config')

export const updateAIConfig = (data: Record<string, unknown>) =>
  request.put('/admin/ai/config', data)

export const getEmbeddingModels = () =>
  request.get('/admin/ai/embedding-models')

// --- Shares ---
export const getPendingShares = (params: { page: number; size: number; status?: string; keyword?: string }) =>
  request.get('/admin/shares/pending', { params })

export const getShareDetail = (id: string) =>
  request.get(`/admin/shares/${id}`)

export const approveShare = (id: string) =>
  request.post(`/admin/shares/${id}/approve`)

export const rejectShare = (id: string, reason?: string) =>
  request.post(`/admin/shares/${id}/reject`, { reason })

export const revokeShare = (id: string) =>
  request.delete(`/admin/shares/${id}/revoke`)

export const batchReviewShares = (shareIds: string[], action: 'approve' | 'reject', reason?: string) =>
  request.post('/admin/shares/batch/review', { share_ids: shareIds, action, reason })

export const getShareStats = () => request.get('/admin/shares/stats')

// --- System Config ---
export const getSystemConfig = () => request.get('/admin/config')

export const updateSystemConfig = (data: Record<string, unknown>) =>
  request.put('/admin/config', data)

// --- OAuth Config ---
export const getOAuthConfig = () => request.get('/admin/auth/oauth')

export const updateOAuthConfig = (data: { github?: Record<string, unknown>; google?: Record<string, unknown> }) =>
  request.put('/admin/auth/oauth', data)

// --- Groups ---
export const getGroups = () => request.get('/admin/groups')

export const createGroup = (data: { name: string; description?: string; is_default?: boolean }) =>
  request.post('/admin/groups', data)

export const updateGroup = (id: string, data: { name?: string; description?: string }) =>
  request.put(`/admin/groups/${id}`, data)

export const deleteGroup = (id: string) =>
  request.delete(`/admin/groups/${id}`)

export const getGroupFeatures = (id: string) =>
  request.get(`/admin/groups/${id}/features`)

export const updateGroupFeatures = (id: string, features: Record<string, unknown>) =>
  request.put(`/admin/groups/${id}/features`, { features })

// --- Audit Logs ---
export const getAuditLogs = (params: {
  page: number
  size: number
  action?: string
  user_id?: string
  start_date?: string
  end_date?: string
}) => request.get('/admin/logs', { params })
