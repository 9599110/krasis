import apiClient from './client'
import type { AskRequest } from './types'

export const ask = (data: AskRequest) =>
  apiClient.post('/ai/ask', data)

export function askStream(
  data: AskRequest,
  onToken: (token: string) => void,
  onDone: () => void,
  onError: (err: Error) => void,
) {
  const token = localStorage.getItem('auth_token')
  const base = import.meta.env.VITE_API_BASE_URL || ''

  const xhr = new XMLHttpRequest()
  xhr.open('POST', `${base}/ai/ask/stream`)
  xhr.setRequestHeader('Content-Type', 'application/json')
  if (token) xhr.setRequestHeader('Authorization', `Bearer ${token}`)
  xhr.responseType = 'text'
  xhr.timeout = 120000

  xhr.onreadystatechange = () => {
    if (xhr.readyState === xhr.HEADERS_RECEIVED) {
      const ct = xhr.getResponseHeader('Content-Type')
      if (!ct?.includes('text/event-stream')) {
        // Not a stream — probably an error response
        const reader = new Response(xhr.response as string)
        reader.text().then((body) => {
          try {
            const json = JSON.parse(body)
            onError(new Error(json?.message || 'Stream error'))
          } catch {
            onError(new Error('Stream error'))
          }
        })
        xhr.abort()
      }
    }
  }

  let buf = ''
  xhr.onprogress = () => {
    const newText = xhr.responseText.slice(buf.length)
    buf = xhr.responseText
    const lines = newText.split('\n')
    for (let i = 0; i < lines.length; i++) {
      const line = lines[i]
      if (line.startsWith('data: ')) {
        try {
          const payload = JSON.parse(line.slice(6))
          if (payload.token != null) onToken(payload.token)
        } catch {
          // skip malformed line
        }
      }
    }
  }

  xhr.onload = () => {
    if (xhr.status >= 200 && xhr.status < 300) onDone()
    else onError(new Error(`HTTP ${xhr.status}`))
  }

  xhr.onerror = () => onError(new Error('Network error'))
  xhr.ontimeout = () => onError(new Error('Request timeout'))

  xhr.send(JSON.stringify(data))

  return () => xhr.abort()
}

export const listConversations = () =>
  apiClient.get('/ai/conversations')

export const getMessages = (conversationId: string) =>
  apiClient.get(`/ai/conversations/${conversationId}/messages`)
