interface KrasisConfig {
    apiBaseUrl: string;
    wsBaseUrl?: string;
    clientId?: string;
    storage?: Storage;
}
interface RequestOptions {
    headers?: Record<string, string>;
}
declare class KrasisClient {
    apiBaseUrl: string;
    wsBaseUrl: string;
    clientId: string;
    private _token;
    private storage;
    constructor(config: KrasisConfig);
    private generateClientId;
    get token(): string | null;
    get isAuthenticated(): boolean;
    setToken(token: string): void;
    clearToken(): void;
    request<T>(method: string, path: string, body?: unknown, options?: RequestOptions): Promise<T>;
    get<T>(path: string, options?: RequestOptions): Promise<T>;
    post<T>(path: string, body?: unknown, options?: RequestOptions): Promise<T>;
    put<T>(path: string, body?: unknown, options?: RequestOptions): Promise<T>;
    delete<T>(path: string, options?: RequestOptions): Promise<T>;
}

interface ApiResponse<T = unknown> {
    code: number;
    message: string;
    data: T;
}
interface PaginatedResponse<T> {
    items: T[];
    total: number;
    page: number;
    size: number;
}
interface User {
    id: string;
    email: string;
    username: string;
    avatar_url: string;
    role: string;
    status: number;
    created_at: string;
}
interface Note {
    id: string;
    title: string;
    content: string;
    content_html?: string;
    owner_id: string;
    folder_id?: string;
    version: number;
    is_public: boolean;
    share_token?: string;
    view_count: number;
    created_at: string;
    updated_at: string;
}
interface NoteVersion {
    id: string;
    note_id: string;
    title?: string;
    content?: string;
    version: number;
    changed_by?: string;
    change_summary?: string;
    created_at: string;
}
interface Folder {
    id: string;
    name: string;
    parent_id?: string;
    owner_id: string;
    color?: string;
    sort_order: number;
    created_at: string;
    updated_at: string;
}
interface ShareStatus {
    share_token: string;
    share_url: string;
    permission: string;
    password_protected: boolean;
    expires_at?: string;
    status: 'pending' | 'approved' | 'rejected' | 'revoked';
    status_description: string;
    created_at: string;
    rejection_reason?: string;
}
interface SearchResult {
    type: string;
    id: string;
    title: string;
    highlights: string[];
    score: number;
    updated_at: string;
}
interface FileItem {
    id: string;
    note_id?: string;
    user_id: string;
    file_name: string;
    file_type: string;
    storage_path: string;
    bucket: string;
    size_bytes?: number;
    status: number;
    created_at: string;
}
interface PresignResult {
    file_id: string;
    upload_url: string;
    expires_in: number;
}
interface Session {
    session_id: string;
    device_name: string;
    device_type: string;
    ip_address: string;
    user_agent: string;
    last_active_at: string;
    created_at: string;
    is_current: boolean;
}
interface AskRequest {
    question: string;
    conversation_id?: string;
    model_id?: string;
    top_k?: number;
    stream?: boolean;
}
interface AskResponse {
    answer: string;
    references?: Array<{
        note_id: string;
        note_title: string;
        text: string;
        chunk_index: number;
    }>;
    conversation_id: string;
    message_id?: string;
}
interface Conversation {
    id: string;
    user_id: string;
    title: string;
    model: string;
    created_at: string;
    updated_at: string;
}
interface Message {
    id: string;
    conversation_id: string;
    role: string;
    content: string;
    references?: unknown[];
    token_count?: number;
    model?: string;
    created_at: string;
}
interface AIModel {
    id: string;
    name: string;
    provider: string;
    type: string;
    endpoint: string;
    model_name: string;
    api_version?: string;
    max_tokens: number;
    temperature: number;
    top_p?: number;
    dimensions?: number;
    is_enabled: boolean;
    is_default: boolean;
    priority: number;
    created_at: string;
    updated_at: string;
}
interface WSMessage {
    type: 'sync' | 'awareness' | 'awareness_query' | 'presence' | 'error';
    payload: Record<string, any>;
}
interface AwarenessPayload {
    user_id: string;
    username: string;
    cursor?: {
        line: number;
        column: number;
    };
    selection?: {
        from: number;
        to: number;
    };
}
interface SyncPayload {
    update: string;
    version: number;
}

declare class AuthModule {
    private client;
    constructor(client: KrasisClient);
    getGitHubLoginUrl(state?: string): string;
    getGoogleLoginUrl(state?: string): string;
    githubCallback(code: string, state: string): Promise<{
        access_token: string;
        token_type: string;
        expires_in: number;
        user: User;
    }>;
    googleCallback(code: string, state: string): Promise<{
        access_token: string;
        token_type: string;
        expires_in: number;
        user: User;
    }>;
    logout(): Promise<void>;
    getMe(): Promise<User>;
}
declare class UserModule {
    private client;
    constructor(client: KrasisClient);
    getSessions(): Promise<{
        sessions: Session[];
    }>;
    deleteSession(sessionId: string): Promise<void>;
    updateProfile(username?: string, avatarUrl?: string): Promise<void>;
}

interface ListNotesParams {
    page?: number;
    size?: number;
    folder_id?: string;
    keyword?: string;
    sort?: string;
    order?: string;
}
interface CreateNoteParams {
    title: string;
    content?: string;
    folder_id?: string;
}
interface UpdateNoteParams {
    title?: string;
    content?: string;
    version: number;
}
declare class NotesModule {
    private client;
    constructor(client: KrasisClient);
    list(params?: ListNotesParams): Promise<{
        items: Note[];
        total: number;
        page: number;
        size: number;
    }>;
    get(id: string): Promise<Note>;
    create(params: CreateNoteParams): Promise<Note>;
    update(id: string, params: UpdateNoteParams): Promise<Note>;
    delete(id: string, permanent?: boolean): Promise<void>;
    getVersions(id: string): Promise<{
        items: NoteVersion[];
    }>;
    restoreVersion(id: string, version: number): Promise<void>;
}
declare class FoldersModule {
    private client;
    constructor(client: KrasisClient);
    list(): Promise<{
        items: Folder[];
    }>;
    create(name: string, parentId?: string, color?: string): Promise<Folder>;
    update(id: string, name?: string, parentId?: string, color?: string, sortOrder?: number): Promise<void>;
    delete(id: string): Promise<void>;
}
declare class ShareModule {
    private client;
    constructor(client: KrasisClient);
    create(noteId: string, params: {
        share_type?: string;
        permission?: string;
        password?: string;
        expires_at?: string;
    }): Promise<ShareStatus>;
    getStatus(noteId: string): Promise<ShareStatus>;
    delete(noteId: string): Promise<void>;
    access(token: string, password?: string): Promise<{
        note: {
            id: string;
            title: string;
            content: string;
        };
        permission: string;
    }>;
}

interface SearchParams {
    q: string;
    page?: number;
    size?: number;
    type?: 'notes' | 'files' | 'all';
}
declare class SearchModule {
    private client;
    constructor(client: KrasisClient);
    search(params: SearchParams): Promise<{
        items: SearchResult[];
        total: number;
        page: number;
        size: number;
    }>;
}
declare class FileModule {
    private client;
    constructor(client: KrasisClient);
    getPresignUrl(fileName: string, fileType: string, noteId?: string): Promise<{
        file_id: string;
        upload_url: string;
        expires_in: number;
    }>;
    confirmUpload(fileId: string, noteId?: string, metadata?: Record<string, unknown>): Promise<void>;
    delete(fileId: string): Promise<void>;
}

declare class AIModule {
    private client;
    constructor(client: KrasisClient);
    ask(params: AskRequest): Promise<AskResponse>;
    askStream(params: AskRequest, onToken: (token: string) => void, onDone?: () => void, onError?: (err: Error) => void): AbortController;
    listConversations(): Promise<Conversation[]>;
    getMessages(conversationId: string): Promise<Message[]>;
}

interface CollabConfig {
    wsBaseUrl: string;
    token: string;
}
type CollabEvent = {
    type: 'open';
} | {
    type: 'close';
    code: number;
    reason: string;
} | {
    type: 'error';
    error: Event;
} | {
    type: 'sync';
    payload: SyncPayload;
    userId: string;
} | {
    type: 'awareness';
    payload: AwarenessPayload;
} | {
    type: 'presence';
    users: Array<{
        user_id: string;
        username: string;
    }>;
};
type CollabEventHandler = (event: CollabEvent) => void;
declare class CollabModule {
    private ws;
    private handlers;
    private noteId;
    private config;
    private reconnectTimer;
    private reconnectAttempts;
    private maxReconnectAttempts;
    constructor(config: CollabConfig);
    on(handler: CollabEventHandler): () => void;
    connect(noteId: string): void;
    private doConnect;
    private handleMessage;
    sendSync(update: string, version: number): void;
    sendAwareness(payload: AwarenessPayload): void;
    sendPresenceQuery(): void;
    private send;
    private scheduleReconnect;
    disconnect(): void;
    private emit;
}

declare class SDKError extends Error {
    code: number;
    status: number;
    data?: unknown;
    constructor(message: string, code: number, status: number, data?: unknown);
}
declare class VersionConflictError extends SDKError {
    currentVersion: number;
    note: unknown;
    constructor(currentVersion: number, note: unknown);
}
declare class RateLimitError extends SDKError {
    retryAfter: number;
    constructor(retryAfter: number);
}

declare class KrasisSDK {
    private client;
    readonly auth: AuthModule;
    readonly users: UserModule;
    readonly notes: NotesModule;
    readonly folders: FoldersModule;
    readonly share: ShareModule;
    readonly search: SearchModule;
    readonly files: FileModule;
    readonly ai: AIModule;
    private _collab;
    constructor(config: KrasisConfig);
    get collab(): CollabModule;
    setToken(token: string): void;
    clearToken(): void;
    get isAuthenticated(): boolean;
}

export { type AIModel, type ApiResponse, type AskRequest, type AskResponse, type AwarenessPayload, type Conversation, type FileItem, type Folder, KrasisSDK, type Message, type Note, type NoteVersion, type PaginatedResponse, type PresignResult, RateLimitError, SDKError, type SearchResult, type Session, type ShareStatus, type SyncPayload, type User, VersionConflictError, type WSMessage, KrasisSDK as default };
