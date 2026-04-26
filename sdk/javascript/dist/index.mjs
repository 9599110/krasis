// src/error.ts
var SDKError = class extends Error {
  constructor(message, code, status, data) {
    super(message);
    this.name = "SDKError";
    this.code = code;
    this.status = status;
    this.data = data;
  }
};
var VersionConflictError = class extends SDKError {
  constructor(currentVersion, note) {
    super("\u7248\u672C\u51B2\u7A81", 1005, 409, { currentVersion, note });
    this.name = "VersionConflictError";
    this.currentVersion = currentVersion;
    this.note = note;
  }
};
var RateLimitError = class extends SDKError {
  constructor(retryAfter) {
    super("\u8BF7\u6C42\u8FC7\u4E8E\u9891\u7E41", 2001, 429, { retryAfter });
    this.name = "RateLimitError";
    this.retryAfter = retryAfter;
  }
};

// src/client.ts
var KrasisClient = class {
  constructor(config) {
    this._token = null;
    this.apiBaseUrl = config.apiBaseUrl.replace(/\/+$/, "");
    this.wsBaseUrl = config.wsBaseUrl || config.apiBaseUrl.replace(/^http/, "ws").replace(/\/+$/, "");
    this.clientId = config.clientId || this.generateClientId();
    this.storage = config.storage ?? (typeof localStorage !== "undefined" ? localStorage : null);
    if (this.storage) {
      this._token = this.storage.getItem("krasis_token");
    }
  }
  generateClientId() {
    return `sdk_${Math.random().toString(36).substring(2, 10)}`;
  }
  get token() {
    return this._token;
  }
  get isAuthenticated() {
    return this._token !== null;
  }
  setToken(token) {
    this._token = token;
    if (this.storage) {
      this.storage.setItem("krasis_token", token);
    }
  }
  clearToken() {
    this._token = null;
    if (this.storage) {
      this.storage.removeItem("krasis_token");
    }
  }
  async request(method, path, body, options) {
    const url = `${this.apiBaseUrl}${path}`;
    const headers = {
      "Content-Type": "application/json",
      ...options?.headers
    };
    if (this._token) {
      headers["Authorization"] = `Bearer ${this._token}`;
    }
    const init = {
      method,
      headers
    };
    if (body && method !== "GET") {
      init.body = JSON.stringify(body);
    }
    const res = await fetch(url, init);
    const json = await res.json();
    if (json.code !== 0) {
      if (json.code === 1005 && res.status === 409) {
        throw new VersionConflictError(
          json.data?.current_version || 0,
          json.data?.note
        );
      }
      if (json.code === 2001) {
        throw new RateLimitError(
          json.data?.retry_after || 60
        );
      }
      throw new SDKError(json.message, json.code, res.status, json.data);
    }
    return json.data;
  }
  get(path, options) {
    return this.request("GET", path, void 0, options);
  }
  post(path, body, options) {
    return this.request("POST", path, body, options);
  }
  put(path, body, options) {
    return this.request("PUT", path, body, options);
  }
  delete(path, options) {
    return this.request("DELETE", path, void 0, options);
  }
};

// src/auth.ts
var AuthModule = class {
  constructor(client) {
    this.client = client;
  }
  getGitHubLoginUrl(state) {
    const params = new URLSearchParams();
    if (state) params.set("state", state);
    return `${this.client.apiBaseUrl}/auth/github/login?${params}`;
  }
  getGoogleLoginUrl(state) {
    const params = new URLSearchParams();
    if (state) params.set("state", state);
    return `${this.client.apiBaseUrl}/auth/google/login?${params}`;
  }
  async githubCallback(code, state) {
    return this.client.get(`/auth/github/callback?code=${encodeURIComponent(code)}&state=${encodeURIComponent(state)}`);
  }
  async googleCallback(code, state) {
    return this.client.get(`/auth/google/callback?code=${encodeURIComponent(code)}&state=${encodeURIComponent(state)}`);
  }
  async logout() {
    try {
      await this.client.post("/auth/logout");
    } finally {
      this.client.clearToken();
    }
  }
  async getMe() {
    return this.client.get("/auth/me");
  }
};
var UserModule = class {
  constructor(client) {
    this.client = client;
  }
  async getSessions() {
    return this.client.get("/user/sessions");
  }
  async deleteSession(sessionId) {
    await this.client.delete(`/user/sessions/${sessionId}`);
  }
  async updateProfile(username, avatarUrl) {
    await this.client.put("/user/profile", { username, avatar_url: avatarUrl });
  }
};

// src/notes.ts
var NotesModule = class {
  constructor(client) {
    this.client = client;
  }
  async list(params) {
    const qs = new URLSearchParams();
    if (params?.page) qs.set("page", String(params.page));
    if (params?.size) qs.set("size", String(params.size));
    if (params?.folder_id) qs.set("folder_id", params.folder_id);
    if (params?.keyword) qs.set("keyword", params.keyword);
    if (params?.sort) qs.set("sort", params.sort);
    if (params?.order) qs.set("order", params.order);
    const query = qs.toString();
    return this.client.get(`/notes${query ? `?${query}` : ""}`);
  }
  async get(id) {
    return this.client.get(`/notes/${id}`);
  }
  async create(params) {
    return this.client.post("/notes", params);
  }
  async update(id, params) {
    return this.client.put(`/notes/${id}`, params, {
      headers: { "If-Match": String(params.version) }
    });
  }
  async delete(id, permanent = false) {
    await this.client.delete(`/notes/${id}${permanent ? "?permanent=true" : ""}`);
  }
  async getVersions(id) {
    return this.client.get(`/notes/${id}/versions`);
  }
  async restoreVersion(id, version) {
    await this.client.post(`/notes/${id}/versions/${version}/restore`);
  }
};
var FoldersModule = class {
  constructor(client) {
    this.client = client;
  }
  async list() {
    return this.client.get("/folders");
  }
  async create(name, parentId, color) {
    return this.client.post("/folders", { name, parent_id: parentId, color });
  }
  async update(id, name, parentId, color, sortOrder) {
    await this.client.put(`/folders/${id}`, { name, parent_id: parentId, color, sort_order: sortOrder });
  }
  async delete(id) {
    await this.client.delete(`/folders/${id}`);
  }
};
var ShareModule = class {
  constructor(client) {
    this.client = client;
  }
  async create(noteId, params) {
    return this.client.post(`/notes/${noteId}/share`, params);
  }
  async getStatus(noteId) {
    return this.client.get(`/notes/${noteId}/share`);
  }
  async delete(noteId) {
    await this.client.delete(`/notes/${noteId}/share`);
  }
  async access(token, password) {
    const headers = {};
    if (password) headers["X-Share-Password"] = password;
    return this.client.get(`/share/${token}`, { headers });
  }
};

// src/search.ts
var SearchModule = class {
  constructor(client) {
    this.client = client;
  }
  async search(params) {
    const qs = new URLSearchParams();
    qs.set("q", params.q);
    if (params.page) qs.set("page", String(params.page));
    if (params.size) qs.set("size", String(params.size));
    if (params.type) qs.set("type", params.type);
    return this.client.get(`/search?${qs.toString()}`);
  }
};
var FileModule = class {
  constructor(client) {
    this.client = client;
  }
  async getPresignUrl(fileName, fileType, noteId) {
    const qs = new URLSearchParams({ file_name: fileName, file_type: fileType });
    if (noteId) qs.set("note_id", noteId);
    return this.client.get(`/files/presign?${qs.toString()}`);
  }
  async confirmUpload(fileId, noteId, metadata) {
    await this.client.post("/files/confirm", { file_id: fileId, note_id: noteId, metadata });
  }
  async delete(fileId) {
    await this.client.delete(`/files/${fileId}`);
  }
};

// src/ai.ts
var AIModule = class {
  constructor(client) {
    this.client = client;
  }
  async ask(params) {
    return this.client.post("/ai/ask", params);
  }
  askStream(params, onToken, onDone, onError) {
    const controller = new AbortController();
    const url = `${this.client.apiBaseUrl}/ai/ask/stream`;
    fetch(url, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        ...this.client.token ? { Authorization: `Bearer ${this.client.token}` } : {}
      },
      body: JSON.stringify({ ...params, stream: true }),
      signal: controller.signal
    }).then(async (res) => {
      if (!res.ok) {
        onError?.(new Error(`HTTP ${res.status}`));
        return;
      }
      const reader = res.body?.getReader();
      if (!reader) {
        onError?.(new Error("ReadableStream not supported"));
        return;
      }
      const decoder = new TextDecoder();
      let buffer = "";
      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split("\n");
        buffer = lines.pop() || "";
        for (const line of lines) {
          if (line.startsWith("event: token")) {
            const nextLine = lines[lines.indexOf(line) + 1];
            if (nextLine?.startsWith("data: ")) {
              try {
                const data = JSON.parse(nextLine.slice(6));
                if (data.token) onToken(data.token);
              } catch {
              }
            }
          } else if (line.startsWith("event: done")) {
            onDone?.();
            return;
          }
        }
      }
      onDone?.();
    }).catch((err) => {
      if (err.name !== "AbortError") {
        onError?.(err);
      }
    });
    return controller;
  }
  async listConversations() {
    return this.client.get("/ai/conversations");
  }
  async getMessages(conversationId) {
    return this.client.get(`/ai/conversations/${conversationId}/messages`);
  }
};

// src/collab.ts
var CollabModule = class {
  constructor(config) {
    this.ws = null;
    this.handlers = [];
    this.noteId = null;
    this.reconnectTimer = null;
    this.reconnectAttempts = 0;
    this.maxReconnectAttempts = 5;
    this.config = config;
  }
  on(handler) {
    this.handlers.push(handler);
    return () => {
      this.handlers = this.handlers.filter((h) => h !== handler);
    };
  }
  connect(noteId) {
    this.noteId = noteId;
    this.reconnectAttempts = 0;
    this.doConnect();
  }
  doConnect() {
    if (this.ws) {
      this.ws.close();
    }
    const url = `${this.config.wsBaseUrl}/ws/collab?note_id=${this.noteId}&token=${this.config.token}`;
    this.ws = new WebSocket(url);
    this.ws.onopen = () => {
      this.reconnectAttempts = 0;
      this.emit({ type: "open" });
    };
    this.ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data);
        this.handleMessage(msg);
      } catch {
      }
    };
    this.ws.onclose = (event) => {
      this.emit({ type: "close", code: event.code, reason: event.reason });
      this.scheduleReconnect();
    };
    this.ws.onerror = (error) => {
      this.emit({ type: "error", error });
    };
  }
  handleMessage(msg) {
    switch (msg.type) {
      case "sync":
        this.emit({ type: "sync", payload: msg.payload, userId: msg.payload.user_id });
        break;
      case "awareness":
        this.emit({ type: "awareness", payload: msg.payload });
        break;
      case "presence":
        this.emit({ type: "presence", users: msg.payload.users });
        break;
      default:
        break;
    }
  }
  sendSync(update, version) {
    this.send({ type: "sync", payload: { update, version } });
  }
  sendAwareness(payload) {
    this.send({ type: "awareness", payload });
  }
  sendPresenceQuery() {
    this.send({ type: "awareness_query", payload: {} });
  }
  send(msg) {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(msg));
    }
  }
  scheduleReconnect() {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) return;
    this.reconnectAttempts++;
    const delay = Math.min(1e3 * Math.pow(2, this.reconnectAttempts), 3e4);
    this.reconnectTimer = setTimeout(() => this.doConnect(), delay);
  }
  disconnect() {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    this.reconnectAttempts = this.maxReconnectAttempts;
    this.ws?.close();
    this.ws = null;
  }
  emit(event) {
    for (const handler of this.handlers) {
      try {
        handler(event);
      } catch {
      }
    }
  }
};

// src/index.ts
var KrasisSDK = class {
  constructor(config) {
    this._collab = null;
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
  get collab() {
    if (!this._collab) {
      this._collab = new CollabModule({
        wsBaseUrl: this.client.wsBaseUrl,
        token: this.client.token || ""
      });
    }
    return this._collab;
  }
  setToken(token) {
    this.client.setToken(token);
  }
  clearToken() {
    this.client.clearToken();
  }
  get isAuthenticated() {
    return this.client.isAuthenticated;
  }
};
var index_default = KrasisSDK;
export {
  KrasisSDK,
  RateLimitError,
  SDKError,
  VersionConflictError,
  index_default as default
};
