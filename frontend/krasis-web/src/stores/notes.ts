import { ref } from 'vue'
import { defineStore } from 'pinia'
import apiClient from '../api/client'
import type { ApiResponse, Note, NoteVersion, CreateNoteRequest, UpdateNoteRequest } from '../api/types'

export const useNotesStore = defineStore('notes', () => {
  const notes = ref<Note[]>([])
  const currentNote = ref<Note | null>(null)
  const versions = ref<NoteVersion[]>([])
  const loading = ref(false)

  async function fetchNotes(folderId?: string | null) {
    loading.value = true
    try {
      const params: Record<string, string> = {}
      if (folderId !== undefined) {
        params.folder_id = folderId ?? ''
      }
      const res = await apiClient.get<ApiResponse<{ items: Note[] }> | ApiResponse<Note[]>>('/notes', { params })
      const data = res.data.data as unknown
      const list = Array.isArray(data) ? (data as Note[]) : ((data as { items: Note[] }).items ?? [])
      notes.value = list
      return list
    } finally {
      loading.value = false
    }
  }

  async function createNote(req: CreateNoteRequest) {
    const res = await apiClient.post<ApiResponse<Note>>('/notes', req)
    notes.value.unshift(res.data.data)
    return res.data.data
  }

  async function getNote(id: string) {
    loading.value = true
    try {
      const res = await apiClient.get<ApiResponse<Note>>(`/notes/${id}`)
      currentNote.value = res.data.data
      return res.data.data
    } finally {
      loading.value = false
    }
  }

  async function updateNote(id: string, req: UpdateNoteRequest, version?: number) {
    const headers: Record<string, string> = {}
    if (version !== undefined) {
      headers['If-Match'] = String(version)
    }
    const res = await apiClient.put<ApiResponse<Note>>(`/notes/${id}`, req, { headers })
    currentNote.value = res.data.data
    // Update in list
    const idx = notes.value.findIndex((n) => n.id === id)
    if (idx >= 0) {
      notes.value[idx] = res.data.data
    }
    return res.data.data
  }

  async function deleteNote(id: string) {
    await apiClient.delete(`/notes/${id}`)
    notes.value = notes.value.filter((n) => n.id !== id)
    if (currentNote.value?.id === id) {
      currentNote.value = null
    }
  }

  async function fetchVersions(noteId: string) {
    loading.value = true
    try {
      const res = await apiClient.get<ApiResponse<{ items: NoteVersion[] }> | ApiResponse<NoteVersion[]>>(
        `/notes/${noteId}/versions`,
      )
      const data = res.data.data as unknown
      const list = Array.isArray(data) ? (data as NoteVersion[]) : ((data as { items: NoteVersion[] }).items ?? [])
      versions.value = list
      return list
    } finally {
      loading.value = false
    }
  }

  async function restoreVersion(noteId: string, versionNumber: number) {
    await apiClient.post(`/notes/${noteId}/versions/${versionNumber}/restore`)
    // Refresh the current note
    if (currentNote.value?.id === noteId) {
      await getNote(noteId)
    }
    await fetchVersions(noteId)
  }

  return {
    notes,
    currentNote,
    versions,
    loading,
    fetchNotes,
    createNote,
    getNote,
    updateNote,
    deleteNote,
    fetchVersions,
    restoreVersion,
  }
})
