import { ref } from 'vue'
import { defineStore } from 'pinia'
import apiClient from '../api/client'
import type { ApiResponse, AskRequest, AskResponse, Conversation, Message } from '../api/types'

export const useAiStore = defineStore('ai', () => {
  const conversations = ref<Conversation[]>([])
  const messages = ref<Message[]>([])
  const currentConversationId = ref<string | null>(null)
  const loading = ref(false)
  const streamingContent = ref('')
  const isStreaming = ref(false)

  async function ask(req: AskRequest): Promise<AskResponse> {
    loading.value = true
    try {
      const res = await apiClient.post<ApiResponse<AskResponse>>('/ai/ask', req)
      return res.data.data
    } finally {
      loading.value = false
    }
  }

  async function askStream(req: AskRequest, onChunk: (text: string) => void): Promise<void> {
    const token = localStorage.getItem('auth_token')
    isStreaming.value = true
    streamingContent.value = ''

    try {
      const response = await fetch('/ai/ask/stream', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify(req),
      })

      if (!response.ok) {
        throw new Error(`SSE request failed: ${response.status}`)
      }

      const reader = response.body?.getReader()
      const decoder = new TextDecoder()
      let buffer = ''

      if (!reader) throw new Error('No response body')

      while (true) {
        const { done, value } = await reader.read()
        if (done) break

        buffer += decoder.decode(value, { stream: true })
        const lines = buffer.split('\n')
        buffer = lines.pop() || ''

        for (const line of lines) {
          const trimmed = line.trim()
          if (trimmed.startsWith('data: ')) {
            const data = trimmed.slice(6)
            if (data === '[DONE]') {
              isStreaming.value = false
              return
            }
            try {
              const parsed = JSON.parse(data)
              const text = parsed.content || parsed.delta || parsed.text || ''
              if (text) {
                streamingContent.value += text
                onChunk(text)
              }
            } catch {
              // If not JSON, treat as raw text
              if (data) {
                streamingContent.value += data
                onChunk(data)
              }
            }
          }
        }
      }
    } finally {
      isStreaming.value = false
    }
  }

  async function fetchConversations() {
    loading.value = true
    try {
      const res = await apiClient.get<ApiResponse<{ items: Conversation[] }> | ApiResponse<Conversation[]>>('/ai/conversations')
      const data = res.data.data as unknown
      const list = Array.isArray(data) ? (data as Conversation[]) : ((data as { items: Conversation[] }).items ?? [])
      conversations.value = list
      return list
    } finally {
      loading.value = false
    }
  }

  async function createConversation(): Promise<Conversation> {
    const res = await apiClient.post<ApiResponse<Conversation>>('/ai/conversations', {})
    conversations.value.unshift(res.data.data)
    return res.data.data
  }

  async function fetchMessages(conversationId: string) {
    loading.value = true
    try {
      const res = await apiClient.get<ApiResponse<{ items: Message[] }> | ApiResponse<Message[]>>(
        `/ai/conversations/${conversationId}/messages`,
      )
      const data = res.data.data as unknown
      const list = Array.isArray(data) ? (data as Message[]) : ((data as { items: Message[] }).items ?? [])
      messages.value = list
      return list
    } finally {
      loading.value = false
    }
  }

  function selectConversation(id: string | null) {
    currentConversationId.value = id
  }

  return {
    conversations,
    messages,
    currentConversationId,
    loading,
    streamingContent,
    isStreaming,
    ask,
    askStream,
    fetchConversations,
    createConversation,
    fetchMessages,
    selectConversation,
  }
})
