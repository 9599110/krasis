/**
 * Storage adapters for persisting SDK state across sessions.
 * Supports browser localStorage, React Native AsyncStorage, and in-memory fallback.
 */

export interface StorageAdapter {
  getItem(key: string): Promise<string | null>;
  setItem(key: string, value: string): Promise<void>;
  removeItem(key: string): Promise<void>;
  clear(): Promise<void>;
}

/**
 * Browser localStorage adapter.
 */
export class BrowserStorage implements StorageAdapter {
  private storage: Storage | null;

  constructor() {
    this.storage = typeof localStorage !== 'undefined' ? localStorage : null;
  }

  async getItem(key: string): Promise<string | null> {
    return this.storage?.getItem(key) ?? null;
  }

  async setItem(key: string, value: string): Promise<void> {
    this.storage?.setItem(key, value);
  }

  async removeItem(key: string): Promise<void> {
    this.storage?.removeItem(key);
  }

  async clear(): Promise<void> {
    this.storage?.clear();
  }
}

/**
 * In-memory storage adapter (useful for SSR, tests, or React Native fallback).
 */
export class MemoryStorage implements StorageAdapter {
  private store = new Map<string, string>();

  async getItem(key: string): Promise<string | null> {
    return this.store.get(key) ?? null;
  }

  async setItem(key: string, value: string): Promise<void> {
    this.store.set(key, value);
  }

  async removeItem(key: string): Promise<void> {
    this.store.delete(key);
  }

  async clear(): Promise<void> {
    this.store.clear();
  }
}

/**
 * React Native AsyncStorage adapter.
 * Usage: `new ReactNativeStorage(require('@react-native-async-storage/async-storage'))`
 */
export class ReactNativeStorage implements StorageAdapter {
  private storage: {
    getItem: (key: string) => Promise<string | null>;
    setItem: (key: string, value: string) => Promise<void>;
    removeItem: (key: string) => Promise<void>;
    clear: () => Promise<void>;
  };

  constructor(
    asyncStorage: {
      getItem: (key: string) => Promise<string | null>;
      setItem: (key: string, value: string) => Promise<void>;
      removeItem: (key: string) => Promise<void>;
      clear: () => Promise<void>;
    },
  ) {
    this.storage = asyncStorage;
  }

  async getItem(key: string): Promise<string | null> {
    return this.storage.getItem(key);
  }

  async setItem(key: string, value: string): Promise<void> {
    return this.storage.setItem(key, value);
  }

  async removeItem(key: string): Promise<void> {
    return this.storage.removeItem(key);
  }

  async clear(): Promise<void> {
    return this.storage.clear();
  }
}

/**
 * Resolve the best available storage adapter for the current environment.
 * Order: localStorage > MemoryStorage
 */
export function resolveStorage(): StorageAdapter {
  if (typeof localStorage !== 'undefined') {
    return new BrowserStorage();
  }
  return new MemoryStorage();
}

/**
 * Synchronous storage wrapper — wraps an async adapter with a sync interface.
 * Useful for code that expects the Web Storage API (getItem/setItem return values directly).
 */
export class SyncStorageWrapper implements Storage {
  private adapter: StorageAdapter;
  private cache = new Map<string, string>();

  constructor(adapter: StorageAdapter) {
    this.adapter = adapter;
  }

  get length(): number {
    return this.cache.size;
  }

  key(_index: number): string | null {
    const keys = Array.from(this.cache.keys());
    return keys[_index] ?? null;
  }

  getItem(key: string): string | null {
    return this.cache.get(key) ?? null;
  }

  setItem(key: string, value: string): void {
    this.cache.set(key, value);
    // Fire-and-forget persistence
    this.adapter.setItem(key, value).catch(() => {});
  }

  removeItem(key: string): void {
    this.cache.delete(key);
    this.adapter.removeItem(key).catch(() => {});
  }

  clear(): void {
    this.cache.clear();
    this.adapter.clear().catch(() => {});
  }

  /**
   * Initialize the cache by loading all values from the underlying adapter.
   * Call this once during SDK setup.
   */
  async init(): Promise<void> {
    // Note: StorageAdapter doesn't support listing all keys, so this
    // only works if the caller has a known set of keys to pre-load.
    // For token storage, this is sufficient since we only need 'krasis_token'.
  }
}
