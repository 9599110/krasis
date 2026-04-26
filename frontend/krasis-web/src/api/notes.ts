import apiClient from './client'
import type { CreateNoteRequest, UpdateNoteRequest } from './types'

// Notes
export const listNotes = (params?: { folder_id?: string; page?: number; size?: number }) =>
  apiClient.get('/notes', { params })

export const createNote = (data: CreateNoteRequest) =>
  apiClient.post('/notes', data)

export const getNote = (id: string) =>
  apiClient.get(`/notes/${id}`)

export const updateNote = (id: string, data: UpdateNoteRequest, version?: number) => {
  const headers: Record<string, string> = {}
  if (version !== undefined && version !== null) {
    headers['If-Match'] = String(version)
  }
  return apiClient.put(`/notes/${id}`, data, { headers })
}

export const deleteNote = (id: string) =>
  apiClient.delete(`/notes/${id}`)

// Versions
export const getNoteVersions = (id: string) =>
  apiClient.get(`/notes/${id}/versions`)

export const restoreVersion = (id: string, version: number, data: { title: string; content: string }) =>
  apiClient.post(`/notes/${id}/versions/${version}/restore`, data)

// Folders
export const listFolders = () =>
  apiClient.get('/folders')

export const createFolder = (data: { name: string; parent_id?: string | null }) =>
  apiClient.post('/folders', data)

export const updateFolder = (id: string, data: { name: string }) =>
  apiClient.put(`/folders/${id}`, data)

export const deleteFolder = (id: string) =>
  apiClient.delete(`/folders/${id}`)

// Shares
export const createShare = (noteId: string, data: { password?: string; expires_at?: string | null }) =>
  apiClient.post(`/notes/${noteId}/share`, data)

export const getShareStatus = (noteId: string) =>
  apiClient.get(`/notes/${noteId}/share`)

export const deleteShare = (noteId: string) =>
  apiClient.delete(`/notes/${noteId}/share`)
