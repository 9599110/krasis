import { KrasisClient } from './client';
import type { Note, NoteVersion, Folder, ShareStatus } from './types';

export interface ListNotesParams {
  page?: number;
  size?: number;
  folder_id?: string;
  keyword?: string;
  sort?: string;
  order?: string;
}

export interface CreateNoteParams {
  title: string;
  content?: string;
  folder_id?: string;
}

export interface UpdateNoteParams {
  title?: string;
  content?: string;
  version: number;
}

export class NotesModule {
  constructor(private client: KrasisClient) {}

  async list(params?: ListNotesParams) {
    const qs = new URLSearchParams();
    if (params?.page) qs.set('page', String(params.page));
    if (params?.size) qs.set('size', String(params.size));
    if (params?.folder_id) qs.set('folder_id', params.folder_id);
    if (params?.keyword) qs.set('keyword', params.keyword);
    if (params?.sort) qs.set('sort', params.sort);
    if (params?.order) qs.set('order', params.order);
    const query = qs.toString();
    return this.client.get<{ items: Note[]; total: number; page: number; size: number }>(`/notes${query ? `?${query}` : ''}`);
  }

  async get(id: string): Promise<Note> {
    return this.client.get(`/notes/${id}`);
  }

  async create(params: CreateNoteParams): Promise<Note> {
    return this.client.post('/notes', params);
  }

  async update(id: string, params: UpdateNoteParams): Promise<Note> {
    return this.client.put(`/notes/${id}`, params, {
      headers: { 'If-Match': String(params.version) },
    });
  }

  async delete(id: string, permanent = false): Promise<void> {
    await this.client.delete(`/notes/${id}${permanent ? '?permanent=true' : ''}`);
  }

  async getVersions(id: string): Promise<{ items: NoteVersion[] }> {
    return this.client.get(`/notes/${id}/versions`);
  }

  async restoreVersion(id: string, version: number): Promise<void> {
    await this.client.post(`/notes/${id}/versions/${version}/restore`);
  }
}

export class FoldersModule {
  constructor(private client: KrasisClient) {}

  async list(): Promise<{ items: Folder[] }> {
    return this.client.get('/folders');
  }

  async create(name: string, parentId?: string, color?: string): Promise<Folder> {
    return this.client.post('/folders', { name, parent_id: parentId, color });
  }

  async update(id: string, name?: string, parentId?: string, color?: string, sortOrder?: number): Promise<void> {
    await this.client.put(`/folders/${id}`, { name, parent_id: parentId, color, sort_order: sortOrder });
  }

  async delete(id: string): Promise<void> {
    await this.client.delete(`/folders/${id}`);
  }
}

export class ShareModule {
  constructor(private client: KrasisClient) {}

  async create(noteId: string, params: { share_type?: string; permission?: string; password?: string; expires_at?: string }): Promise<ShareStatus> {
    return this.client.post(`/notes/${noteId}/share`, params);
  }

  async getStatus(noteId: string): Promise<ShareStatus> {
    return this.client.get(`/notes/${noteId}/share`);
  }

  async delete(noteId: string): Promise<void> {
    await this.client.delete(`/notes/${noteId}/share`);
  }

  async access(token: string, password?: string): Promise<{ note: { id: string; title: string; content: string }; permission: string }> {
    const headers: Record<string, string> = {};
    if (password) headers['X-Share-Password'] = password;
    return this.client.get(`/share/${token}`, { headers });
  }
}
