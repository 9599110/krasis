import apiClient from './client'
import type { FolderFile } from './files'

/** List hidden files in a folder (only visible via this explicit endpoint) */
export async function listHiddenFiles(folderId: string): Promise<FolderFile[]> {
  const res = await apiClient.get(`/files/folder/${folderId}/hidden`)
  return res.data?.data || []
}

/** Unhide a file (is_hidden=false), making it visible in normal listings */
export async function unhideFile(fileId: string): Promise<void> {
  await apiClient.post(`/files/${fileId}/unhide`)
}

/** Mark a file as hidden (is_hidden=true), hiding it from normal listings */
export async function markHiddenFile(fileId: string): Promise<void> {
  await apiClient.post(`/files/${fileId}/hide`)
}
