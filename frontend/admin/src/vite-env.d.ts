/// <reference types="vite/client" />

declare namespace API {
  interface User {
    id: string
    email: string
    username: string
    role: string
    status: number
    created_at: string
  }

  interface Model {
    id: string
    name: string
    provider: string
    type: string
    model_name: string
    endpoint?: string
    is_enabled: boolean
    is_default: boolean
    created_at: string
  }

  interface Share {
    id: string
    note_title?: string
    creator_email?: string
    status: string
    expires_at?: string
    created_at: string
  }

  interface Log {
    id: string
    action: string
    target_type: string
    admin_id: string
    ip_address?: string
    created_at: string
  }

  interface Group {
    id: string
    name: string
    description: string
    is_default: boolean
    user_count: number
    created_at: string
    updated_at?: string
  }
}
