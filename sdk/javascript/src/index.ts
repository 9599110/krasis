import { KrasisClient } from './client';
import { AuthModule, UserModule } from './auth';
import { NotesModule, FoldersModule, ShareModule } from './notes';
import { SearchModule, FileModule } from './search';
import { AIModule } from './ai';
import { CollabModule } from './collab';
import { SDKError, VersionConflictError, RateLimitError } from './error';
import type { KrasisConfig } from './client';

export { SDKError, VersionConflictError, RateLimitError };
export * from './types';
export { BrowserStorage, MemoryStorage, ReactNativeStorage, resolveStorage, SyncStorageWrapper } from './storage';
export type { StorageAdapter } from './storage';

export class KrasisSDK {
  private client: KrasisClient;

  public readonly auth: AuthModule;
  public readonly users: UserModule;
  public readonly notes: NotesModule;
  public readonly folders: FoldersModule;
  public readonly share: ShareModule;
  public readonly search: SearchModule;
  public readonly files: FileModule;
  public readonly ai: AIModule;
  private _collab: CollabModule | null = null;

  constructor(config: KrasisConfig) {
    this.client = new KrasisClient(config);

    this.auth = new AuthModule(this.client);
    this.users = new UserModule(this.client);
    this.notes = new NotesModule(this.client);
    this.folders = new FoldersModule(this.client);
    this.share = new ShareModule(this.client);
    this.search = new SearchModule(this.client);
    this.files = new FileModule(this.client);
    this.ai = new AIModule(this.client);
  }

  get collab(): CollabModule {
    if (!this._collab) {
      this._collab = new CollabModule({
        wsBaseUrl: this.client.wsBaseUrl,
        token: this.client.token || '',
      });
    }
    return this._collab;
  }

  setToken(token: string): void {
    this.client.setToken(token);
  }

  clearToken(): void {
    this.client.clearToken();
  }

  get isAuthenticated(): boolean {
    return this.client.isAuthenticated;
  }
}

export default KrasisSDK;
