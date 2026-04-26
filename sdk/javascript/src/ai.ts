import { KrasisClient } from './client';
import type { AskRequest, AskResponse, Conversation, Message } from './types';

export class AIModule {
  constructor(private client: KrasisClient) {}

  async ask(params: AskRequest): Promise<AskResponse> {
    return this.client.post('/ai/ask', params);
  }

  askStream(params: AskRequest, onToken: (token: string) => void, onDone?: () => void, onError?: (err: Error) => void): AbortController {
    const controller = new AbortController();
    const url = `${this.client.apiBaseUrl}/ai/ask/stream`;

    fetch(url, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        ...(this.client.token ? { Authorization: `Bearer ${this.client.token}` } : {}),
      },
      body: JSON.stringify({ ...params, stream: true }),
      signal: controller.signal,
    })
      .then(async (res) => {
        if (!res.ok) {
          onError?.(new Error(`HTTP ${res.status}`));
          return;
        }

        const reader = res.body?.getReader();
        if (!reader) {
          onError?.(new Error('ReadableStream not supported'));
          return;
        }

        const decoder = new TextDecoder();
        let buffer = '';

        while (true) {
          const { done, value } = await reader.read();
          if (done) break;

          buffer += decoder.decode(value, { stream: true });
          const lines = buffer.split('\n');
          buffer = lines.pop() || '';

          for (const line of lines) {
            if (line.startsWith('event: token')) {
              const nextLine = lines[lines.indexOf(line) + 1];
              if (nextLine?.startsWith('data: ')) {
                try {
                  const data = JSON.parse(nextLine.slice(6));
                  if (data.token) onToken(data.token);
                } catch { /* skip */ }
              }
            } else if (line.startsWith('event: done')) {
              onDone?.();
              return;
            }
          }
        }
        onDone?.();
      })
      .catch((err) => {
        if (err.name !== 'AbortError') {
          onError?.(err);
        }
      });

    return controller;
  }

  async listConversations(): Promise<Conversation[]> {
    return this.client.get('/ai/conversations');
  }

  async getMessages(conversationId: string): Promise<Message[]> {
    return this.client.get(`/ai/conversations/${conversationId}/messages`);
  }
}
