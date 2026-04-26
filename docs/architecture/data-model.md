# 数据模型设计

## 1. 领域模型概览

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           领域模型关系图                                  │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│     ┌─────────┐         ┌─────────┐         ┌─────────┐               │
│     │  User   │────────<│UserRole │         │   AI    │               │
│     └────┬────┘         └─────────┘         │Conversation│            │
│          │                  │                └────┬─────┘               │
│          │                  │                     │                     │
│          │    ┌─────────────┼─────────────┐       │                     │
│          │    │             │             │       │                     │
│          │    ▼             ▼             ▼       ▼                     │
│          │ ┌──────┐    ┌────────┐    ┌───────┐ ┌──────────┐           │
│          └─│ Note │    │ Folder │    │ Share │ │AIMessage │           │
│            └──┬───┘    └────────┘    └───┬───┘ └────┬─────┘           │
│               │                          │          │                   │
│               │         ┌────────────────┘          │                   │
│               │         │                          │                   │
│               ▼         ▼                          ▼                   │
│           ┌────────┐  ┌────────┐  ┌────────────┐                     │
│           │  File  │  │  Note  │  │   Note     │                     │
│           │        │  │Version │  │Embedding   │                     │
│           └────────┘  └────────┘  └────────────┘                     │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

## 2. 核心实体定义

### 2.1 User (用户)

```go
// internal/user/model.go

type User struct {
    ID           uuid.UUID   `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
    Email        string      `json:"email" gorm:"type:varchar(255);unique_index;not null"`
    Username     string      `json:"username" gorm:"type:varchar(100);not null"`
    PasswordHash string      `json:"-" gorm:"type:varchar(255)"`
    AvatarURL    string      `json:"avatar_url" gorm:"type:text"`
    Status       int8        `json:"status" gorm:"default:1"` // 1: 正常, 0: 禁用
    CreatedAt    time.Time   `json:"created_at"`
    UpdatedAt    time.Time   `json:"updated_at"`
    
    // 关联
    OAuths       []UserOAuth `json:"oauths,omitempty" gorm:"foreignKey:UserID"`
    Roles        []Role      `json:"roles,omitempty" gorm:"many2many:user_roles;"`
    Notes        []Note      `json:"notes,omitempty" gorm:"foreignKey:OwnerID"`
}

func (u *User) IsAdmin() bool {
    for _, role := range u.Roles {
        if role.Name == "admin" {
            return true
        }
    }
    return false
}

func (u *User) HasPermission(permission string) bool {
    if u.IsAdmin() {
        return true
    }
    for _, role := range u.Roles {
        for _, p := range role.Permissions {
            if p == permission || p == "*" {
                return true
            }
        }
    }
    return false
}
```

### 2.2 Note (笔记)

```go
// internal/note/model.go

type Note struct {
    ID           uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
    Title        string     `json:"title" gorm:"type:varchar(500);not null;default:'Untitled'"`
    Content      string     `json:"content" gorm:"type:text"`
    ContentHTML  string     `json:"content_html" gorm:"type:text"`
    OwnerID      uuid.UUID  `json:"owner_id" gorm:"type:uuid;not null;index"`
    FolderID     *uuid.UUID `json:"folder_id" gorm:"type:uuid;index"`
    Version      int        `json:"version" gorm:"default:1"`
    IsPublic     bool       `json:"is_public" gorm:"default:false"`
    IsDeleted    bool       `json:"is_deleted" gorm:"default:false;index"`
    ShareToken   string     `json:"share_token,omitempty" gorm:"type:varchar(64);unique_index"`
    ViewCount    int        `json:"view_count" gorm:"default:0"`
    CreatedAt    time.Time  `json:"created_at"`
    UpdatedAt    time.Time  `json:"updated_at"`
    DeletedAt    *time.Time `json:"deleted_at,omitempty"`
    
    // 关联
    Owner        *User      `json:"owner,omitempty" gorm:"foreignKey:OwnerID"`
    Folder       *Folder     `json:"folder,omitempty" gorm:"foreignKey:FolderID"`
    Files        []File      `json:"files,omitempty" gorm:"foreignKey:NoteID"`
    Versions     []NoteVersion `json:"versions,omitempty" gorm:"foreignKey:NoteID"`
    Embeddings   []NoteEmbedding `json:"embeddings,omitempty" gorm:"foreignKey:NoteID"`
}

type NoteVersion struct {
    ID            uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
    NoteID        uuid.UUID `json:"note_id" gorm:"type:uuid;not null;index"`
    Title         string    `json:"title" gorm:"type:varchar(500)"`
    Content       string    `json:"content" gorm:"type:text"`
    ContentHTML   string    `json:"content_html" gorm:"type:text"`
    Version       int       `json:"version" gorm:"not null"`
    ChangedByID   uuid.UUID `json:"changed_by_id" gorm:"type:uuid"`
    ChangeSummary string    `json:"change_summary" gorm:"type:text"`
    CreatedAt     time.Time `json:"created_at"`
    
    ChangedBy     *User     `json:"changed_by,omitempty" gorm:"foreignKey:ChangedByID"`
}

type CreateNoteRequest struct {
    Title    string     `json:"title" binding:"max=500"`
    Content  string     `json:"content"`
    FolderID *uuid.UUID `json:"folder_id"`
}

type UpdateNoteRequest struct {
    Title    string `json:"title" binding:"required,max=500"`
    Content  string `json:"content"`
    Version  int    `json:"version" binding:"required,min=1"`
}

type NoteListQuery struct {
    Page     int        `form:"page,default=1" binding:"min=1"`
    Size     int        `form:"size,default=20" binding:"max=100"`
    FolderID *uuid.UUID `form:"folder_id"`
    Keyword  string     `form:"keyword"`
    Sort     string     `form:"sort,default=updated_at"` // updated_at, created_at, title
    Order    string     `form:"order,default=desc"`      // asc, desc
}
```

### 2.3 Folder (文件夹)

```go
// internal/folder/model.go

type Folder struct {
    ID        uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
    Name      string     `json:"name" gorm:"type:varchar(255);not null"`
    ParentID  *uuid.UUID `json:"parent_id" gorm:"type:uuid;index"`
    OwnerID   uuid.UUID  `json:"owner_id" gorm:"type:uuid;not null;index"`
    Color     string     `json:"color" gorm:"type:varchar(7)"` // #RRGGBB
    SortOrder int        `json:"sort_order" gorm:"default:0"`
    IsDeleted bool       `json:"is_deleted" gorm:"default:false"`
    CreatedAt time.Time  `json:"created_at"`
    UpdatedAt time.Time  `json:"updated_at"`
    
    // 树形结构支持
    Parent   *Folder   `json:"parent,omitempty" gorm:"foreignKey:ParentID"`
    Children []Folder  `json:"children,omitempty" gorm:"foreignKey:ParentID"`
    Notes    []Note    `json:"notes,omitempty" gorm:"foreignKey:FolderID"`
}

// 树形结构构建
type FolderTree struct {
    Folder
    Children []*FolderTree `json:"children,omitempty"`
}

func BuildFolderTree(folders []Folder) []*FolderTree {
    folderMap := make(map[uuid.UUID]*FolderTree)
    var roots []*FolderTree
    
    // 创建所有节点
    for _, f := range folders {
        folderMap[f.ID] = &FolderTree{Folder: f, Children: []*FolderTree{}}
    }
    
    // 构建树
    for _, f := range folders {
        node := folderMap[f.ID]
        if f.ParentID == nil {
            roots = append(roots, node)
        } else {
            if parent, ok := folderMap[*f.ParentID]; ok {
                parent.Children = append(parent.Children, node)
            }
        }
    }
    
    return roots
}
```

### 2.4 File (文件)

```go
// internal/file/model.go

type File struct {
    ID            uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
    NoteID        *uuid.UUID `json:"note_id,omitempty" gorm:"type:uuid;index"`
    UserID        uuid.UUID  `json:"user_id" gorm:"type:uuid;not null;index"`
    FileName      string     `json:"file_name" gorm:"type:varchar(255);not null"`
    FileType      string     `json:"file_type" gorm:"type:varchar(50)"` // image, video, audio, archive, document
    MimeType      string     `json:"mime_type" gorm:"type:varchar(100)"`
    StoragePath   string     `json:"storage_path" gorm:"type:varchar(500);not null"`
    Bucket        string     `json:"bucket" gorm:"type:varchar(100);default:'notes'"`
    SizeBytes     int64      `json:"size_bytes" gorm:"default:0"`
    Width         int        `json:"width,omitempty" gorm:"default:0"`
    Height        int        `json:"height,omitempty" gorm:"default:0"`
    DurationSec   float64   `json:"duration_sec,omitempty"`
    ThumbnailPath string     `json:"thumbnail_path,omitempty" gorm:"type:varchar(500)"`
    Metadata      JSONMap    `json:"metadata,omitempty" gorm:"type:jsonb;default:'{}'"`
    Status        int8       `json:"status" gorm:"default:0"` // 0: 上传中, 1: 完成, 2: 处理中, 3: 失败
    ProcessedAt   *time.Time `json:"processed_at,omitempty"`
    CreatedAt     time.Time  `json:"created_at"`
    UpdatedAt     time.Time  `json:"updated_at"`
    
    Note *Note `json:"note,omitempty" gorm:"foreignKey:NoteID"`
}

// 文件类型判断
func (f *File) IsImage() bool {
    return f.FileType == "image"
}

func (f *File) IsVideo() bool {
    return f.FileType == "video"
}

func (f *File) IsAudio() bool {
    return f.FileType == "audio"
}

func (f *File) IsArchive() bool {
    return f.FileType == "archive"
}
```

### 2.5 Share (分享)

```go
// internal/share/model.go

// 分享状态枚举
type ShareStatus string

const (
    ShareStatusPending   ShareStatus = "pending"    // 待审核
    ShareStatusApproved  ShareStatus = "approved"   // 已通过
    ShareStatusRejected  ShareStatus = "rejected"    // 已拒绝
)

type Share struct {
    ID               uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
    NoteID           uuid.UUID  `json:"note_id" gorm:"type:uuid;not null;index"`
    ShareToken       string     `json:"share_token" gorm:"type:varchar(64);unique_index;not null"`
    ShareType        string     `json:"share_type" gorm:"type:varchar(20);default:'link'"` // link, email, user
    ShareWithUserID  *uuid.UUID `json:"share_with_user_id,omitempty" gorm:"type:uuid"`
    ShareWithEmail   string     `json:"share_with_email,omitempty" gorm:"type:varchar(255)"`
    Permission       string     `json:"permission" gorm:"type:varchar(20);default:'read'"` // read, write
    PasswordHash     string     `json:"-" gorm:"type:varchar(255)"`
    ExpiresAt        *time.Time `json:"expires_at,omitempty"`              // 用户自定过期时间，NULL表示永久
    Status           ShareStatus `json:"status" gorm:"type:varchar(20);default:'pending'"` // pending/approved/rejected
    ContentSnapshot  string     `json:"content_snapshot,omitempty" gorm:"type:text"` // 分享时的内容快照
    ReviewedAt       *time.Time `json:"reviewed_at,omitempty"`
    ReviewedByID     *uuid.UUID `json:"reviewed_by_id,omitempty" gorm:"type:uuid"`
    RejectionReason  string     `json:"rejection_reason,omitempty" gorm:"type:text"`
    CreatedAt        time.Time  `json:"created_at"`
    CreatedByID      uuid.UUID  `json:"created_by_id" gorm:"type:uuid"`

    Note      *Note   `json:"note,omitempty" gorm:"foreignKey:NoteID"`
    CreatedBy *User   `json:"created_by,omitempty" gorm:"foreignKey:CreatedByID"`
    ReviewedBy *User  `json:"reviewed_by,omitempty" gorm:"foreignKey:ReviewedByID"`
}

type ShareReview struct {
    ID        uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
    ShareID   uuid.UUID  `json:"share_id" gorm:"type:uuid;not null;index"`
    Action    string     `json:"action" gorm:"type:varchar(20);not null"` // approve, reject, re-review
    Reason    string     `json:"reason,omitempty" gorm:"type:text"`
    ReviewedByID *uuid.UUID `json:"reviewed_by_id,omitempty" gorm:"type:uuid"`
    CreatedAt time.Time  `json:"created_at"`

    Share     *Share    `json:"share,omitempty" gorm:"foreignKey:ShareID"`
}

type CreateShareRequest struct {
    ShareType  string     `json:"share_type" binding:"required,oneof=link email user"`
    Permission string     `json:"permission" binding:"required,oneof=read write"`
    Password  string     `json:"password,omitempty"`
    ExpiresAt *time.Time `json:"expires_at,omitempty"` // 用户自定过期时间，NULL表示永久
    ShareWith string      `json:"share_with,omitempty"` // 用户 ID 或邮箱
}
```

### 2.6 AI 对话

```go
// internal/ai/model.go

type AIConversation struct {
    ID        uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
    UserID    uuid.UUID `json:"user_id" gorm:"type:uuid;not null;index"`
    Title     string    `json:"title" gorm:"type:varchar(255)"`
    Model     string    `json:"model" gorm:"type:varchar(50);default:'gpt-4'"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
    
    Messages []AIMessage `json:"messages,omitempty" gorm:"foreignKey:ConversationID"`
}

type AIMessage struct {
    ID             uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
    ConversationID uuid.UUID  `json:"conversation_id" gorm:"type:uuid;not null;index"`
    Role           string     `json:"role" gorm:"type:varchar(20);not null"` // user, assistant, system
    Content        string     `json:"content" gorm:"type:text;not null"`
    References     JSONArray  `json:"references,omitempty" gorm:"type:jsonb;default:'[]'"`
    TokenCount     int        `json:"token_count,omitempty"`
    Model          string     `json:"model,omitempty" gorm:"type:varchar(50)"`
    CreatedAt      time.Time  `json:"created_at"`
}

type AIRefrence struct {
    NoteID      uuid.UUID `json:"note_id"`
    NoteTitle   string    `json:"note_title"`
    ChunkText   string    `json:"chunk_text"`
    Score       float64   `json:"score"`
}

type AskRequest struct {
    Question       string  `json:"question" binding:"required"`
    ConversationID string  `json:"conversation_id,omitempty"`
    Model          string  `json:"model,omitempty"`
    TopK           int     `json:"top_k,omitempty"`
    Stream         bool    `json:"stream,omitempty"`
}
```

### 2.7 Session (会话)

```go
// internal/auth/session/model.go

type Session struct {
    SessionID   string    `json:"session_id"`
    UserID      uuid.UUID `json:"user_id"`
    UserAgent   string    `json:"user_agent"`
    IPAddress   string    `json:"ip_address"`
    DeviceName  string    `json:"device_name"`
    DeviceType  string    `json:"device_type"` // mobile, desktop, web
    LastActive  time.Time `json:"last_active_at"`
    CreatedAt   time.Time `json:"created_at"`
    ExpiresAt   time.Time `json:"expires_at"`
}

type SessionInfo struct {
    UserAgent string
    IPAddress string
}
```

### 2.8 向量嵌入

```go
// internal/ai/embedding.go

type NoteEmbedding struct {
    ID         uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
    NoteID     uuid.UUID `json:"note_id" gorm:"type:uuid;not null;unique_index:idx_note_chunk"`
    ChunkIndex int       `json:"chunk_index" gorm:"unique_index:idx_note_chunk"`
    ChunkText  string    `json:"chunk_text" gorm:"type:text;not null"`
    VectorID   string    `json:"vector_id" gorm:"type:varchar(255)"` // Qdrant 中的 ID
    TokenCount int       `json:"token_count,omitempty"`
    CreatedAt  time.Time `json:"created_at"`
    UpdatedAt  time.Time `json:"updated_at"`
    
    Note *Note `json:"note,omitempty" gorm:"foreignKey:NoteID"`
}

// 文本切片策略
type ChunkConfig struct {
    MaxTokens    int    `yaml:"max_tokens"`    // 最大 token 数
    OverlapTokens int   `yaml:"overlap_tokens"` // 重叠 token 数
    SplitBy      string `yaml:"split_by"`      // paragraph, sentence, fixed
}

func ChunkText(text string, config ChunkConfig) []string {
    // 1. 按段落分割
    paragraphs := strings.Split(text, "\n\n")
    
    var chunks []string
    var currentChunk strings.Builder
    
    for _, para := range paragraphs {
        tokens := estimateTokens(para)
        
        if currentChunk.Len() == 0 {
            currentChunk.WriteString(para)
        } else if estimateTokens(currentChunk.String())+tokens <= config.MaxTokens {
            currentChunk.WriteString("\n\n")
            currentChunk.WriteString(para)
        } else {
            chunks = append(chunks, currentChunk.String())
            currentChunk.Reset()
            currentChunk.WriteString(para)
        }
    }
    
    if currentChunk.Len() > 0 {
        chunks = append(chunks, currentChunk.String())
    }
    
    return chunks
}
```

## 3. DTO 转换

```go
// internal/note/dto.go

type NoteDTO struct {
    ID              uuid.UUID  `json:"id"`
    Title           string     `json:"title"`
    Content         string     `json:"content,omitempty"`
    ContentHTML     string     `json:"content_html,omitempty"`
    OwnerID         uuid.UUID  `json:"owner_id"`
    FolderID        *uuid.UUID `json:"folder_id,omitempty"`
    Version         int        `json:"version"`
    IsPublic        bool       `json:"is_public"`
    ShareToken      string     `json:"share_token,omitempty"`
    ViewCount       int        `json:"view_count"`
    CreatedAt       time.Time  `json:"created_at"`
    UpdatedAt       time.Time  `json:"updated_at"`
    ContentPreview  string     `json:"content_preview,omitempty"` // 预览文本
    Files           []FileDTO  `json:"files,omitempty"`
}

type NoteListItemDTO struct {
    ID             uuid.UUID  `json:"id"`
    Title          string     `json:"title"`
    ContentPreview string     `json:"content_preview"`
    OwnerID        uuid.UUID  `json:"owner_id"`
    FolderID       *uuid.UUID `json:"folder_id,omitempty"`
    Version        int        `json:"version"`
    IsPublic       bool       `json:"is_public"`
    ShareToken     string     `json:"share_token,omitempty"`
    ViewCount      int        `json:"view_count"`
    CreatedAt      time.Time  `json:"created_at"`
    UpdatedAt      time.Time  `json:"updated_at"`
}

func (n *Note) ToDTO() *NoteDTO {
    dto := &NoteDTO{
        ID:          n.ID,
        Title:       n.Title,
        Content:     n.Content,
        ContentHTML: n.ContentHTML,
        OwnerID:     n.OwnerID,
        FolderID:    n.FolderID,
        Version:     n.Version,
        IsPublic:    n.IsPublic,
        ShareToken:  n.ShareToken,
        ViewCount:   n.ViewCount,
        CreatedAt:   n.CreatedAt,
        UpdatedAt:   n.UpdatedAt,
    }
    
    // 生成预览
    if len(n.Content) > 200 {
        dto.ContentPreview = n.Content[:200] + "..."
    } else {
        dto.ContentPreview = n.Content
    }
    
    return dto
}

func (n *Note) ToListItemDTO() *NoteListItemDTO {
    content := n.Content
    if len(content) > 200 {
        content = content[:200] + "..."
    }
    
    return &NoteListItemDTO{
        ID:             n.ID,
        Title:          n.Title,
        ContentPreview: content,
        OwnerID:        n.OwnerID,
        FolderID:       n.FolderID,
        Version:        n.Version,
        IsPublic:       n.IsPublic,
        ShareToken:     n.ShareToken,
        ViewCount:      n.ViewCount,
        CreatedAt:      n.CreatedAt,
        UpdatedAt:      n.UpdatedAt,
    }
}
```

## 4. 枚举类型

```go
// pkg/enum/enum.go

type UserStatus int8

const (
    UserStatusNormal  UserStatus = 1
    UserStatusDisabled UserStatus = 0
)

type FileStatus int8

const (
    FileStatusUploading   FileStatus = 0
    FileStatusCompleted   FileStatus = 1
    FileStatusProcessing  FileStatus = 2
    FileStatusFailed      FileStatus = 3
)

type FileType string

const (
    FileTypeImage    FileType = "image"
    FileTypeVideo    FileType = "video"
    FileTypeAudio    FileType = "audio"
    FileTypeArchive  FileType = "archive"
    FileTypeDocument FileType = "document"
)

type ShareType string

const (
    ShareTypeLink   ShareType = "link"
    ShareTypeEmail  ShareType = "email"
    ShareTypeUser   ShareType = "user"
)

type SharePermission string

const (
    SharePermissionRead  SharePermission = "read"
    SharePermissionWrite SharePermission = "write"
)

type AIMessageRole string

const (
    AIMessageRoleUser      AIMessageRole = "user"
    AIMessageRoleAssistant AIMessageRole = "assistant"
    AIMessageRoleSystem    AIMessageRole = "system"
)
```

## 5. 索引设计

```sql
-- 用户表索引
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_status ON users(status);

-- 笔记表索引
CREATE INDEX idx_notes_owner ON notes(owner_id);
CREATE INDEX idx_notes_folder ON notes(folder_id);
CREATE INDEX idx_notes_share_token ON notes(share_token);
CREATE INDEX idx_notes_public ON notes(is_public);
CREATE INDEX idx_notes_deleted ON notes(is_deleted);
CREATE INDEX idx_notes_updated ON notes(updated_at DESC);

-- 复合索引优化常见查询
CREATE INDEX idx_notes_owner_folder ON notes(owner_id, folder_id);
CREATE INDEX idx_notes_owner_updated ON notes(owner_id, updated_at DESC);

-- 文件表索引
CREATE INDEX idx_files_note ON files(note_id);
CREATE INDEX idx_files_user ON files(user_id);
CREATE INDEX idx_files_type ON files(file_type);
CREATE INDEX idx_files_status ON files(status);

-- AI 对话索引
CREATE INDEX idx_ai_conv_user ON ai_conversations(user_id);
CREATE INDEX idx_ai_msg_conv ON ai_messages(conversation_id);
CREATE INDEX idx_ai_msg_created ON ai_messages(created_at DESC);

-- 嵌入向量索引
CREATE INDEX idx_note_emb_note ON note_embeddings(note_id);
CREATE INDEX idx_note_emb_updated ON note_embeddings(updated_at DESC);
```

## 6. 数据库约束

```sql
-- 用户邮箱唯一
ALTER TABLE users ADD CONSTRAINT users_email_unique UNIQUE (email);

-- OAuth provider + provider_user_id 唯一
ALTER TABLE user_oauth ADD CONSTRAINT oauth_unique UNIQUE (provider, provider_user_id);

-- 笔记乐观锁版本号非负
ALTER TABLE notes ADD CONSTRAINT notes_version_positive CHECK (version >= 1);

-- 文件大小合理
ALTER TABLE files ADD CONSTRAINT files_size_reasonable CHECK (size_bytes > 0 AND size_bytes < 1073741824); -- < 1GB

-- 分享 Token 唯一
ALTER TABLE note_shares ADD CONSTRAINT shares_token_unique UNIQUE (share_token);
```
