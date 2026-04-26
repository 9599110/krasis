import { KrasisClient } from './client';
import type { User, Session } from './types';

export class AuthModule {
  constructor(private client: KrasisClient) {}

  getGitHubLoginUrl(state?: string): string {
    const params = new URLSearchParams();
    if (state) params.set('state', state);
    return `${this.client.apiBaseUrl}/auth/github/login?${params}`;
  }

  getGoogleLoginUrl(state?: string): string {
    const params = new URLSearchParams();
    if (state) params.set('state', state);
    return `${this.client.apiBaseUrl}/auth/google/login?${params}`;
  }

  async githubCallback(code: string, state: string): Promise<{ access_token: string; token_type: string; expires_in: number; user: User }> {
    return this.client.get(`/auth/github/callback?code=${encodeURIComponent(code)}&state=${encodeURIComponent(state)}`);
  }

  async googleCallback(code: string, state: string): Promise<{ access_token: string; token_type: string; expires_in: number; user: User }> {
    return this.client.get(`/auth/google/callback?code=${encodeURIComponent(code)}&state=${encodeURIComponent(state)}`);
  }

  async logout(): Promise<void> {
    try {
      await this.client.post('/auth/logout');
    } finally {
      this.client.clearToken();
    }
  }

  async getMe(): Promise<User> {
    return this.client.get('/auth/me');
  }
}

export class UserModule {
  constructor(private client: KrasisClient) {}

  async getSessions(): Promise<{ sessions: Session[] }> {
    return this.client.get('/user/sessions');
  }

  async deleteSession(sessionId: string): Promise<void> {
    await this.client.delete(`/user/sessions/${sessionId}`);
  }

  async updateProfile(username?: string, avatarUrl?: string): Promise<void> {
    await this.client.put('/user/profile', { username, avatar_url: avatarUrl });
  }
}
