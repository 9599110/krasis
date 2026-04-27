// API envelope wrapper
export interface ApiResponse<T = unknown> {
  data: T
  code: number
  message: string
}

// Auth / User
export interface User {
  id: string
  email: string
  name: string
  role?: string
  avatar_url?: string
  created_at: string
  updated_at: string
}

export interface LoginRequest {
  username: string
  password: string
}

export interface RegisterRequest {
  username: string
  email: string
  password: string
  name: string
}

export interface AuthResponse {
  token: string
  user: User
}

// Session
export interface Session {
  id: string
  ip_address?: string
  device_name?: string
  is_current?: boolean
  user_agent: string
  created_at: string
  last_active: string
}

// Notes
export interface Note {
  id: string
  title: string
  content: string
  folder_id: string | null
  user_id: string
  version: number
  created_at: string
  updated_at: string
}

export interface CreateNoteRequest {
  title: string
  content?: string
  folder_id?: string | null
}

export interface UpdateNoteRequest {
  title?: string
  content?: string
  folder_id?: string | null
}

// Note Version
export interface NoteVersion {
  id: string
  note_id: string
  version: number
  title: string
  content: string
  created_at: string
}

// Folders
export interface Folder {
  id: string
  name: string
  user_id: string
  parent_id: string | null
  sort_order: number
  created_at: string
  updated_at: string
}

export interface CreateFolderRequest {
  name: string
  parent_id?: string | null
}

// Share
export const ShareStatus = {
  Active: 'active',
  Expired: 'expired',
  Revoked: 'revoked',
} as const

export type ShareStatus = (typeof ShareStatus)[keyof typeof ShareStatus]

export interface ShareLink {
  id: string
  note_id: string
  token: string
  status: ShareStatus
  expires_at: string | null
  created_at: string
}

export interface CreateShareRequest {
  expires_at?: string | null
}

export interface SharedNote {
  note: Note
  shared_at: string
}

// Search
export interface SearchResult {
  id: string
  title: string
  content: string
  folder_id: string | null
  user_id: string
  version: number
  created_at: string
  updated_at: string
  highlights?: string[]
  score?: number
}

export interface SearchResponse {
  results: SearchResult[]
  total: number
  page: number
  size: number
}

// AI
export interface AskRequest {
  question: string
  conversation_id?: string
  model_id?: string
  top_k?: number
}

export interface AskResponse {
  answer: string
  conversation_id: string
}

export interface Conversation {
  id: string
  title: string
  user_id: string
  created_at: string
  updated_at: string
}

export interface Message {
  id: string
  conversation_id: string
  role: 'user' | 'assistant'
  content: string
  created_at: string
}

// Collaboration
export interface CollabEvent {
  type: 'cursor' | 'change' | 'presence'
  user_id: string
  user_name: string
  data: unknown
}
