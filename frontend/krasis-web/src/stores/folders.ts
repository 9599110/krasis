import { ref } from 'vue'
import { defineStore } from 'pinia'
import apiClient from '../api/client'
import type { ApiResponse, Folder, CreateFolderRequest } from '../api/types'

export const useFoldersStore = defineStore('folders', () => {
  const folders = ref<Folder[]>([])
  const loading = ref(false)

  async function fetchFolders() {
    loading.value = true
    try {
      const res = await apiClient.get<ApiResponse<{ items: Folder[] }> | ApiResponse<Folder[]>>('/folders')
      // backend currently returns `{code,message,data}`; `data` shape may differ by implementation
      const data = res.data.data as unknown
      const list = Array.isArray(data) ? (data as Folder[]) : ((data as { items: Folder[] }).items ?? [])
      folders.value = list
      return list
    } finally {
      loading.value = false
    }
  }

  async function createFolder(req: CreateFolderRequest) {
    const res = await apiClient.post<ApiResponse<Folder>>('/folders', req)
    folders.value.push(res.data.data)
    return res.data.data
  }

  async function updateFolder(id: string, updates: Partial<CreateFolderRequest>) {
    const res = await apiClient.put<ApiResponse<Folder>>(`/folders/${id}`, updates)
    const idx = folders.value.findIndex((f) => f.id === id)
    if (idx >= 0) {
      folders.value[idx] = res.data.data
    }
    return res.data.data
  }

  async function deleteFolder(id: string) {
    await apiClient.delete(`/folders/${id}`)
    folders.value = folders.value.filter((f) => f.id !== id)
  }

  return {
    folders,
    loading,
    fetchFolders,
    createFolder,
    updateFolder,
    deleteFolder,
  }
})
