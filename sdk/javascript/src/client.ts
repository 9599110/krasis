import { SDKError, VersionConflictError, RateLimitError } from './error';
import type { ApiResponse } from './types';

export interface KrasisConfig {
  apiBaseUrl: string;
  wsBaseUrl?: string;
  clientId?: string;
  storage?: Storage;
}

export interface RequestOptions {
  headers?: Record<string, string>;
}

export class KrasisClient {
  public apiBaseUrl: string;
  public wsBaseUrl: string;
  public clientId: string;
  private _token: string | null = null;
  private storage: Storage | null;

  constructor(config: KrasisConfig) {
    this.apiBaseUrl = config.apiBaseUrl.replace(/\/+$/, '');
    this.wsBaseUrl = config.wsBaseUrl || config.apiBaseUrl.replace(/^http/, 'ws').replace(/\/+$/, '');
    this.clientId = config.clientId || this.generateClientId();
    this.storage = config.storage ?? (typeof localStorage !== 'undefined' ? localStorage : null);

    // Restore token from storage
    if (this.storage) {
      this._token = this.storage.getItem('krasis_token');
    }
  }

  private generateClientId(): string {
    return `sdk_${Math.random().toString(36).substring(2, 10)}`;
  }

  get token(): string | null { return this._token; }

  get isAuthenticated(): boolean {
    return this._token !== null;
  }

  setToken(token: string): void {
    this._token = token;
    if (this.storage) {
      this.storage.setItem('krasis_token', token);
    }
  }

  clearToken(): void {
    this._token = null;
    if (this.storage) {
      this.storage.removeItem('krasis_token');
    }
  }

  async request<T>(method: string, path: string, body?: unknown, options?: RequestOptions): Promise<T> {
    const url = `${this.apiBaseUrl}${path}`;
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      ...options?.headers,
    };

    if (this._token) {
      headers['Authorization'] = `Bearer ${this._token}`;
    }

    const init: RequestInit = {
      method,
      headers,
    };

    if (body && method !== 'GET') {
      init.body = JSON.stringify(body);
    }

    const res = await fetch(url, init);
    const json: ApiResponse<T> = await res.json();

    if (json.code !== 0) {
      if (json.code === 1005 && res.status === 409) {
        throw new VersionConflictError(
          (json.data as Record<string, unknown>)?.current_version as number || 0,
          (json.data as Record<string, unknown>)?.note,
        );
      }
      if (json.code === 2001) {
        throw new RateLimitError(
          (json.data as Record<string, unknown>)?.retry_after as number || 60,
        );
      }
      throw new SDKError(json.message, json.code, res.status, json.data);
    }

    return json.data;
  }

  get<T>(path: string, options?: RequestOptions): Promise<T> {
    return this.request<T>('GET', path, undefined, options);
  }

  post<T>(path: string, body?: unknown, options?: RequestOptions): Promise<T> {
    return this.request<T>('POST', path, body, options);
  }

  put<T>(path: string, body?: unknown, options?: RequestOptions): Promise<T> {
    return this.request<T>('PUT', path, body, options);
  }

  delete<T>(path: string, options?: RequestOptions): Promise<T> {
    return this.request<T>('DELETE', path, undefined, options);
  }
}
