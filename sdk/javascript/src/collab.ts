import type { WSMessage, AwarenessPayload, SyncPayload } from './types';

export interface CollabConfig {
  wsBaseUrl: string;
  token: string;
}

export type CollabEvent =
  | { type: 'open' }
  | { type: 'close'; code: number; reason: string }
  | { type: 'error'; error: Event }
  | { type: 'sync'; payload: SyncPayload; userId: string }
  | { type: 'awareness'; payload: AwarenessPayload }
  | { type: 'presence'; users: Array<{ user_id: string; username: string }> };

type CollabEventHandler = (event: CollabEvent) => void;

export class CollabModule {
  private ws: WebSocket | null = null;
  private handlers: CollabEventHandler[] = [];
  private noteId: string | null = null;
  private config: CollabConfig;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;

  constructor(config: CollabConfig) {
    this.config = config;
  }

  on(handler: CollabEventHandler): () => void {
    this.handlers.push(handler);
    return () => {
      this.handlers = this.handlers.filter((h) => h !== handler);
    };
  }

  connect(noteId: string): void {
    this.noteId = noteId;
    this.reconnectAttempts = 0;
    this.doConnect();
  }

  private doConnect(): void {
    if (this.ws) {
      this.ws.close();
    }

    const url = `${this.config.wsBaseUrl}/ws/collab?note_id=${this.noteId}&token=${this.config.token}`;
    this.ws = new WebSocket(url);

    this.ws.onopen = () => {
      this.reconnectAttempts = 0;
      this.emit({ type: 'open' });
    };

    this.ws.onmessage = (event) => {
      try {
        const msg: WSMessage = JSON.parse(event.data);
        this.handleMessage(msg);
      } catch {
        // ignore parse errors
      }
    };

    this.ws.onclose = (event) => {
      this.emit({ type: 'close', code: event.code, reason: event.reason });
      this.scheduleReconnect();
    };

    this.ws.onerror = (error) => {
      this.emit({ type: 'error', error });
    };
  }

  private handleMessage(msg: WSMessage): void {
    switch (msg.type) {
      case 'sync':
        this.emit({ type: 'sync', payload: msg.payload as SyncPayload, userId: msg.payload.user_id as string });
        break;
      case 'awareness':
        this.emit({ type: 'awareness', payload: msg.payload as AwarenessPayload });
        break;
      case 'presence':
        this.emit({ type: 'presence', users: msg.payload.users as Array<{ user_id: string; username: string }> });
        break;
      default:
        break;
    }
  }

  sendSync(update: string, version: number): void {
    this.send({ type: 'sync', payload: { update, version } });
  }

  sendAwareness(payload: AwarenessPayload): void {
    this.send({ type: 'awareness', payload });
  }

  sendPresenceQuery(): void {
    this.send({ type: 'awareness_query', payload: {} });
  }

  private send(msg: WSMessage): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(msg));
    }
  }

  private scheduleReconnect(): void {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) return;

    this.reconnectAttempts++;
    const delay = Math.min(1000 * Math.pow(2, this.reconnectAttempts), 30000);
    this.reconnectTimer = setTimeout(() => this.doConnect(), delay);
  }

  disconnect(): void {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    this.reconnectAttempts = this.maxReconnectAttempts; // prevent reconnect
    this.ws?.close();
    this.ws = null;
  }

  private emit(event: CollabEvent): void {
    for (const handler of this.handlers) {
      try {
        handler(event);
      } catch {
        // ignore handler errors
      }
    }
  }
}
