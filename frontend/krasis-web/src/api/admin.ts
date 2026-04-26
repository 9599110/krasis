import apiClient from './client'

// --- Stats ---
export const getStatsOverview = () => apiClient.get('/admin/stats/overview')
export const getUserStats = () => apiClient.get('/admin/stats/users')
export const getUsageStats = () => apiClient.get('/admin/stats/usage')

// --- Users ---
export const listUsers = (params: Record<string, any>) => apiClient.get('/admin/users', { params })
export const getUser = (id: string) => apiClient.get(`/admin/users/${id}`)
export const createUser = (data: Record<string, any>) => apiClient.post('/admin/users', data)
export const updateUser = (id: string, data: Record<string, any>) => apiClient.put(`/admin/users/${id}`, data)
export const deleteUser = (id: string) => apiClient.delete(`/admin/users/${id}`)
export const updateUserRole = (id: string, role: string) => apiClient.put(`/admin/users/${id}/role`, { role })
export const updateUserStatus = (id: string, status: number) => apiClient.put(`/admin/users/${id}/status`, { status })
export const batchDisableUsers = (userIds: string[]) => apiClient.post('/admin/users/batch/disable', { user_ids: userIds })
export const exportUsers = () => apiClient.post('/admin/users/export', { responseType: 'blob' })

// --- Share Review ---
export const getPendingShares = (params: Record<string, any>) => apiClient.get('/admin/shares/pending', { params })
export const getShareDetail = (id: string) => apiClient.get(`/admin/shares/${id}`)
export const approveShare = (id: string) => apiClient.post(`/admin/shares/${id}/approve`, {})
export const rejectShare = (id: string, reason: string) => apiClient.post(`/admin/shares/${id}/reject`, { reason })
export const reReviewShare = (id: string) => apiClient.post(`/admin/shares/${id}/re-review`, {})
export const revokeShare = (id: string) => apiClient.delete(`/admin/shares/${id}/revoke`)
export const batchReview = (shareIds: string[], action: string, reason?: string) =>
  apiClient.post('/admin/shares/batch/review', { share_ids: shareIds, action, reason })
export const getShareStats = () => apiClient.get('/admin/shares/stats')

// --- AI Models ---
export const listModels = (params?: Record<string, any>) => apiClient.get('/admin/ai/models', { params })
export const createModel = (data: Record<string, any>) => apiClient.post('/admin/ai/models', data)
export const updateModel = (id: string, data: Record<string, any>) => apiClient.put(`/admin/ai/models/${id}`, data)
export const deleteModel = (id: string) => apiClient.delete(`/admin/ai/models/${id}`)
export const testModel = (id: string) => apiClient.post(`/admin/ai/models/${id}/test`, {})
export const getAIConfig = () => apiClient.get('/admin/ai/config')
export const updateAIConfig = (data: Record<string, any>) => apiClient.put('/admin/ai/config', data)
export const listEmbeddingModels = () => apiClient.get('/admin/ai/embedding-models')

// --- System Config ---
export const getSystemConfig = () => apiClient.get('/admin/config')
export const updateSystemConfig = (data: Record<string, any>) => apiClient.put('/admin/config', data)

// --- OAuth Config ---
export const getOAuthConfig = () => apiClient.get('/admin/auth/oauth')
export const updateOAuthConfig = (data: Record<string, any>) => apiClient.put('/admin/auth/oauth', data)

// --- Groups ---
export const listGroups = () => apiClient.get('/admin/groups')
export const createGroup = (data: Record<string, any>) => apiClient.post('/admin/groups', data)
export const updateGroup = (id: string, data: Record<string, any>) => apiClient.put(`/admin/groups/${id}`, data)
export const deleteGroup = (id: string) => apiClient.delete(`/admin/groups/${id}`)
export const getGroupFeatures = (id: string) => apiClient.get(`/admin/groups/${id}/features`)
export const updateGroupFeatures = (id: string, features: Record<string, any>) =>
  apiClient.put(`/admin/groups/${id}/features`, { features })

// --- Audit Logs ---
export const getAuditLogs = (params: Record<string, any>) => apiClient.get('/admin/logs', { params })
