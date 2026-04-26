export interface ApiResponse<T = unknown> {
  code: number;
  message: string;
  data: T;
}

export interface PaginatedResponse<T> {
  items: T[];
  total: number;
  page: number;
  size: number;
}

export interface User {
  id: string;
  email: string;
  username: string;
  avatar_url: string;
  role: string;
  status: number;
  created_at: string;
}

export interface Note {
  id: string;
  title: string;
  content: string;
  content_html?: string;
  owner_id: string;
  folder_id?: string;
  version: number;
  is_public: boolean;
  share_token?: string;
  view_count: number;
  created_at: string;
  updated_at: string;
}

export interface NoteVersion {
  id: string;
  note_id: string;
  title?: string;
  content?: string;
  version: number;
  changed_by?: string;
  change_summary?: string;
  created_at: string;
}

export interface Folder {
  id: string;
  name: string;
  parent_id?: string;
  owner_id: string;
  color?: string;
  sort_order: number;
  created_at: string;
  updated_at: string;
}

export interface ShareStatus {
  share_token: string;
  share_url: string;
  permission: string;
  password_protected: boolean;
  expires_at?: string;
  status: 'pending' | 'approved' | 'rejected' | 'revoked';
  status_description: string;
  created_at: string;
  rejection_reason?: string;
}

export interface SearchResult {
  type: string;
  id: string;
  title: string;
  highlights: string[];
  score: number;
  updated_at: string;
}

export interface FileItem {
  id: string;
  note_id?: string;
  user_id: string;
  file_name: string;
  file_type: string;
  storage_path: string;
  bucket: string;
  size_bytes?: number;
  status: number;
  created_at: string;
}

export interface PresignResult {
  file_id: string;
  upload_url: string;
  expires_in: number;
}

export interface Session {
  session_id: string;
  device_name: string;
  device_type: string;
  ip_address: string;
  user_agent: string;
  last_active_at: string;
  created_at: string;
  is_current: boolean;
}

export interface AskRequest {
  question: string;
  conversation_id?: string;
  model_id?: string;
  top_k?: number;
  stream?: boolean;
}

export interface AskResponse {
  answer: string;
  references?: Array<{
    note_id: string;
    note_title: string;
    text: string;
    chunk_index: number;
  }>;
  conversation_id: string;
  message_id?: string;
}

export interface Conversation {
  id: string;
  user_id: string;
  title: string;
  model: string;
  created_at: string;
  updated_at: string;
}

export interface Message {
  id: string;
  conversation_id: string;
  role: string;
  content: string;
  references?: unknown[];
  token_count?: number;
  model?: string;
  created_at: string;
}

export interface AIModel {
  id: string;
  name: string;
  provider: string;
  type: string;
  endpoint: string;
  model_name: string;
  api_version?: string;
  max_tokens: number;
  temperature: number;
  top_p?: number;
  dimensions?: number;
  is_enabled: boolean;
  is_default: boolean;
  priority: number;
  created_at: string;
  updated_at: string;
}

// WebSocket collaboration types
export interface WSMessage {
  type: 'sync' | 'awareness' | 'awareness_query' | 'presence' | 'error';
  payload: Record<string, any>;
}

export interface AwarenessPayload {
  user_id: string;
  username: string;
  cursor?: { line: number; column: number };
  selection?: { from: number; to: number };
}

export interface SyncPayload {
  update: string; // base64-encoded
  version: number;
}
