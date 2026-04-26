import { KrasisClient } from './client';
import type { SearchResult } from './types';

export interface SearchParams {
  q: string;
  page?: number;
  size?: number;
  type?: 'notes' | 'files' | 'all';
}

export class SearchModule {
  constructor(private client: KrasisClient) {}

  async search(params: SearchParams) {
    const qs = new URLSearchParams();
    qs.set('q', params.q);
    if (params.page) qs.set('page', String(params.page));
    if (params.size) qs.set('size', String(params.size));
    if (params.type) qs.set('type', params.type);
    return this.client.get<{ items: SearchResult[]; total: number; page: number; size: number }>(`/search?${qs.toString()}`);
  }
}

export class FileModule {
  constructor(private client: KrasisClient) {}

  async getPresignUrl(fileName: string, fileType: string, noteId?: string): Promise<{ file_id: string; upload_url: string; expires_in: number }> {
    const qs = new URLSearchParams({ file_name: fileName, file_type: fileType });
    if (noteId) qs.set('note_id', noteId);
    return this.client.get(`/files/presign?${qs.toString()}`);
  }

  async confirmUpload(fileId: string, noteId?: string, metadata?: Record<string, unknown>): Promise<void> {
    await this.client.post('/files/confirm', { file_id: fileId, note_id: noteId, metadata });
  }

  async delete(fileId: string): Promise<void> {
    await this.client.delete(`/files/${fileId}`);
  }
}
