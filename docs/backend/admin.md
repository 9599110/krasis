# 管理后台 API 规范

## 1. 概述

管理后台提供系统配置、用户管理、大模型配置等功能，仅限 `admin` 角色访问。

### 1.1 认证方式

所有管理接口需要携带有效的 JWT Token，且 `role` 必须为 `admin`。

---

## 2. 大模型配置管理

### 2.1 获取模型列表

```
GET /admin/ai/models
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
        "name": "gpt-4",
        "provider": "openai",
        "type": "llm",
        "endpoint": "https://api.openai.com/v1",
        "api_key": "sk-***",
        "model_name": "gpt-4",
        "max_tokens": 4096,
        "temperature": 0.7,
        "is_enabled": true,
        "is_default": true,
        "priority": 1,
        "created_at": "2024-01-01T00:00:00Z",
        "updated_at": "2024-01-02T00:00:00Z"
      }
    ]
  }
}
```

---

### 2.2 创建模型配置

```
POST /admin/ai/models
```

**Request Body**

```json
{
  "name": "gpt-4",
  "provider": "openai",
  "type": "llm",
  "endpoint": "https://api.openai.com/v1",
  "api_key": "sk-xxx",
  "model_name": "gpt-4",
  "max_tokens": 4096,
  "temperature": 0.7,
  "is_enabled": true,
  "is_default": true,
  "priority": 1
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | 配置名称（唯一标识） |
| provider | string | 是 | 提供商：openai / azure / anthropic / ollama / 本地 |
| type | string | 是 | 类型：llm / embedding |
| endpoint | string | 否 | API 端点 |
| api_key | string | 否 | API Key |
| model_name | string | 是 | 模型名称 |
| api_version | string | 否 | Azure API 版本 |
| max_tokens | int | 否 | 最大 token 数，默认 4096 |
| temperature | float | 否 | 温度参数，默认 0.7 |
| top_p | float | 否 | top_p 参数 |
| is_enabled | bool | 否 | 是否启用 |
| is_default | bool | 否 | 是否默认模型 |
| priority | int | 否 | 优先级，数字越小优先级越高 |

---

### 2.3 更新模型配置

```
PUT /admin/ai/models/{id}
```

**Request Body**

```json
{
  "name": "gpt-4-turbo",
  "is_enabled": true,
  "is_default": false,
  "temperature": 0.8,
  "priority": 2
}
```

---

### 2.4 删除模型配置

```
DELETE /admin/ai/models/{id}
```

**Response**

```json
{
  "code": 0,
  "message": "success"
}
```

---

### 2.5 测试模型连接

```
POST /admin/ai/models/{id}/test
```

**Response**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "success": true,
    "latency_ms": 150,
    "model_name": "gpt-4",
    "test_output": "连接正常"
  }
}
```

---

### 2.6 获取嵌入模型列表

```
GET /admin/ai/embedding-models
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
        "name": "text-embedding-ada-002",
        "provider": "openai",
        "endpoint": "https://api.openai.com/v1",
        "model_name": "text-embedding-ada-002",
        "dimensions": 1536,
        "is_enabled": true,
        "is_default": true,
        "created_at": "2024-01-01T00:00:00Z"
      }
    ]
  }
}
```

---

## 3. AI 系统配置

### 3.1 获取 AI 配置

```
GET /admin/ai/config
```

**Response**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "chunk_size": 500,
    "chunk_overlap": 50,
    "top_k": 5,
    "score_threshold": 0.7,
    "enable_rag": true,
    "max_context_tokens": 8000,
    "system_prompt": "你是一个智能笔记助手...",
    "enable_streaming": true
  }
}
```

---

### 3.2 更新 AI 配置

```
PUT /admin/ai/config
```

**Request Body**

```json
{
  "chunk_size": 500,
  "chunk_overlap": 50,
  "top_k": 5,
  "score_threshold": 0.7,
  "enable_rag": true,
  "max_context_tokens": 8000,
  "system_prompt": "你是一个智能笔记助手...",
  "enable_streaming": true
}
```

---

## 3.5 用户组管理

### 3.5.1 获取用户组列表

```
GET /admin/groups
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
        "name": "free",
        "description": "免费用户组",
        "is_default": true,
        "user_count": 100,
        "created_at": "2024-01-01T00:00:00Z"
      },
      {
        "id": "uuid",
        "name": "pro",
        "description": "专业版用户组",
        "is_default": false,
        "user_count": 50,
        "created_at": "2024-01-01T00:00:00Z"
      }
    ]
  }
}
```

---

### 3.5.2 获取组功能配置

```
GET /admin/groups/{id}/features
```

**Response**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [
      {
        "feature_key": "enable_sharing",
        "feature_value": {"value": true},
        "updated_at": "2024-01-01T00:00:00Z"
      },
      {
        "feature_key": "ai_ask_limit",
        "feature_value": {"value": 10, "period": "minute"},
        "updated_at": "2024-01-01T00:00:00Z"
      },
      {
        "feature_key": "version_history_limit",
        "feature_value": {"value": 10},
        "updated_at": "2024-01-01T00:00:00Z"
      }
    ]
  }
}
```

---

### 3.5.3 更新组功能配置

```
PUT /admin/groups/{id}/features
```

**Request Body**

```json
{
  "features": [
    {
      "feature_key": "ai_ask_limit",
      "feature_value": {"value": 60, "period": "minute"}
    },
    {
      "feature_key": "enable_ai",
      "feature_value": {"value": true}
    }
  ]
}
```

**说明**：功能开关按用户组控制，不同组可有不同限额。常用功能配置项：

| 功能项 | 说明 | 示例值 |
|--------|------|--------|
| enable_sharing | 是否启用分享功能 | `{"value": true}` |
| enable_ai | 是否启用 AI 功能 | `{"value": true}` |
| ai_ask_limit | AI 问答限流（次/周期） | `{"value": 10, "period": "minute"}` |
| version_history_limit | 版本历史保留数量 | `{"value": 10}` |
| storage_limit_bytes | 存储空间限制（字节） | `{"value": 10737418240}` |

---

## 4. 用户管理

### 4.1 获取用户列表

```
GET /admin/users
```

**Query Parameters**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码，默认 1 |
| size | int | 否 | 每页数量，默认 20 |
| role | string | 否 | 角色过滤：admin, member, viewer |
| status | int | 否 | 状态：0=禁用, 1=启用 |
| keyword | string | 否 | 搜索关键词（邮箱/用户名） |
| created_after | string | 否 | 注册时间筛选 |
| order_by | string | 否 | 排序字段：created_at, last_login_at, note_count |

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
        "note_count": 50,
        "storage_used_bytes": 104857600,
        "created_at": "2024-01-01T00:00:00Z",
        "last_login_at": "2024-01-02T00:00:00Z",
        "oauth_providers": ["github", "google"]
      }
    ],
    "total": 100,
    "page": 1,
    "size": 20
  }
}
```

---

### 4.2 获取用户详情

```
GET /admin/users/{id}
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
    "status": 1,
    "note_count": 50,
    "folder_count": 10,
    "file_count": 25,
    "storage_used_bytes": 104857600,
    "ai_conversation_count": 30,
    "created_at": "2024-01-01T00:00:00Z",
    "last_login_at": "2024-01-02T00:00:00Z",
    "oauth_providers": ["github", "google"],
    "sessions": [
      {
        "session_id": "uuid",
        "device_name": "iPhone 15 Pro",
        "ip_address": "192.168.1.1",
        "last_active_at": "2024-01-02T00:00:00Z"
      }
    ]
  }
}
```

---

### 4.3 创建用户（管理员）

```
POST /admin/users
```

**Request Body**

```json
{
  "email": "newuser@example.com",
  "username": "newuser",
  "password": "securepassword",
  "role": "member"
}
```

---

### 4.4 更新用户

```
PUT /admin/users/{id}
```

**Request Body**

```json
{
  "username": "newname",
  "role": "admin",
  "status": 1
}
```

---

### 4.5 删除用户

```
DELETE /admin/users/{id}
```

**Query Parameters**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| force | bool | 否 | 是否强制删除（包括所有数据），默认 false |

---

### 4.6 批量禁用用户

```
POST /admin/users/batch/disable
```

**Request Body**

```json
{
  "user_ids": ["uuid1", "uuid2"]
}
```

---

### 4.7 导出用户数据

```
POST /admin/users/export
```

**Request Body**

```json
{
  "format": "csv",
  "fields": ["email", "username", "role", "created_at", "note_count"],
  "filter": {
    "role": "member",
    "status": 1
  }
}
```

**Response**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "export_id": "uuid",
    "download_url": "/admin/exports/{export_id}/download"
  }
}
```

---

## 5. 系统统计

### 5.1 获取系统概览

```
GET /admin/stats/overview
```

**Response**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "total_users": 1000,
    "active_users_today": 150,
    "active_users_week": 400,
    "total_notes": 50000,
    "total_folders": 10000,
    "total_files": 20000,
    "total_storage_bytes": 107374182400,
    "ai_conversations_today": 500,
    "ai_tokens_used_today": 1000000
  }
}
```

---

### 5.2 获取用户增长统计

```
GET /admin/stats/users
```

**Query Parameters**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| period | string | 否 | 统计周期：day, week, month |
| start_date | string | 否 | 开始日期 |
| end_date | string | 否 | 结束日期 |

**Response**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [
      {
        "date": "2024-01-01",
        "new_users": 10,
        "active_users": 50,
        "total_users": 500
      }
    ]
  }
}
```

---

### 5.3 获取使用统计

```
GET /admin/stats/usage
```

**Response**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "notes_created_today": 100,
    "notes_updated_today": 200,
    "files_uploaded_today": 50,
    "storage_used_gb": 100.5,
    "ai_requests_today": 1000,
    "search_requests_today": 5000,
    "api_requests_today": 50000
  }
}
```

---

## 6. 系统配置

### 6.1 获取系统配置

```
GET /admin/config
```

**Response**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "site_name": "krasis",
    "site_url": "https://krasis.com",
    "allow_signup": true,
    "require_email_verification": true,
    "default_role": "member",
    "max_notes_per_user": -1,
    "max_storage_per_user_bytes": 10737418240,
    "max_file_size_bytes": 104857600,
    "allowed_file_types": ["image", "video", "audio", "document", "archive"],
    "session_duration_days": 7,
    "max_devices_per_user": 10,
    "enable_sharing": true,
    "enable_ai": true,
    "maintenance_mode": false
  }
}
```

---

### 6.2 更新系统配置

```
PUT /admin/config
```

**Request Body**

```json
{
  "site_name": "krasis",
  "allow_signup": true,
  "require_email_verification": false,
  "default_role": "member",
  "max_notes_per_user": 1000,
  "max_storage_per_user_bytes": 5368709120,
  "enable_sharing": true,
  "enable_ai": true,
  "maintenance_mode": false
}
```

---

## 7. OAuth 配置

### 7.1 获取 OAuth 配置

```
GET /admin/auth/oauth
```

**Response**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "github": {
      "enabled": true,
      "client_id": "xxx",
      "client_secret": "***",
      "redirect_uri": "https://api.krasis.com/auth/github/callback"
    },
    "google": {
      "enabled": true,
      "client_id": "xxx",
      "client_secret": "***",
      "redirect_uri": "https://api.krasis.com/auth/google/callback"
    }
  }
}
```

---

### 7.2 更新 OAuth 配置

```
PUT /admin/auth/oauth
```

**Request Body**

```json
{
  "github": {
    "enabled": true,
    "client_id": "new-client-id",
    "client_secret": "new-client-secret"
  },
  "google": {
    "enabled": false
  }
}
```

---

## 8. 操作日志

### 8.1 获取操作日志

```
GET /admin/logs
```

**Query Parameters**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码 |
| size | int | 否 | 每页数量 |
| action | string | 否 | 操作类型 |
| user_id | string | 否 | 用户 ID |
| start_date | string | 否 | 开始日期 |
| end_date | string | 否 | 结束日期 |

**Response**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [
      {
        "id": "uuid",
        "action": "user.role.updated",
        "target_type": "user",
        "target_id": "user-uuid",
        "admin_id": "admin-uuid",
        "admin_username": "admin",
        "changes": {
          "role": {"from": "member", "to": "admin"}
        },
        "ip_address": "192.168.1.1",
        "created_at": "2024-01-01T00:00:00Z"
      }
    ],
    "total": 1000,
    "page": 1,
    "size": 20
  }
}
```

---

## 9. 分享审核管理

### 9.1 获取分享待审核列表

```
GET /admin/shares/pending
```

**Query Parameters**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码，默认 1 |
| size | int | 否 | 每页数量，默认 20 |
| status | string | 否 | 状态：pending, approved, rejected |
| keyword | string | 否 | 搜索关键词（笔记标题/用户名） |
| start_date | string | 否 | 开始日期 |
| end_date | string | 否 | 结束日期 |
| order_by | string | 否 | 排序：created_at, updated_at |

**Response**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [
      {
        "id": "uuid",
        "share_token": "abc123xyz",
        "note_id": "uuid",
        "note_title": "笔记标题",
        "note_preview": "笔记内容预览...",
        "note_owner": {
          "id": "user-uuid",
          "username": "username",
          "email": "user@example.com"
        },
        "share_type": "link",
        "permission": "read",
        "password_protected": true,
        "expires_at": "2024-12-31T23:59:59Z",
        "status": "pending",
        "created_at": "2024-01-01T10:00:00Z",
        "note_content_snapshot": "分享时的内容快照（用于审核）"
      }
    ],
    "total": 100,
    "page": 1,
    "size": 20,
    "pending_count": 50
  }
}
```

---

### 9.2 获取分享详情（审核用）

```
GET /admin/shares/{id}
```

**Response**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "uuid",
    "share_token": "abc123xyz",
    "note_id": "uuid",
    "note_title": "笔记标题",
    "note_owner": {
      "id": "user-uuid",
      "username": "username",
      "email": "user@example.com"
    },
    "share_type": "link",
    "permission": "read",
    "password_protected": true,
    "expires_at": "2024-12-31T23:59:59Z",
    "status": "pending",
    "content_snapshot": "完整的笔记内容快照",
    "created_at": "2024-01-01T10:00:00Z",
    "reviewed_at": null,
    "reviewed_by": null,
    "rejection_reason": null,
    "audit_logs": [
      {
        "action": "share.created",
        "timestamp": "2024-01-01T10:00:00Z",
        "details": "用户创建分享"
      }
    ]
  }
}
```

---

### 9.3 审核通过

```
POST /admin/shares/{id}/approve
```

**Request Body**

```json
{
  "comment": "审核通过，内容合规"
}
```

**Response**

```json
{
  "code": 0,
  "message": "success"
}
```

---

### 9.4 审核拒绝

```
POST /admin/shares/{id}/reject
```

**Request Body**

```json
{
  "reason": "内容包含违规信息"
}
```

**Response**

```json
{
  "code": 0,
  "message": "success"
}
```

---

### 9.5 批量审核

```
POST /admin/shares/batch/review
```

**Request Body**

```json
{
  "share_ids": ["uuid1", "uuid2"],
  "action": "approve",
  "comment": "批量审核通过"
}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| share_ids | string[] | 是 | 分享 ID 列表 |
| action | string | 是 | approve / reject |
| reason | string | 否 | 拒绝原因（action=reject 时必填） |
| comment | string | 否 | 审核备注 |

---

### 9.6 获取分享统计

```
GET /admin/shares/stats
```

**Response**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "total_shares": 1000,
    "pending_shares": 50,
    "approved_shares": 900,
    "rejected_shares": 50,
    "today_pending": 10,
    "today_approved": 20,
    "today_rejected": 5,
    "avg_review_time_minutes": 30
  }
}
```

---

### 9.7 获取分享统计（按时段）

```
GET /admin/shares/stats/timeline
```

**Query Parameters**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| period | string | 否 | 统计周期：day, week, month |
| start_date | string | 否 | 开始日期 |
| end_date | string | 否 | 结束日期 |

**Response**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [
      {
        "date": "2024-01-01",
        "created": 20,
        "approved": 15,
        "rejected": 3,
        "pending": 2
      }
    ]
  }
}
```

---

### 9.8 撤回已发布的分享

```
DELETE /admin/shares/{id}/revoke
```

**说明**：管理员可直接撤销已通过的分享

**Response**

```json
{
  "code": 0,
  "message": "success"
}
```

---

### 9.9 分享内容复审

```
POST /admin/shares/{id}/re-review
```

**说明**：重新将已发布的分享设为待审核状态

**Request Body**

```json
{
  "reason": "需要重新审核"
}
```

**Response**

```json
{
  "code": 0,
  "message": "success"
}
```

**说明**：
- 被复审的分享状态从 `approved` 变更为 `pending`
- 分享者不会收到系统通知
- 分享者在查询分享状态时可看到状态变更
- 被拒绝的分享可直接重新提交（创建新分享记录）

---

## 10. 错误码（管理后台专用）

| 错误码 | HTTP Status | 说明 |
|--------|-------------|------|
| 1003 | 403 | 非管理员操作被拒绝 |
| 4001 | 400 | 无效的模型配置 |
| 4002 | 400 | 模型连接测试失败 |
| 4003 | 409 | 模型名称重复 |
| 5001 | 400 | 无法删除最后一个管理员 |
| 5002 | 400 | 无法禁用自己的账号 |

---

## 11. 管理后台前端页面（Flutter）

### 11.1 页面结构

```
admin/
├── lib/
│   ├── pages/
│   │   ├── dashboard_page.dart        # 仪表盘
│   │   ├── users_page.dart           # 用户管理
│   │   ├── user_detail_page.dart     # 用户详情
│   │   ├── ai_models_page.dart        # AI 模型配置
│   │   ├── ai_config_page.dart        # AI 系统配置
│   │   ├── shares_page.dart          # 分享审核列表
│   │   ├── share_review_page.dart     # 分享审核详情
│   │   ├── system_config_page.dart    # 系统配置
│   │   ├── oauth_config_page.dart     # OAuth 配置
│   │   └── logs_page.dart            # 操作日志
│   └── widgets/
│       ├── stats_card.dart           # 统计卡片
│       ├── model_form.dart           # 模型配置表单
│       ├── data_table.dart           # 数据表格
│       └── share_content_viewer.dart  # 分享内容预览
```

### 11.2 核心功能

| 页面 | 功能 |
|------|------|
| 仪表盘 | 系统概览、用户增长图、使用统计、待审核分享数 |
| 用户管理 | 用户列表、搜索、筛选、CRUD |
| 用户详情 | 用户信息、会话管理、数据统计 |
| AI 模型 | 模型列表、新增/编辑/删除/测试 |
| AI 配置 | RAG 参数、系统提示词、切片配置 |
| **分享审核** | 待审核列表、批量审核、通过/拒绝、统计 |
| 系统配置 | 注册设置、存储限制、功能开关 |
| OAuth 配置 | GitHub/Google OAuth 应用配置 |
| 操作日志 | 管理员操作记录查询 |

### 11.3 分享审核页面功能

| 功能 | 说明 |
|------|------|
| 待审核列表 | 显示所有待审核的分享，显示笔记标题、创建者、创建时间 |
| 内容预览 | 查看分享时的笔记内容快照 |
| 快速审核 | 快速通过/拒绝单个分享 |
| 批量审核 | 勾选多个分享后批量操作 |
| 拒绝原因 | 拒绝时必须填写原因 |
| 审核历史 | 查看该分享的审核状态变更记录 |
| 统计面板 | 显示待审核数、通过数、拒绝数、审核平均时长 |
