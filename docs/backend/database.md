# 数据库设计

## 1. 数据库选型

| 数据库 | 用途 | 版本 |
|--------|------|------|
| PostgreSQL | 主数据库，存储用户、笔记、角色等元数据 | 15+ |
| Redis | Session 存储、缓存、任务队列 | 7.x |
| Elasticsearch | 全文检索索引 | 8.x |
| Qdrant | 向量数据库，存储文档嵌入向量 | Latest |
| MinIO | 对象存储，文件存储 | Latest |

## 2. PostgreSQL 表结构

### 2.1 用户相关表

#### users - 用户表

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    username VARCHAR(100) NOT NULL,
    password_hash VARCHAR(255),  -- 本地密码（可选）
    avatar_url TEXT,
    status SMALLINT DEFAULT 1,   -- 1: 正常, 0: 禁用
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_status ON users(status);
```

#### user_oauth - OAuth 关联表

```sql
CREATE TABLE user_oauth (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL,           -- github, google
    provider_user_id VARCHAR(255) NOT NULL,
    provider_access_token TEXT,
    provider_refresh_token TEXT,
    token_expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ,
    UNIQUE(provider, provider_user_id)
);

CREATE INDEX idx_oauth_user ON user_oauth(user_id);
CREATE INDEX idx_oauth_provider ON user_oauth(provider, provider_user_id);
```

#### roles - 角色表

```sql
CREATE TABLE roles (
    id SMALLSERIAL PRIMARY KEY,
    name VARCHAR(50) UNIQUE NOT NULL,
    description TEXT,
    permissions JSONB DEFAULT '[]',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 初始化角色
INSERT INTO roles (name, description, permissions) VALUES
('admin', '超级管理员', '["*"]'),
('member', '普通用户', '["note:create", "note:read", "note:update", "note:delete", "note:share", "ai:ask", "search:query"]'),
('viewer', '访客', '["share:read"]');
```

#### user_roles - 用户角色关联表

```sql
CREATE TABLE user_roles (
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    role_id SMALLINT REFERENCES roles(id) ON DELETE CASCADE,
    assigned_at TIMESTAMPTZ DEFAULT NOW(),
    assigned_by UUID REFERENCES users(id),
    PRIMARY KEY(user_id, role_id)
);

CREATE INDEX idx_user_roles_user ON user_roles(user_id);
CREATE INDEX idx_user_roles_role ON user_roles(role_id);
```

### 2.2.1 groups - 用户组表（用于功能开关控制）

```sql
CREATE TABLE groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    description TEXT,
    is_default BOOLEAN DEFAULT false,     -- 是否为默认组
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ
);

-- 初始化默认组
INSERT INTO groups (name, description, is_default) VALUES
('free', '免费用户组', true),
('pro', '专业版用户组', false),
('enterprise', '企业版用户组', false);
```

### 2.2.2 user_groups - 用户组关联表

```sql
CREATE TABLE user_groups (
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    group_id UUID REFERENCES groups(id) ON DELETE CASCADE,
    joined_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY(user_id, group_id)
);

CREATE INDEX idx_user_groups_user ON user_groups(user_id);
CREATE INDEX idx_user_groups_group ON user_groups(group_id);
```

### 2.2.3 group_features - 组功能开关配置表

```sql
CREATE TABLE group_features (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    feature_key VARCHAR(100) NOT NULL,    -- enable_sharing, enable_ai, enable_ai_ask_limit 等
    feature_value JSONB NOT NULL,         -- 功能配置值
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(group_id, feature_key)
);

CREATE INDEX idx_group_features_group ON group_features(group_id);

-- 初始化功能配置（示例）
INSERT INTO group_features (group_id, feature_key, feature_value) VALUES
-- 免费用户组
((SELECT id FROM groups WHERE name='free'), 'enable_sharing', '{"value": true}'),
((SELECT id FROM groups WHERE name='free'), 'enable_ai', '{"value": true}'),
((SELECT id FROM groups WHERE name='free'), 'ai_ask_limit', '{"value": 10, "period": "minute"}'),
((SELECT id FROM groups WHERE name='free'), 'version_history_limit', '{"value": 10}'),
((SELECT id FROM groups WHERE name='free'), 'storage_limit_bytes', '{"value": 10737418240}'), -- 10GB
-- 专业版用户组
((SELECT id FROM groups WHERE name='pro'), 'enable_sharing', '{"value": true}'),
((SELECT id FROM groups WHERE name='pro'), 'enable_ai', '{"value": true}'),
((SELECT id FROM groups WHERE name='pro'), 'ai_ask_limit', '{"value": 60, "period": "minute"}'),
((SELECT id FROM groups WHERE name='pro'), 'version_history_limit', '{"value": 50}'),
((SELECT id FROM groups WHERE name='pro'), 'storage_limit_bytes', '{"value": 53687091200}'), -- 50GB
-- 企业版用户组
((SELECT id FROM groups WHERE name='enterprise'), 'enable_sharing', '{"value": true}'),
((SELECT id FROM groups WHERE name='enterprise'), 'enable_ai', '{"value": true}'),
((SELECT id FROM groups WHERE name='enterprise'), 'ai_ask_limit', '{"value": -1, "period": "unlimited"}'),
((SELECT id FROM groups WHERE name='enterprise'), 'version_history_limit', '{"value": -1}'),
((SELECT id FROM groups WHERE name='enterprise'), 'storage_limit_bytes', '{"value": 536870912000}'); -- 500GB
```

### 2.2.4 笔记相关表

#### notes - 笔记表

```sql
CREATE TABLE notes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(500) NOT NULL DEFAULT 'Untitled',
    content TEXT,
    content_html TEXT,           -- 富文本 HTML
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    folder_id UUID REFERENCES folders(id) ON DELETE SET NULL,
    version INT DEFAULT 1,
    is_public BOOLEAN DEFAULT false,
    is_deleted BOOLEAN DEFAULT false,    -- 软删除
    share_token VARCHAR(64) UNIQUE,
    share_expires_at TIMESTAMPTZ,
    view_count INT DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_notes_owner ON notes(owner_id);
CREATE INDEX idx_notes_folder ON notes(folder_id);
CREATE INDEX idx_notes_share_token ON notes(share_token);
CREATE INDEX idx_notes_public ON notes(is_public);
CREATE INDEX idx_notes_deleted ON notes(is_deleted);
CREATE INDEX idx_notes_updated ON notes(updated_at DESC);
```

#### folders - 文件夹表

```sql
CREATE TABLE folders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    parent_id UUID REFERENCES folders(id) ON DELETE CASCADE,
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    color VARCHAR(7),            -- #RRGGBB
    sort_order INT DEFAULT 0,
    is_deleted BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ
);

CREATE INDEX idx_folders_owner ON folders(owner_id);
CREATE INDEX idx_folders_parent ON folders(parent_id);
```

#### note_versions - 笔记版本历史

```sql
CREATE TABLE note_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    note_id UUID NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
    title VARCHAR(500),
    content TEXT,
    content_html TEXT,
    version INT NOT NULL,
    changed_by UUID REFERENCES users(id),
    change_summary TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_note_versions_note ON note_versions(note_id);
CREATE INDEX idx_note_versions_version ON note_versions(note_id, version DESC);
```

#### note_shares - 笔记分享记录

```sql
CREATE TABLE note_shares (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    note_id UUID NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
    share_token VARCHAR(64) UNIQUE NOT NULL,
    share_type VARCHAR(20) DEFAULT 'link',  -- link, email, user
    share_with_user_id UUID REFERENCES users(id),
    share_with_email VARCHAR(255),
    permission VARCHAR(20) DEFAULT 'read',  -- read, write
    password_hash VARCHAR(255),              -- 可选密码保护
    expires_at TIMESTAMPTZ,                   -- 用户自定过期时间，NULL表示永久
    status VARCHAR(20) DEFAULT 'pending',     -- pending:待审核, approved:已通过, rejected:已拒绝
    content_snapshot TEXT,                   -- 分享时保存的内容快照（用于审核）
    reviewed_at TIMESTAMPTZ,
    reviewed_by UUID REFERENCES users(id),
    rejection_reason TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    created_by UUID REFERENCES users(id)
);

CREATE INDEX idx_note_shares_token ON note_shares(share_token);
CREATE INDEX idx_note_shares_note ON note_shares(note_id);
CREATE INDEX idx_note_shares_status ON note_shares(status);
```

#### share_reviews - 分享审核记录表

```sql
CREATE TABLE share_reviews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    share_id UUID NOT NULL REFERENCES note_shares(id) ON DELETE CASCADE,
    action VARCHAR(20) NOT NULL,             -- approve, reject, re-review
    reason TEXT,                             -- 拒绝原因或复审原因
    reviewed_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_share_reviews_share ON share_reviews(share_id);
```

### 2.3 文件相关表

#### files - 文件表

```sql
CREATE TABLE files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    note_id UUID REFERENCES notes(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    file_name VARCHAR(255) NOT NULL,
    file_type VARCHAR(50),              -- image, video, audio, archive, document
    mime_type VARCHAR(100),
    storage_path VARCHAR(500) NOT NULL,
    bucket VARCHAR(100) DEFAULT 'notes',
    size_bytes BIGINT,
    width INT,                          -- 图片/视频
    height INT,
    duration_sec FLOAT,                -- 音视频
    duration_frames INT,                -- 视频帧数
    thumbnail_path VARCHAR(500),        -- 缩略图路径
    metadata JSONB DEFAULT '{}',        -- 扩展信息
    status SMALLINT DEFAULT 0,         -- 0: 上传中, 1: 完成, 2: 处理中, 3: 失败
    processed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ
);

CREATE INDEX idx_files_note ON files(note_id);
CREATE INDEX idx_files_user ON files(user_id);
CREATE INDEX idx_files_type ON files(file_type);
CREATE INDEX idx_files_status ON files(status);
```

#### file_processing_tasks - 文件处理任务

```sql
CREATE TABLE file_processing_tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_id UUID NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    task_type VARCHAR(50) NOT NULL,     -- thumbnail, transcode, extract_text
    status SMALLINT DEFAULT 0,          -- 0: 待处理, 1: 进行中, 2: 完成, 3: 失败
    progress INT DEFAULT 0,
    input_path VARCHAR(500),
    output_path VARCHAR(500),
    error_message TEXT,
    retry_count INT DEFAULT 0,
    max_retries INT DEFAULT 3,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_fpt_file ON file_processing_tasks(file_id);
CREATE INDEX idx_fpt_status ON file_processing_tasks(status);
```

### 2.4 AI 相关表

#### ai_conversations - AI 对话会话

```sql
CREATE TABLE ai_conversations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(255),
    model VARCHAR(50) DEFAULT 'gpt-4',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ
);

CREATE INDEX idx_ai_conv_user ON ai_conversations(user_id);
```

#### ai_messages - AI 消息记录

```sql
CREATE TABLE ai_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES ai_conversations(id) ON DELETE CASCADE,
    role VARCHAR(20) NOT NULL,           -- user, assistant, system
    content TEXT NOT NULL,
    references JSONB DEFAULT '[]',       -- 引用笔记片段
    token_count INT,
    model VARCHAR(50),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_ai_msg_conv ON ai_messages(conversation_id);
```

#### note_embeddings - 笔记向量索引

```sql
CREATE TABLE note_embeddings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    note_id UUID NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
    chunk_index INT NOT NULL,
    chunk_text TEXT NOT NULL,
    vector_id VARCHAR(255),              -- Qdrant 中的 ID
    token_count INT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ,
    UNIQUE(note_id, chunk_index)
);

CREATE INDEX idx_note_emb_note ON note_embeddings(note_id);
```

#### ai_models - AI 模型配置表

```sql
CREATE TABLE ai_models (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) UNIQUE NOT NULL,       -- 配置名称（唯一标识）
    provider VARCHAR(50) NOT NULL,           -- openai, azure, anthropic, ollama, 本地
    model_type VARCHAR(20) NOT NULL,         -- llm, embedding
    endpoint VARCHAR(500),                   -- API 端点
    api_key VARCHAR(500),                    -- API Key（加密存储）
    model_name VARCHAR(100) NOT NULL,        -- 实际模型名称
    api_version VARCHAR(50),                 -- Azure API 版本
    max_tokens INT DEFAULT 4096,
    temperature FLOAT DEFAULT 0.7,
    top_p FLOAT,
    dimensions INT,                           -- embedding 模型维度
    is_enabled BOOLEAN DEFAULT true,
    is_default BOOLEAN DEFAULT false,
    priority INT DEFAULT 100,                -- 优先级，数字越小优先级越高
    config JSONB DEFAULT '{}',               -- 其他配置
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ
);

CREATE INDEX idx_ai_models_type ON ai_models(model_type);
CREATE INDEX idx_ai_models_enabled ON ai_models(is_enabled);
CREATE INDEX idx_ai_models_default ON ai_models(is_default, is_enabled);
```

#### ai_config - AI 系统配置表

```sql
CREATE TABLE ai_config (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    config_key VARCHAR(100) UNIQUE NOT NULL,
    config_value JSONB NOT NULL,
    description TEXT,
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    updated_by UUID REFERENCES users(id)
);

-- 初始化默认配置
INSERT INTO ai_config (config_key, config_value, description) VALUES
('chunk_size', '{"value": 500}', '文本切片大小（token数）'),
('chunk_overlap', '{"value": 50}', '文本切片重叠大小'),
('top_k', '{"value": 5}', 'RAG 检索返回数量'),
('score_threshold', '{"value": 0.7}', '相似度阈值'),
('enable_rag', '{"value": true}', '是否启用 RAG'),
('max_context_tokens', '{"value": 8000}', '最大上下文 token 数'),
('system_prompt', '{"value": "你是一个智能笔记助手..."}', '系统提示词'),
('enable_streaming', '{"value": true}', '是否启用流式响应');
```

### 2.5 审计日志表

#### audit_logs - 审计日志

```sql
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id),
    action VARCHAR(100) NOT NULL,        -- login, logout, create_note, delete_note, etc.
    resource_type VARCHAR(50),           -- user, note, file, etc.
    resource_id UUID,
    ip_address INET,
    user_agent TEXT,
    details JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_audit_user ON audit_logs(user_id);
CREATE INDEX idx_audit_action ON audit_logs(action);
CREATE INDEX idx_audit_resource ON audit_logs(resource_type, resource_id);
CREATE INDEX idx_audit_time ON audit_logs(created_at DESC);
```

## 3. Redis 数据结构

### 3.1 Session 存储

```redis
# 单个 Session
session:{session_id} = HASH {
    user_id: "uuid",
    user_agent: "string",
    ip_address: "string",
    device_name: "string",
    last_active_at: "timestamp",
    created_at: "timestamp",
    expires_at: "timestamp"
}
TTL: 7 days

# 用户所有 Session 列表
user_sessions:{user_id} = SET {
    session_id_1,
    session_id_2,
    ...
}
```

### 3.2 JWT 黑名单

```redis
# 登出/强制下线后加入黑名单
jwt_blacklist:{jti} = "1"
TTL: 与 JWT 过期时间一致
```

### 3.3 笔记版本缓存

```redis
# 笔记最新版本号
note_version:{note_id} = INT
TTL: 1 hour

# 笔记乐观锁
note_lock:{note_id} = STRING (分布式锁)
TTL: 5 seconds
```

### 3.4 缓存

```redis
# 用户信息缓存
user_cache:{user_id} = HASH {
    id, email, username, avatar_url, role
}
TTL: 5 minutes

# 笔记列表缓存
notes_list:{user_id}:{page}:{size} = JSON string
TTL: 1 minute

# 权限缓存
user_permissions:{user_id} = SET { permission1, permission2, ... }
TTL: 10 minutes
```

### 3.5 任务队列

```redis
# Asynq 任务队列
asynq:queues:default = LIST
asynq:queues:critical = LIST
asynq:queues:low = LIST

# 延迟任务
asynq:scheduled = SORTED SET
```

## 4. Elasticsearch 索引

### 4.1 笔记全文检索索引

```json
{
  "settings": {
    "number_of_shards": 3,
    "number_of_replicas": 1,
    "analysis": {
      "analyzer": {
        "ik_analyzer": {
          "type": "custom",
          "tokenizer": "ik_max_word",
          "filter": ["lowercase"]
        }
      }
    }
  },
  "mappings": {
    "properties": {
      "id": { "type": "keyword" },
      "title": {
        "type": "text",
        "analyzer": "ik_analyzer",
        "fields": {
          "keyword": { "type": "keyword" }
        }
      },
      "content": {
        "type": "text",
        "analyzer": "ik_analyzer"
      },
      "owner_id": { "type": "keyword" },
      "folder_id": { "type": "keyword" },
      "tags": { "type": "keyword" },
      "created_at": { "type": "date" },
      "updated_at": { "type": "date" },
      "is_public": { "type": "boolean" },
      "file_contents": {
        "type": "nested",
        "properties": {
          "file_id": { "type": "keyword" },
          "content": { "type": "text", "analyzer": "ik_analyzer" }
        }
      }
    }
  }
}
```

### 4.2 用户操作日志索引

```json
{
  "settings": {
    "number_of_shards": 1,
    "number_of_replicas": 1
  },
  "mappings": {
    "properties": {
      "timestamp": { "type": "date" },
      "user_id": { "type": "keyword" },
      "action": { "type": "keyword" },
      "resource": { "type": "keyword" },
      "query": { "type": "text" },
      "result_count": { "type": "integer" },
      "latency_ms": { "type": "integer" }
    }
  }
}
```

## 5. Qdrant Collection

### 5.1 笔记向量集合

```json
{
  "name": "note_chunks",
  "vectors": {
    "size": 768,  // 取决于嵌入模型
    "distance": "Cosine"
  },
  "hnsw_config": {
    "m": 16,
    "ef_construct": 100
  },
  "payload_schema": {
    "note_id": "keyword",
    "user_id": "keyword",
    "chunk_index": "integer",
    "chunk_text": "text",
    "created_at": "datetime"
  }
}
```

## 6. MinIO 存储结构

```
notes-bucket/
├── {year}/{month}/{day}/
│   ├── {uuid}/
│   │   ├── original/
│   │   │   └── {filename}
│   │   ├── thumbnails/
│   │   │   ├── small_{filename}.webp
│   │   │   ├── medium_{filename}.webp
│   │   │   └── large_{filename}.webp
│   │   ├── processed/
│   │   │   ├── {filename}.mp4
│   │   │   └── {filename}.m3u8
│   │   └── extracted/
│   │       └── {extracted_files...}
│   └── avatars/
│       └── {user_id}.{ext}
```

## 7. 数据库迁移策略

### 7.1 迁移工具

使用 `golang-migrate` 进行数据库版本管理：

```bash
# 创建迁移文件
migrate create -ext sql -dir migrations -seq create_users_table
```

### 7.2 迁移文件命名

```
000001_create_users_table.up.sql
000001_create_users_table.down.sql
000002_create_notes_table.up.sql
000002_create_notes_table.down.sql
...
```

### 7.3 迁移执行

```bash
# 开发环境
migrate -path migrations -database $DB_DSN up

# 生产环境（需确认）
migrate -path migrations -database $DB_DSN up 1
```

## 8. 备份策略

| 备份类型 | 频率 | 保留时间 | 说明 |
|----------|------|----------|------|
| 全量备份 | 每天 | 30 天 | 凌晨 3:00 |
| 增量备份 | 每 6 小时 | 7 天 | PostgreSQL WAL |
| 实时复制 | 持续 | N/A | 主从同步 |
| 文件备份 | 每天 | 7 天 | MinIO 对象 |
