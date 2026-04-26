import apiClient from './client'

export type PresignResult = {
  file_id: string
  upload_url: string
  expires_in: number
}

export async function presignUpload(params: { file_name: string; file_type: string; note_id?: string }) {
  return apiClient.get('/files/presign', { params })
}

export async function confirmUpload(body: { file_id: string; note_id?: string; metadata?: Record<string, any> }) {
  return apiClient.post('/files/confirm', body)
}

