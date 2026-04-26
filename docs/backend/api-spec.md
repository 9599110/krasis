# API 接口规范

## 1. 概述

### 1.1 基本信息

- **Base URL**: `https://api.notekeeper.com/api/v1`
- **Content-Type**: `application/json`
- **认证方式**: Bearer Token (JWT)

### 1.2 通用响应格式

```json
{
  "code": 0,
  "message": "success",
  "data": {}
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| code | int | 状态码，0=成功，非0=失败 |
| message | string | 描述信息 |
| data | object | 响应数据 |

### 1.3 错误码定义

| 错误码 | HTTP Status | 说明 |
|--------|-------------|------|
| 0 | 200 | 成功 |
| 1001 | 400 | 参数错误 |
| 1002 | 401 | 未认证 |
| 1003 | 403 | 权限不足 |
| 1004 | 404 | 资源不存在 |
| 1005 | 409 | 资源冲突（如版本冲突） |
| 1006 | 422 | 业务逻辑错误 |
| 2001 | 429 | 请求过于频繁 |
| 3001 | 500 | 服务器内部错误 |
| 3002 | 503 | 服务暂不可用 |

### 1.4 分页响应格式

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [],
    "total": 100,
    "page": 1,
    "size": 20,
    "has_more": true
  }
}
```

---

## 2. 认证相关 API

### 2.1 OAuth 登录

#### GitHub OAuth

```
GET /auth/github/login
```

重定向到 GitHub 授权页面。

**Query Parameters**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| redirect_uri | string | 否 | 授权后跳转 URI |

**Response**: 302 Redirect 到 GitHub 授权页

---

```
GET /auth/github/callback
```

GitHub 授权回调。

**Query Parameters**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| code | string | 是 | 授权码 |
| state | string | 是 | 防止 CSRF 的状态码 |

**Response**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "access_token": "eyJhbG...",
    "token_type": "Bearer",
    "expires_in": 604800,
    "user": {
      "id": "uuid",
      "email": "user@example.com",
      "username": "username",
      "avatar_url": "https://...",
      "role": "member"
    }
  }
}
```

---

#### Google OAuth

```
GET /auth/google/login
GET /auth/google/callback
```

与 GitHub 类似，不再赘述。

---

### 2.2 本地登录

```
POST /auth/login
```

**Request Body**

```json
{
  "email": "user@example.com",
  "password": "password123"
}
```

**Response**: 同 OAuth 回调

---

### 2.3 登出

```
POST /auth/logout
```

**Headers**

```
Authorization: Bearer <token>
```

**Response**

```json
{
  "code": 0,
  "message": "success"
}
```

---

### 2.4 获取当前用户信息

```
GET /auth/me
```

**Response**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "uuid",
    "email": "user@example.com",
    "username": "username",
    "avatar_url": "https://...",
    "role": "member",
    "created_at": "2024-01-01T00:00:00Z"
  }
}
```

---

## 3. 用户管理 API

### 3.1 获取所有登录设备

```
GET /user/sessions
```

**Response**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "sessions": [
      {
        "session_id": "uuid",
        "device_name": "iPhone 15 Pro",
        "device_type": "mobile",
        "ip_address": "192.168.1.1",
        "user_agent": "Mozilla/5.0...",
        "last_active_at": "2024-01-01T12:00:00Z",
        "created_at": "2024-01-01T10:00:00Z",
        "is_current": true
      }
    ]
  }
}
```

---

### 3.2 强制下线设备

```
DELETE /user/sessions/{session_id}
```

**Path Parameters**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| session_id | string | 是 | Session ID |

**Response**

```json
{
  "code": 0,
  "message": "success"
}
```

---

### 3.3 更新用户资料

```
PUT /user/profile
```

**Request Body**

```json
{
  "username": "new_username",
  "avatar_url": "https://..."
}
```

---

### 3.4 管理员：获取用户列表

```
GET /admin/users
```

**Query Parameters**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码，默认 1 |
| size | int | 否 | 每页数量，默认 20 |
| role | string | 否 | 角色过滤 |
| keyword | string | 否 | 搜索关键词 |

**Response**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [
      {
        "id": "uuid",
        "email": "user@example.com",
        "username": "username",
        "avatar_url": "https://...",
        "role": "member",
        "status": 1,
        "created_at": "2024-01-01T00:00:00Z",
        "last_login_at": "2024-01-02T00:00:00Z"
      }
    ],
    "total": 100,
    "page": 1,
    "size": 20
  }
}
```

---

### 3.5 管理员：修改用户角色

```
PUT /admin/users/{user_id}/role
```

**Request Body**

```json
{
  "role": "admin"
}
```

---

### 3.6 管理员：禁用/启用用户

```
PUT /admin/users/{user_id}/status
```

**Request Body**

```json
{
  "status": 0
}
```

---

## 4. 笔记管理 API

### 4.1 获取笔记列表

```
GET /notes
```

**Query Parameters**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码，默认 1 |
| size | int | 否 | 每页数量，默认 20 |
| folder_id | string | 否 | 文件夹 ID |
| keyword | string | 否 | 搜索关键词 |
| sort | string | 否 | 排序：updated_at, created_at, title |
| order | string | 否 | 排序方向：asc, desc |

**Response**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [
      {
        "id": "uuid",
        "title": "笔记标题",
        "content_preview": "笔记内容预览...",
        "owner_id": "uuid",
        "folder_id": "uuid",
        "version": 3,
        "is_public": false,
        "share_token": "abc123",
        "view_count": 10,
        "created_at": "2024-01-01T00:00:00Z",
        "updated_at": "2024-01-02T00:00:00Z"
      }
    ],
    "total": 100,
    "page": 1,
    "size": 20
  }
}
```

---

### 4.2 获取笔记详情

```
GET /notes/{id}
```

**Path Parameters**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | 笔记 ID |

**Response**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "uuid",
    "title": "笔记标题",
    "content": "笔记内容（Markdown）",
    "content_html": "笔记内容（HTML）",
    "owner_id": "uuid",
    "folder_id": "uuid",
    "version": 3,
    "is_public": false,
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-02T00:00:00Z",
    "files": [
      {
        "id": "uuid",
        "file_name": "image.jpg",
        "file_type": "image",
        "url": "https://...",
        "thumbnail_url": "https://..."
      }
    ]
  }
}
```

---

### 4.3 创建笔记

```
POST /notes
```

**Request Body**

```json
{
  "title": "新笔记",
  "content": "笔记内容",
  "folder_id": "uuid"
}
```

**Response**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "uuid",
    "title": "新笔记",
    "version": 1,
    "created_at": "2024-01-01T00:00:00Z"
  }
}
```

---

### 4.4 更新笔记（乐观锁）

```
PUT /notes/{id}
```

**Headers**

```
Authorization: Bearer <token>
If-Match: <version>
```

**Path Parameters**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | 笔记 ID |

**Request Body**

```json
{
  "title": "更新后的标题",
  "content": "更新后的内容",
  "version": 2
}
```

**Success Response (200)**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "uuid",
    "title": "更新后的标题",
    "version": 3,
    "updated_at": "2024-01-02T00:00:00Z"
  }
}
```

**Conflict Response (409)**

```json
{
  "code": 1005,
  "message": "版本冲突",
  "data": {
    "current_version": 4,
    "note": {
      "id": "uuid",
      "title": "最新标题",
      "content": "最新内容",
      "version": 4
    }
  }
}
```

---

### 4.5 删除笔记

```
DELETE /notes/{id}
```

**Query Parameters**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| permanent | bool | 否 | 是否永久删除，默认 false（软删除） |

---

### 4.6 笔记版本历史

```
GET /notes/{id}/versions
```

**Response**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [
      {
        "id": "uuid",
        "version": 3,
        "title": "标题",
        "change_summary": "修改了内容",
        "changed_by": {
          "id": "uuid",
          "username": "username"
        },
        "created_at": "2024-01-01T00:00:00Z"
      }
    ]
  }
}
```

---

### 4.7 恢复历史版本

```
POST /notes/{id}/versions/{version}/restore
```

---

## 5. 分享 API

### 5.1 生成分享链接

```
POST /notes/{id}/share
```

**Request Body**

```json
{
  "share_type": "link",
  "permission": "read",
  "password": "optional_password",
  "expires_at": "2024-12-31T23:59:59Z"
}
```

**Response**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "share_token": "abc123xyz",
    "share_url": "https://notekeeper.com/share/abc123xyz",
    "expires_at": "2024-12-31T23:59:59Z",
    "status": "pending",
    "status_description": "分享待审核，审核通过后可被访问"
  }
}
```

**说明**：
- `status` 字段：`pending`(待审核) | `approved`(已通过) | `rejected`(已拒绝)
- 新创建的分享默认状态为 `pending`，需要管理员审核
- 分享创建者可通过 `GET /notes/{id}/share` 查询审核状态
- `expires_at` 为用户自定过期时间，`null` 表示永久分享
- 审核通过后笔记内容变更不影响已发布的分享（使用内容快照）

---

### 5.2 获取分享状态

```
GET /notes/{id}/share
```

**Response**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "share_token": "abc123xyz",
    "share_url": "https://notekeeper.com/share/abc123xyz",
    "permission": "read",
    "password_protected": true,
    "expires_at": "2024-12-31T23:59:59Z",
    "status": "approved",
    "status_description": "已通过",
    "reviewed_at": "2024-01-02T12:00:00Z",
    "reviewed_by": {
      "id": "admin-uuid",
      "username": "admin"
    },
    "rejection_reason": null,
    "created_at": "2024-01-01T10:00:00Z"
  }
}
```

---

### 5.3 访问分享笔记

```
GET /share/{token}
```

**Headers** (可选)

```
Authorization: Bearer <token>
X-Share-Password: <password>
```

**Response (审核通过)**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "note": {
      "id": "uuid",
      "title": "分享的笔记",
      "content": "笔记内容"
    },
    "permission": "read"
  }
}
```

**需要密码 Response (401)**

```json
{
  "code": 1002,
  "message": "需要密码访问",
  "data": {
    "require_password": true
  }
}
```

**待审核 Response (403)**

```json
{
  "code": 1003,
  "message": "分享待审核，暂不可访问",
  "data": {
    "status": "pending"
  }
}
```

**已拒绝 Response (403)**

```json
{
  "code": 1003,
  "message": "分享未通过审核",
  "data": {
    "status": "rejected",
    "rejection_reason": "内容包含违规信息"
  }
}
```

---

### 5.4 取消分享

```
DELETE /notes/{id}/share
```

---

## 6. AI 知识库 API

### 6.1 提问

```
POST /ai/ask
```

**Request Body**

```json
{
  "question": "用户的问题是什么？",
  "conversation_id": "uuid",
  "model": "gpt-4",
  "stream": true
}
```

**Response (流式)**

```
HTTP/1.1 200 OK
Content-Type: text/event-stream

event: token
data: {"token": "你"}

event: token
data: {"token": "好"}

event: reference
data: {"note_id": "uuid", "note_title": "相关笔记", "chunk_text": "..."}

event: done
data: {"answer": "完整答案", "references": [...]}
```

**Response (非流式)**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "answer": "AI 生成的答案",
    "references": [
      {
        "note_id": "uuid",
        "note_title": "相关笔记标题",
        "chunk_text": "引用的内容片段",
        "relevance_score": 0.95
      }
    ],
    "conversation_id": "uuid",
    "message_id": "uuid"
  }
}
```

---

### 6.2 获取对话历史

```
GET /ai/conversations
GET /ai/conversations/{id}/messages
```

---

## 7. 全文检索 API

### 7.1 搜索

```
GET /search
```

**Query Parameters**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| q | string | 是 | 搜索关键词 |
| page | int | 否 | 页码，默认 1 |
| size | int | 否 | 每页数量，默认 20 |
| type | string | 否 | 搜索类型：notes, files, all |

**Response**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [
      {
        "type": "note",
        "id": "uuid",
        "title": "匹配的笔记",
        "highlights": ["...匹配<em>关键词</em>..."],
        "score": 0.95,
        "updated_at": "2024-01-01T00:00:00Z"
      }
    ],
    "total": 50,
    "page": 1,
    "size": 20,
    "took_ms": 15
  }
}
```

---

## 8. 文件上传 API

### 8.1 获取预签名上传 URL

```
GET /upload/presign
```

**Query Parameters**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| file_name | string | 是 | 文件名 |
| file_type | string | 是 | 文件类型：image, video, audio, archive, document |
| note_id | string | 否 | 关联的笔记 ID |
| size | int | 否 | 文件大小（字节） |

**Response**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "file_id": "uuid",
    "upload_url": "https://minio.../presigned-put-url",
    "expires_in": 300
  }
}
```

---

### 8.2 确认上传完成

```
POST /upload/confirm
```

**Request Body**

```json
{
  "file_id": "uuid",
  "note_id": "uuid",
  "metadata": {
    "width": 1920,
    "height": 1080,
    "duration_sec": 120.5
  }
}
```

---

### 8.3 删除文件

```
DELETE /upload/{file_id}
```

---

## 9. 文件夹管理 API

```
GET    /folders                    # 获取文件夹列表
POST   /folders                    # 创建文件夹
PUT    /folders/{id}              # 更新文件夹
DELETE /folders/{id}              # 删除文件夹
```

---

## 10. WebSocket API

### 10.1 协同编辑

```
WS /ws/collab?note_id={note_id}&token={jwt}
```

**消息格式**

```json
{
  "type": "sync|awareness|presence",
  "payload": {}
}
```

**同步消息 (sync)**

```json
{
  "type": "sync",
  "payload": {
    "update": "base64-encoded-update",
    "version": 5
  }
}
```

**在线用户 (awareness)**

```json
{
  "type": "awareness",
  "payload": {
    "user_id": "uuid",
    "username": "username",
    "cursor": { "line": 10, "column": 5 },
    "selection": { "from": 100, "to": 150 }
  }
}
```

---

## 11. 健康检查 API

```
GET /health
```

**Response**

```json
{
  "status": "healthy",
  "version": "1.0.0",
  "uptime": 3600,
  "services": {
    "database": "ok",
    "redis": "ok",
    "elasticsearch": "ok",
    "qdrant": "ok"
  }
}
```

---

## 12. API 限流

| 等级 | 限制 | 说明 |
|------|------|------|
| 普通用户 | 100 次/分钟 | 普通 API |
| 付费用户 | 1000 次/分钟 | 普通 API |
| AI 问答 | 按组控制 | 由后台 `group_features` 配置，默认 10 次/分钟 |
| 文件上传 | 10 次/分钟 | 独立限制 |

**限流 Response (429)**

```json
{
  "code": 2001,
  "message": "请求过于频繁",
  "data": {
    "retry_after": 60
  }
}
```
