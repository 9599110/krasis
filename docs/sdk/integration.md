# SDK 设计文档

## 1. SDK 概述

NoteKeeper SDK 提供多语言客户端库，简化与 NoteKeeper API 的集成。

### 1.1 支持的语言/平台

| 语言/平台 | 包名 | 版本 | 状态 |
|-----------|------|------|------|
| TypeScript/JavaScript | `@notekeeper/sdk-js` | 1.0.0 | 开发中 |
| Dart/Flutter | `notekeeper_sdk` | 1.0.0 | 开发中 |
| Python | `notekeeper-sdk` | 1.0.0 | 规划中 |
| Go | `github.com/notekeeper/sdk-go` | 1.0.0 | 规划中 |

### 1.2 核心功能

- **认证管理**: OAuth 2.0 登录、JWT Token 管理
- **笔记操作**: CRUD、版本控制、分享
- **文件上传**: 预签名 URL、流式上传、进度追踪
- **AI 问答**: 流式响应、上下文管理
- **全文检索**: 高级查询、高亮展示
- **实时协同**: WebSocket 连接、冲突处理
- **离线支持**: 本地缓存、同步队列

## 2. TypeScript/JavaScript SDK

### 2.1 安装

```bash
npm install @notekeeper/sdk-js
# 或
yarn add @notekeeper/sdk-js
# 或
pnpm add @notekeeper/sdk-js
```

### 2.2 初始化

```typescript
import { NoteKeeper } from '@notekeeper/sdk-js';

const client = new NoteKeeper({
  apiBaseUrl: 'https://api.notekeeper.com/api/v1',
  wsBaseUrl: 'wss://api.notekeeper.com',
  clientId: 'your-client-id',
  storage: localStorage,  // 可选：自定义存储适配器
});

// 初始化（恢复登录状态）
await client.initialize();
```

### 2.3 认证模块

```typescript
// 2.3.1 OAuth 登录
const authResult = await client.auth.loginWithOAuth({
  provider: 'github',  // 'github' | 'google'
  redirectUri: window.location.origin + '/callback',
});

// 登录成功后会回调 redirectUri

// 2.3.2 OAuth 回调处理
// 在回调页面调用：
await client.auth.handleOAuthCallback();

// 2.3.3 检查登录状态
const isAuthenticated = client.auth.isAuthenticated();

// 2.3.4 获取当前用户
const user = client.auth.getCurrentUser();
console.log(user.id, user.email, user.role);

// 2.3.5 登出
await client.auth.logout();

// 2.3.6 刷新 Token
await client.auth.refreshToken();
```

### 2.4 笔记模块

```typescript
// 2.4.1 获取笔记列表
const noteList = await client.notes.list({
  page: 1,
  size: 20,
  folderId: 'folder-uuid',  // 可选
  keyword: '搜索关键词',      // 可选
});

// 2.4.2 获取笔记详情
const note = await client.notes.get('note-uuid');
console.log(note.title, note.content, note.version);

// 2.4.3 创建笔记
const newNote = await client.notes.create({
  title: '我的新笔记',
  content: '# 标题\n\n内容...',
  folderId: 'folder-uuid',  // 可选
});

// 2.4.4 更新笔记（乐观锁）
try {
  const updatedNote = await client.notes.update('note-uuid', {
    title: '更新后的标题',
    content: '更新后的内容',
    version: note.version,  // 当前版本号
  });
  console.log('更新成功，新版本:', updatedNote.version);
} catch (error) {
  if (error instanceof VersionConflictError) {
    // 版本冲突处理
    console.log('服务器版本:', error.currentVersion);
    console.log('当前版本:', error.serverNote);
    // 合并或覆盖逻辑
  }
}

// 2.4.5 删除笔记
await client.notes.delete('note-uuid');

// 2.4.6 笔记版本历史
const versions = await client.notes.getVersions('note-uuid');
versions.forEach(v => {
  console.log(`版本 ${v.version}:`, v.changeSummary);
});

// 2.4.7 恢复历史版本
await client.notes.restoreVersion('note-uuid', 3);

// 2.4.8 监听笔记变化（实时同步）
const unsubscribe = client.notes.subscribe('note-uuid', (event) => {
  if (event.type === 'updated') {
    console.log('笔记已更新:', event.note);
  }
  if (event.type === 'deleted') {
    console.log('笔记已删除');
  }
});

// 取消订阅
unsubscribe();
```

### 2.5 文件模块

```typescript
// 2.5.1 获取预签名上传 URL
const presign = await client.files.getPresignUrl({
  fileName: 'image.jpg',
  fileType: 'image',
  noteId: 'note-uuid',
  size: 1024 * 1024 * 2,  // 2MB
});

// 2.5.2 上传文件
const progressCallback = (progress: number) => {
  console.log(`上传进度: ${progress}%`);
};

await client.files.upload({
  presignedUrl: presign.uploadUrl,
  file: fileObject,
  onProgress: progressCallback,
});

// 2.5.3 确认上传
await client.files.confirmUpload({
  fileId: presign.fileId,
  noteId: 'note-uuid',
  metadata: {
    width: 1920,
    height: 1080,
  },
});

// 2.5.4 删除文件
await client.files.delete('file-uuid');

// 2.5.5 获取文件访问 URL
const fileUrl = await client.files.getUrl('file-uuid');
// 返回带签名的临时访问 URL

// 2.5.6 下载文件
const blob = await client.files.download('file-uuid');
```

### 2.6 分享模块

```typescript
// 2.6.1 生成分享链接
const share = await client.shares.create('note-uuid', {
  shareType: 'link',
  permission: 'read',  // 'read' | 'write'
  password: 'optional-password',  // 可选
  expiresAt: new Date('2024-12-31'),
});

console.log('分享链接:', share.shareUrl);

// 2.6.2 访问分享的笔记
const note = await client.shares.access({
  token: 'share-token',
  password: 'password-if-required',
});

// 2.6.3 取消分享
await client.shares.delete('note-uuid');

// 2.6.4 列出我的分享
const shares = await client.shares.list();
```

### 2.7 AI 问答模块

```typescript
// 2.7.1 普通问答
const response = await client.ai.ask({
  question: '我的笔记中关于项目计划的内容有哪些？',
  conversationId: 'conv-uuid',  // 可选，关联对话
});

console.log('回答:', response.answer);
console.log('引用:', response.references);

// 2.7.2 流式问答
const stream = client.ai.askStream({
  question: '总结一下我的笔记要点',
  conversationId: 'conv-uuid',
});

let fullResponse = '';
for await (const chunk of stream) {
  if (chunk.type === 'token') {
    process.stdout.write(chunk.token);
    fullResponse += chunk.token;
  }
  if (chunk.type === 'reference') {
    console.log('引用:', chunk.reference);
  }
}

// 2.7.3 中止流式问答
stream.abort();

// 2.7.4 获取对话历史
const messages = await client.ai.getConversationMessages('conv-uuid');

// 2.7.5 列出对话列表
const conversations = await client.ai.listConversations();
```

### 2.8 搜索模块

```typescript
// 2.8.1 全文搜索
const results = await client.search.query({
  q: '项目计划',
  page: 1,
  size: 20,
  type: 'notes',  // 'notes' | 'files' | 'all'
});

results.items.forEach(item => {
  console.log('标题:', item.title);
  console.log('高亮片段:', item.highlights);
  console.log('相关性:', item.score);
});

// 2.8.2 搜索建议
const suggestions = await client.search.suggest('项目');
console.log('建议:', suggestions);
```

### 2.9 实时协同模块

```typescript
// 2.9.1 连接协同编辑
const collab = client.collab.connect('note-uuid', {
  onSync: (update) => {
    // 应用远程更新
    editor.applyUpdate(update);
  },
  onAwareness: (users) => {
    // 更新在线用户状态
    updateOnlineUsers(users);
  },
  onConflict: (conflict) => {
    // 处理冲突
    resolveConflict(conflict);
  },
});

// 2.9.2 发送本地更新
collab.sendUpdate(localUpdate);

// 2.9.3 更新光标位置
collab.updateAwareness({
  cursor: { line: 10, column: 5 },
  selection: { from: 100, to: 150 },
  username: 'Current User',
});

// 2.9.4 断开连接
collab.disconnect();

// 2.9.5 获取在线用户
const onlineUsers = collab.getOnlineUsers();
```

### 2.10 完整使用示例

```typescript
import { NoteKeeper } from '@notekeeper/sdk-js';

async function main() {
  // 1. 初始化客户端
  const client = new NoteKeeper({
    apiBaseUrl: 'https://api.notekeeper.com/api/v1',
    clientId: 'your-client-id',
  });

  // 2. OAuth 登录
  await client.auth.loginWithOAuth({
    provider: 'github',
    redirectUri: 'https://yourapp.com/callback',
  });

  // 3. 创建笔记
  const note = await client.notes.create({
    title: '会议纪要',
    content: '# 会议纪要\n\n时间：2024-01-01\n参会人：...\n\n## 讨论事项\n\n1. ...',
  });

  // 4. 上传附件
  const presign = await client.files.getPresignUrl({
    fileName: 'slide.pdf',
    fileType: 'document',
    noteId: note.id,
  });

  await client.files.upload({
    presignedUrl: presign.uploadUrl,
    file: fileInput.files[0],
    onProgress: (p) => console.log(`上传 ${p}%`),
  });

  await client.files.confirmUpload({
    fileId: presign.fileId,
    noteId: note.id,
  });

  // 5. AI 问答
  const response = await client.ai.ask({
    question: '这次会议的主要结论是什么？',
  });

  console.log('AI 回答:', response.answer);

  // 6. 分享笔记
  const share = await client.shares.create(note.id, {
    permission: 'read',
  });

  console.log('分享链接:', share.shareUrl);

  // 7. 全文检索
  const searchResults = await client.search.query({
    q: '会议纪要 项目',
  });

  console.log('搜索结果:', searchResults.items);
}

// 运行
main().catch(console.error);
```

## 3. Flutter SDK (Dart)

### 3.1 安装

```yaml
# pubspec.yaml
dependencies:
  notekeeper_sdk: ^1.0.0
```

### 3.2 初始化

```dart
import 'package:notekeeper_sdk/notekeeper_sdk.dart';

final sdk = NoteKeeperSDK(
  apiBaseUrl: 'https://api.notekeeper.com/api/v1',
  wsBaseUrl: 'wss://api.notekeeper.com',
  clientId: 'your-client-id',
);

// 初始化
await sdk.initialize();
```

### 3.3 认证

```dart
// OAuth 登录
final result = await sdk.auth.loginWithOAuth(OAuthProvider.github);

// 监听登录状态
sdk.auth.authStateStream.listen((state) {
  switch (state) {
    case AuthState.authenticated():
      print('已登录: ${state.user.email}');
    case AuthState.unauthenticated():
      print('未登录');
    case AuthState.loading():
      print('加载中...');
  }
});

// 登出
await sdk.auth.logout();
```

### 3.4 笔记操作

```dart
// 创建笔记
final note = await sdk.notes.create(
  title: '新笔记',
  content: '# 标题\n\n内容',
);

// 获取列表
final notes = await sdk.notes.list(
  page: 1,
  size: 20,
);

// 更新（乐观锁）
try {
  final updated = await sdk.notes.update(
    id: note.id,
    title: '更新标题',
    content: '更新内容',
    version: note.version,
  );
} on VersionConflictException catch (e) {
  // 处理冲突
  print('服务器版本: ${e.currentVersion}');
  // 合并或覆盖
}
```

### 3.5 AI 问答

```dart
// 普通问答
final response = await sdk.ai.ask('我的笔记中关于...');

// 流式问答
final stream = sdk.ai.askStream('总结一下...');

await for (final chunk in stream) {
  switch (chunk) {
    case TokenChunk():
      print(chunk.token, end: '');
    case ReferenceChunk():
      print('\n\n引用: ${chunk.noteTitle}');
    case DoneChunk():
      print('\n\n回答完成');
  }
}

// 中止
stream.abort();
```

## 4. SDK 配置选项

### 4.1 全局配置

```typescript
interface SDKConfig {
  // API 配置
  apiBaseUrl: string;
  wsBaseUrl?: string;
  
  // 认证配置
  clientId: string;
  redirectUri?: string;
  
  // 存储配置
  storage?: StorageAdapter;
  
  // 请求配置
  timeout?: number;
  retries?: number;
  
  // 日志配置
  logLevel?: 'debug' | 'info' | 'warn' | 'error';
  
  // 自定义配置
  headers?: Record<string, string>;
}
```

### 4.2 存储适配器

```typescript
interface StorageAdapter {
  get(key: string): Promise<string | null>;
  set(key: string, value: string): Promise<void>;
  remove(key: string): Promise<void>;
  clear(): Promise<void>;
}

// 内置适配器
// - BrowserStorage: localStorage / sessionStorage
// - ReactNativeStorage: AsyncStorage
// - NodeStorage: Memory / FileSystem
// - CustomStorage: 自定义实现
```

## 5. 错误处理

### 5.1 错误类型

```typescript
// SDK 错误基类
class SDKError extends Error {
  code: number;
  message: string;
  details?: any;
}

// 特定错误类型
class AuthenticationError extends SDKError { }
class PermissionError extends SDKError { }
class NotFoundError extends SDKError { }
class VersionConflictError extends SDKError {
  currentVersion: number;
  serverNote: Note;
}
class RateLimitError extends SDKError {
  retryAfter: number;
}
class NetworkError extends SDKError { }
```

### 5.2 错误处理示例

```typescript
import { NoteKeeper, SDKError } from '@notekeeper/sdk-js';

const client = new NoteKeeper({ ... });

try {
  await client.notes.update(id, data);
} catch (error) {
  if (error instanceof SDKError) {
    switch (error.code) {
      case 1002:  // 未认证
        // 跳转到登录页
        break;
      case 1003:  // 权限不足
        // 提示权限不足
        break;
      case 1005:  // 版本冲突
        // 合并处理
        break;
      case 2001:  // 限流
        // 等待后重试
        break;
      default:
        // 其他错误
        break;
    }
  }
}
```

## 6. TypeScript 类型定义

```typescript
// lib/types/index.ts

export interface User {
  id: string;
  email: string;
  username: string;
  avatarUrl?: string;
  role: 'admin' | 'member' | 'viewer';
  createdAt: string;
}

export interface Note {
  id: string;
  title: string;
  content: string;
  contentHtml?: string;
  ownerId: string;
  folderId?: string;
  version: number;
  isPublic: boolean;
  shareToken?: string;
  createdAt: string;
  updatedAt: string;
}

export interface FileInfo {
  id: string;
  fileName: string;
  fileType: FileType;
  mimeType: string;
  url: string;
  thumbnailUrl?: string;
  sizeBytes: number;
  width?: number;
  height?: number;
  durationSec?: number;
}

export interface AIResponse {
  answer: string;
  references: AIRefrence[];
  conversationId: string;
  messageId: string;
}

export interface SearchResult {
  type: 'note' | 'file';
  id: string;
  title: string;
  highlights: string[];
  score: number;
}

export interface Share {
  shareToken: string;
  shareUrl: string;
  permission: 'read' | 'write';
  expiresAt?: string;
}

export type FileType = 'image' | 'video' | 'audio' | 'archive' | 'document';
```
