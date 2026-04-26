# 后端模块详细设计

## 1. 项目结构

```
backend/
├── cmd/
│   └── server/
│       └── main.go                 # 应用入口
├── internal/
│   ├── config/
│   │   └── config.go              # 配置管理
│   ├── server/
│   │   ├── http.go                 # HTTP 服务器
│   │   └── ws.go                   # WebSocket 服务器
│   ├── middleware/
│   │   ├── auth.go                 # 认证中间件
│   │   ├── cors.go                 # CORS 中间件
│   │   ├── ratelimit.go            # 限流中间件
│   │   ├── logging.go              # 日志中间件
│   │   └── recovery.go             # 异常恢复中间件
│   ├── auth/
│   │   ├── oauth.go                # OAuth2 实现
│   │   ├── jwt.go                  # JWT 工具
│   │   ├── session.go              # Session 管理
│   │   └── handler.go              # 认证处理器
│   ├── user/
│   │   ├── model.go                # 用户模型
│   │   ├── service.go               # 用户服务
│   │   ├── handler.go               # 用户处理器
│   │   └── repository.go            # 用户仓储
│   ├── note/
│   │   ├── model.go                # 笔记模型
│   │   ├── service.go               # 笔记服务
│   │   ├── handler.go               # 笔记处理器
│   │   ├── repository.go            # 笔记仓储
│   │   └── version.go               # 版本控制
│   ├── folder/
│   │   ├── model.go
│   │   ├── service.go
│   │   └── handler.go
│   ├── share/
│   │   ├── model.go
│   │   ├── service.go
│   │   └── handler.go
│   ├── file/
│   │   ├── model.go
│   │   ├── service.go
│   │   ├── handler.go
│   │   ├── presigner.go             # 预签名 URL
│   │   └── processor.go              # 文件处理器
│   ├── search/
│   │   ├── elastic.go               # Elasticsearch 客户端
│   │   ├── indexer.go               # 索引器
│   │   ├── service.go
│   │   └── handler.go
│   ├── ai/
│   │   ├── embedding.go             # 嵌入模型
│   │   ├── retriever.go             # 检索器
│   │   ├── llm.go                   # LLM 调用
│   │   ├── service.go               # AI 服务
│   │   └── handler.go               # AI 处理器
│   └── collab/
│       ├── hub.go                  # WebSocket Hub
│       ├── room.go                 # 协同房间
│       ├── handler.go              # 协同处理器
│       └── sync.go                 # 同步逻辑
├── pkg/
│   ├── database/
│   │   ├── postgres.go              # PostgreSQL 连接
│   │   └── redis.go                 # Redis 连接
│   ├── storage/
│   │   └── minio.go                 # MinIO 客户端
│   ├── vector/
│   │   └── qdrant.go                # Qdrant 客户端
│   ├── search/
│   │   └── elastic.go               # ES 客户端
│   └── worker/
│       └── asynq.go                 # 异步任务队列
├── migrations/                       # 数据库迁移
├── scripts/                         # 脚本
├── config.yaml                      # 配置文件
├── go.mod
└── go.sum
```

## 2. 核心模块详解

### 2.1 认证模块 (auth)

#### 2.1.1 JWT 实现

```go
// internal/auth/jwt/jwt.go

type Claims struct {
    UserID      string `json:"user_id"`
    Role        string `json:"role"`
    SessionID   string `json:"session_id"`
    jti         string `json:"jti"`       // JWT ID，用于黑名单
    jwt.RegisteredClaims
}

type JWTManager struct {
    secretKey     []byte
    tokenDuration  time.Duration
    issuer         string
}

func (m *JWTManager) Generate(userID, role, sessionID string) (string, error) {
    now := time.Now()
    claims := &Claims{
        UserID:    userID,
        Role:      role,
        SessionID: sessionID,
        jti:       uuid.New().String(),
        RegisteredClaims: jwt.RegisteredClaims{
            Issuer:    m.issuer,
            Subject:   userID,
            IssuedAt:  jwt.NewNumericDate(now),
            ExpiresAt: jwt.NewNumericDate(now.Add(m.tokenDuration)),
        },
    }
    
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString(m.secretKey)
}

func (m *JWTManager) Validate(tokenString string) (*Claims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, ErrInvalidSigningMethod
        }
        return m.secretKey, nil
    })
    
    if err != nil {
        return nil, err
    }
    
    claims, ok := token.Claims.(*Claims)
    if !ok || !token.Valid {
        return nil, ErrInvalidToken
    }
    
    // 检查黑名单
    if m.IsBlacklisted(claims.jti) {
        return nil, ErrTokenRevoked
    }
    
    return claims, nil
}
```

#### 2.1.2 Session 管理

```go
// internal/auth/session/session.go

type Session struct {
    SessionID   string    `json:"session_id"`
    UserID      string    `json:"user_id"`
    UserAgent   string    `json:"user_agent"`
    IPAddress   string    `json:"ip_address"`
    DeviceName  string    `json:"device_name"`
    DeviceType  string    `json:"device_type"`  // mobile, desktop, web
    LastActive  time.Time `json:"last_active_at"`
    CreatedAt    time.Time `json:"created_at"`
    ExpiresAt    time.Time `json:"expires_at"`
}

type SessionManager struct {
    redis    *redis.Client
    duration time.Duration
}

func (m *SessionManager) Create(userID string, info *SessionInfo) (*Session, error) {
    sessionID := uuid.New().String()
    session := &Session{
        SessionID:  sessionID,
        UserID:     userID,
        UserAgent:  info.UserAgent,
        IPAddress:  info.IPAddress,
        DeviceName: detectDeviceName(info.UserAgent),
        DeviceType: detectDeviceType(info.UserAgent),
        LastActive: time.Now(),
        CreatedAt:  time.Now(),
        ExpiresAt:  time.Now().Add(m.duration),
    }
    
    key := fmt.Sprintf("session:%s", sessionID)
    if err := m.redis.HMSet(key, toMap(session)).Err(); err != nil {
        return nil, err
    }
    m.redis.Expire(key, m.duration)
    
    // 添加到用户 session 列表
    m.redis.SAdd(fmt.Sprintf("user_sessions:%s", userID), sessionID)
    
    return session, nil
}

func (m *SessionManager) Get(sessionID string) (*Session, error) {
    key := fmt.Sprintf("session:%s", sessionID)
    data, err := m.redis.HGetAll(key).Result()
    if err != nil || len(data) == 0 {
        return nil, ErrSessionNotFound
    }
    
    return parseSession(data), nil
}

func (m *SessionManager) Delete(sessionID string) error {
    session, err := m.Get(sessionID)
    if err != nil {
        return err
    }
    
    pipe := m.redis.Pipeline()
    pipe.Del(fmt.Sprintf("session:%s", sessionID))
    pipe.SRem(fmt.Sprintf("user_sessions:%s", session.UserID), sessionID)
    _, err = pipe.Exec()
    
    return err
}

func (m *SessionManager) GetUserSessions(userID string) ([]*Session, error) {
    sessionIDs, err := m.redis.SMembers(fmt.Sprintf("user_sessions:%s", userID)).Result()
    if err != nil {
        return nil, err
    }
    
    sessions := make([]*Session, 0, len(sessionIDs))
    for _, id := range sessionIDs {
        session, err := m.Get(id)
        if err == nil {
            sessions = append(sessions, session)
        }
    }
    
    return sessions, nil
}

func (m *SessionManager) Refresh(sessionID string) error {
    key := fmt.Sprintf("session:%s", sessionID)
    return m.redis.HSet(key, "last_active_at", time.Now().Format(time.RFC3339)).Err()
}
```

#### 2.1.3 OAuth2 实现

```go
// internal/auth/oauth/oauth.go

type Provider string

const (
    ProviderGitHub Provider = "github"
    ProviderGoogle Provider = "google"
)

type OAuthConfig struct {
    ClientID     string
    ClientSecret string
    RedirectURI  string
    Scopes       []string
}

type OAuthManager struct {
    configs map[Provider]OAuthConfig
    httpClient *http.Client
}

func (m *OAuthManager) GetAuthURL(provider Provider, state string) (string, error) {
    config := m.configs[provider]
    
    switch provider {
    case ProviderGitHub:
        return githubAuthURL(config, state)
    case ProviderGoogle:
        return googleAuthURL(config, state)
    }
    return "", ErrUnknownProvider
}

func (m *OAuthManager) ExchangeCode(provider Provider, code string) (*TokenResponse, error) {
    config := m.configs[provider]
    
    switch provider {
    case ProviderGitHub:
        return m.exchangeGitHubCode(config, code)
    case ProviderGoogle:
        return m.exchangeGoogleCode(config, code)
    }
    return nil, ErrUnknownProvider
}

func (m *OAuthManager) GetUserInfo(provider Provider, accessToken string) (*UserInfo, error) {
    switch provider {
    case ProviderGitHub:
        return m.getGitHubUserInfo(accessToken)
    case ProviderGoogle:
        return m.getGoogleUserInfo(accessToken)
    }
    return nil, ErrUnknownProvider
}
```

### 2.2 笔记模块 (note)

#### 2.2.1 乐观锁实现

```go
// internal/note/service.go

type NoteService struct {
    repo     *NoteRepository
    search   *search.Service
    ai       *ai.Service
}

func (s *NoteService) Update(ctx context.Context, userID string, req *UpdateNoteRequest) (*Note, error) {
    // 1. 获取当前笔记
    note, err := s.repo.GetByID(ctx, req.ID)
    if err != nil {
        return nil, err
    }
    
    // 2. 校验权限
    if note.OwnerID != userID {
        return nil, ErrPermissionDenied
    }
    
    // 3. 乐观锁更新
    updatedNote, err := s.repo.UpdateWithOptimisticLock(ctx, &UpdateParams{
        ID:      req.ID,
        Title:   req.Title,
        Content: req.Content,
        Version: req.Version,
    })
    
    if err != nil {
        if errors.Is(err, ErrVersionConflict) {
            // 获取最新版本
            latestNote, _ := s.repo.GetByID(ctx, req.ID)
            return nil, &ConflictError{
                CurrentVersion: latestNote.Version,
                Note:           latestNote,
            }
        }
        return nil, err
    }
    
    // 4. 保存版本历史
    go s.saveVersionHistory(note, userID, "内容更新")
    
    // 5. 异步更新搜索索引
    if s.search != nil {
        go s.search.IndexNote(updatedNote)
    }
    
    // 6. 异步更新向量索引
    if s.ai != nil {
        go s.ai.UpdateEmbeddings(updatedNote)
    }
    
    return updatedNote, nil
}
```

```go
// internal/note/repository.go

func (r *NoteRepository) UpdateWithOptimisticLock(ctx context.Context, params *UpdateParams) (*Note, error) {
    query := `
        UPDATE notes
        SET title = $1, content = $2, content_html = $3,
            version = version + 1, updated_at = NOW()
        WHERE id = $4 AND version = $5
        RETURNING id, title, content, content_html, owner_id, version, updated_at
    `
    
    var note Note
    err := r.db.QueryRowContext(ctx, query,
        params.Title, params.Content, params.ContentHTML,
        params.ID, params.Version,
    ).Scan(&note.ID, &note.Title, &note.Content, &note.ContentHTML,
        &note.OwnerID, &note.Version, &note.UpdatedAt)
    
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, ErrVersionConflict
        }
        return nil, err
    }
    
    return &note, nil
}
```

#### 2.2.2 分享功能

```go
// internal/note/share.go

type ShareService struct {
    repo       *NoteRepository
    shareRepo  *ShareRepository
}

func (s *ShareService) CreateShare(ctx context.Context, userID, noteID string, req *CreateShareRequest) (*Share, error) {
    note, err := s.repo.GetByID(ctx, noteID)
    if err != nil {
        return nil, err
    }
    
    if note.OwnerID != userID {
        return nil, ErrPermissionDenied
    }
    
    share := &Share{
        ID:          uuid.New(),
        NoteID:      noteID,
        ShareToken:  generateShareToken(),
        ShareType:   req.ShareType,
        Permission:  req.Permission,
        PasswordHash: "",
        ExpiresAt:   req.ExpiresAt,
        CreatedAt:   time.Now(),
        CreatedBy:   userID,
    }
    
    if req.Password != "" {
        share.PasswordHash = hashPassword(req.Password)
    }
    
    return s.shareRepo.Create(ctx, share)
}

func (s *ShareService) AccessShare(ctx context.Context, token, password string) (*Note, string, error) {
    share, err := s.shareRepo.GetByToken(ctx, token)
    if err != nil {
        return nil, "", err
    }
    
    // 校验过期
    if share.ExpiresAt != nil && share.ExpiresAt.Before(time.Now()) {
        return nil, "", ErrShareExpired
    }
    
    // 校验密码
    if share.PasswordHash != "" {
        if !verifyPassword(password, share.PasswordHash) {
            return nil, "", ErrInvalidPassword
        }
    }
    
    // 获取笔记
    note, err := s.repo.GetByID(ctx, share.NoteID)
    if err != nil {
        return nil, "", err
    }
    
    return note, share.Permission, nil
}
```

### 2.3 文件模块 (file)

#### 2.3.1 预签名上传

```go
// internal/file/presigner.go

type Presigner struct {
    minioClient *minio.Client
    bucket      string
    expiry      time.Duration
}

type PresignResult struct {
    FileID     string `json:"file_id"`
    UploadURL  string `json:"upload_url"`
    ObjectKey  string `json:"object_key"`
    ExpiresIn  int    `json:"expires_in"`
}

func (p *Presigner) GenerateUploadURL(ctx context.Context, req *PresignRequest) (*PresignResult, error) {
    fileID := uuid.New().String()
    objectKey := p.generateObjectKey(fileID, req.FileName)
    
    // 生成预签名 PUT URL
    reqParams := make(minio.PutObjectOptions)
    reqParams["Content-Type"] = req.ContentType
    
    presignedURL, err := p.minioClient.PresignedPutObject(
        ctx,
        p.bucket,
        objectKey,
        p.expiry,
    )
    if err != nil {
        return nil, err
    }
    
    return &PresignResult{
        FileID:    fileID,
        UploadURL: presignedURL.String(),
        ObjectKey: objectKey,
        ExpiresIn: int(p.expiry.Seconds()),
    }, nil
}

func (p *Presigner) generateObjectKey(fileID, fileName) string {
    now := time.Now()
    ext := path.Ext(fileName)
    return fmt.Sprintf("%d/%02d/%02d/%s/original/%s%s",
        now.Year(), now.Month(), now.Day(), fileID, fileID, ext)
}
```

#### 2.3.2 文件处理器

```go
// internal/file/processor.go

type Processor struct {
    minioClient *minio.Client
    bucket      string
    worker      *worker.Client
}

func (p *Processor) ProcessFile(ctx context.Context, file *File) error {
    switch file.FileType {
    case "image":
        return p.processImage(ctx, file)
    case "video":
        return p.processVideo(ctx, file)
    case "audio":
        return p.processAudio(ctx, file)
    case "archive":
        return p.processArchive(ctx, file)
    case "document":
        return p.processDocument(ctx, file)
    }
    return nil
}

func (p *Processor) processImage(ctx context.Context, file *File) error {
    // 入队缩略图生成任务
    _, err := p.worker.EnqueueContext(ctx, &worker.Task{
        Type: "image:thumbnail",
        Payload: map[string]interface{}{
            "file_id":    file.ID,
            "object_key": file.StoragePath,
            "bucket":     file.Bucket,
        },
    })
    return err
}

func (p *Processor) processVideo(ctx context.Context, file *File) error {
    tasks := []*worker.Task{
        {
            Type: "video:thumbnail",
            Payload: map[string]interface{}{
                "file_id":    file.ID,
                "object_key": file.StoragePath,
                "bucket":     file.Bucket,
            },
        },
        {
            Type: "video:transcode",
            Payload: map[string]interface{}{
                "file_id":    file.ID,
                "object_key": file.StoragePath,
                "bucket":     file.Bucket,
                "formats":    []string{"mp4", "hls"},
            },
        },
    }
    
    for _, task := range tasks {
        if _, err := p.worker.EnqueueContext(ctx, task); err != nil {
            return err
        }
    }
    return nil
}

func (p *Processor) processArchive(ctx context.Context, file *File) error {
    _, err := p.worker.EnqueueContext(ctx, &worker.Task{
        Type: "archive:extract",
        Payload: map[string]interface{}{
            "file_id":    file.ID,
            "object_key": file.StoragePath,
            "bucket":     file.Bucket,
            "note_id":    file.NoteID,
        },
    })
    return err
}
```

### 2.4 AI 模块 (ai)

#### 2.4.1 模型配置管理器

```go
// internal/ai/config.go

// AIModelProvider 大模型提供商
type AIModelProvider string

const (
    ProviderOpenAI   AIModelProvider = "openai"
    ProviderAzure    AIModelProvider = "azure"
    ProviderAnthropic AIModelProvider = "anthropic"
    ProviderOllama   AIModelProvider = "ollama"
    ProviderLocal    AIModelProvider = "local"
)

// AIModelType 模型类型
type AIModelType string

const (
    ModelTypeLLM       AIModelType = "llm"
    ModelTypeEmbedding AIModelType = "embedding"
)

// AIModel 大模型配置（从数据库读取）
type AIModel struct {
    ID          string           `json:"id"`
    Name        string           `json:"name"`        // 配置名称
    Provider    AIModelProvider  `json:"provider"`    // 提供商
    Type        AIModelType      `json:"type"`        // llm / embedding
    Endpoint    string           `json:"endpoint"`    // API 端点
    APIKey      string           `json:"api_key"`      // API Key（加密存储）
    ModelName   string           `json:"model_name"`   // 模型名称
    APIVersion  string           `json:"api_version"`  // Azure API 版本
    MaxTokens   int              `json:"max_tokens"`
    Temperature float64          `json:"temperature"`
    TopP        float64          `json:"top_p"`
    IsEnabled   bool             `json:"is_enabled"`
    IsDefault   bool             `json:"is_default"`
    Priority    int              `json:"priority"`     // 优先级
}

// ModelConfigManager 模型配置管理器（从数据库读取）
type ModelConfigManager struct {
    db          *sql.DB
    cache       *redis.Client
    cacheTTL    time.Duration
    mu          sync.RWMutex
    models      []*AIModel       // 内存缓存
    embeddingModels []*AIModel    // 缓存的 embedding 模型
}

func (m *ModelConfigManager) LoadModels(ctx context.Context) error {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    rows, err := m.db.QueryContext(ctx, `
        SELECT id, name, provider, type, endpoint, api_key, model_name, 
               api_version, max_tokens, temperature, top_p, 
               is_enabled, is_default, priority
        FROM ai_models 
        WHERE is_enabled = true
        ORDER BY type, priority
    `)
    if err != nil {
        return err
    }
    defer rows.Close()
    
    m.models = nil
    m.embeddingModels = nil
    
    for rows.Next() {
        var model AIModel
        var apiKeyEncrypted []byte
        err := rows.Scan(
            &model.ID, &model.Name, &model.Provider, &model.Type,
            &model.Endpoint, &apiKeyEncrypted, &model.ModelName,
            &model.APIVersion, &model.MaxTokens, &model.Temperature,
            &model.TopP, &model.IsEnabled, &model.IsDefault, &model.Priority,
        )
        if err != nil {
            return err
        }
        
        // 解密 API Key
        model.APIKey = decrypt(apiKeyEncrypted)
        
        m.models = append(m.models, &model)
        if model.Type == ModelTypeEmbedding {
            m.embeddingModels = append(m.embeddingModels, &model)
        }
    }
    
    // 缓存到 Redis
    m.cacheModels()
    
    return nil
}

func (m *ModelConfigManager) GetDefaultLLM() *AIModel {
    m.mu.RLock()
    defer m.mu.RUnlock()
    
    for _, model := range m.models {
        if model.Type == ModelTypeLLM && model.IsDefault {
            return model
        }
    }
    return nil
}

func (m *ModelConfigManager) GetDefaultEmbedding() *AIModel {
    m.mu.RLock()
    defer m.mu.RUnlock()
    
    for _, model := range m.models {
        if model.Type == ModelTypeEmbedding && model.IsDefault {
            return model
        }
    }
    return nil
}

func (m *ModelConfigManager) GetLLM(modelID string) *AIModel {
    m.mu.RLock()
    defer m.mu.RUnlock()
    
    for _, model := range m.models {
        if model.Type == ModelTypeLLM && model.ID == modelID {
            return model
        }
    }
    return nil
}
```

#### 2.4.2 RAG 服务（使用配置的大模型）

```go
// internal/ai/service.go

type AIService struct {
    modelManager  *ModelConfigManager  // 模型配置管理器
    vectorStore   VectorStore
    noteRepo      *NoteRepository
    config        *AIConfig           // AI 系统配置
    llmFactory    *LLMFactory         // LLM 工厂
    embeddingCache sync.Map            // 缓存 embedding 实例
}

// AIConfig AI 系统配置（从数据库 ai_config 表读取）
type AIConfig struct {
    ChunkSize        int     `json:"chunk_size"`
    ChunkOverlap     int     `json:"chunk_overlap"`
    TopK             int     `json:"top_k"`
    ScoreThreshold   float64 `json:"score_threshold"`
    EnableRAG        bool    `json:"enable_rag"`
    MaxContextTokens  int     `json:"max_context_tokens"`
    SystemPrompt     string  `json:"system_prompt"`
    EnableStreaming  bool    `json:"enable_streaming"`
}

type AskRequest struct {
    Question       string   `json:"question"`
    ConversationID string   `json:"conversation_id,omitempty"`
    ModelID        string   `json:"model_id,omitempty"`      // 指定模型 ID
    TopK           int      `json:"top_k,omitempty"`
    Stream         bool     `json:"stream,omitempty"`
}

func (s *AIService) Ask(ctx context.Context, userID string, req *AskRequest) (*AskResponse, error) {
    // 1. 获取 embedding 模型并向量化
    embeddingModel := s.modelManager.GetDefaultEmbedding()
    if embeddingModel == nil {
        return nil, ErrNoEmbeddingModelConfigured
    }
    
    embedder := s.getEmbedder(embeddingModel)
    queryVector, err := embedder.Encode(ctx, req.Question)
    if err != nil {
        return nil, fmt.Errorf("embedding error: %w", err)
    }
    
    // 2. 在向量数据库中检索相关片段
    topK := req.TopK
    if topK <= 0 {
        topK = s.config.TopK
    }
    
    chunks, err := s.vectorStore.Search(ctx, &VectorSearchRequest{
        Vector:          queryVector,
        UserID:          userID,
        TopK:            topK,
        ScoreThreshold:  s.config.ScoreThreshold,
    })
    if err != nil {
        return nil, fmt.Errorf("vector search error: %w", err)
    }
    
    if len(chunks) == 0 {
        return &AskResponse{
            Answer: "抱歉，我在您的笔记中没有找到相关内容。",
        }, nil
    }
    
    // 3. 获取 LLM 模型
    var llmModel *AIModel
    if req.ModelID != "" {
        llmModel = s.modelManager.GetLLM(req.ModelID)
    }
    if llmModel == nil {
        llmModel = s.modelManager.GetDefaultLLM()
    }
    if llmModel == nil {
        return nil, ErrNoLLMModelConfigured
    }
    
    // 4. 构建 Prompt
    context := s.buildContext(chunks)
    prompt := s.buildPrompt(req.Question, context)
    
    // 5. 获取 LLM 实例并调用
    llm := s.llmFactory.GetLLM(llmModel)
    
    if req.Stream || s.config.EnableStreaming {
        return s.askStream(ctx, llm, prompt, chunks)
    }
    return s.askNoStream(ctx, llm, prompt, chunks)
}

func (s *AIService) buildPrompt(question, context string) string {
    systemPrompt := s.config.SystemPrompt
    if systemPrompt == "" {
        systemPrompt = "你是一个智能笔记助手。请根据以下上下文回答用户的问题。"
    }
    
    return fmt.Sprintf(`%s

上下文：
%s

问题：%s

请基于上下文回答，如果上下文中没有相关信息，请说明无法回答。
回答要求：
1. 准确引用相关笔记内容
2. 回答简洁明了
3. 如果有多个相关来源，请分别引用
`, systemPrompt, context, question)
}

func (s *AIService) buildContext(chunks []*Chunk) string {
    var sb strings.Builder
    for i, chunk := range chunks {
        sb.WriteString(fmt.Sprintf("\n[%d] 来源：%s\n内容：%s\n",
            i+1,
            chunk.NoteTitle,
            chunk.Text,
        ))
    }
    return sb.String()
}
```

#### 2.4.2 向量化处理（支持多种提供商）

```go
// internal/ai/embedding.go

type Embedder interface {
    Encode(ctx context.Context, text string) ([]float32, error)
    EncodeBatch(ctx context.Context, texts []string) ([][]float32, error)
    Dimensions() int
    Name() string
}

// EmbeddingFactory Embedder 工厂
type EmbeddingFactory struct {
    httpClient *http.Client
}

func (f *EmbeddingFactory) GetEmbedder(model *AIModel) Embedder {
    switch model.Provider {
    case ProviderOpenAI:
        return NewOpenAIEmbedder(model)
    case ProviderOllama:
        return NewOllamaEmbedder(model)
    case ProviderLocal:
        return NewLocalEmbedder(model)
    default:
        return NewOpenAIEmbedder(model) // 默认使用 OpenAI
    }
}

// OpenAIEmbedder OpenAI 嵌入模型
type OpenAIEmbedder struct {
    modelName string
    apiKey    string
    endpoint  string
    client    *http.Client
    dim       int
}

func NewOpenAIEmbedder(model *AIModel) *OpenAIEmbedder {
    endpoint := model.Endpoint
    if endpoint == "" {
        endpoint = "https://api.openai.com/v1/embeddings"
    }
    
    return &OpenAIEmbedder{
        modelName: model.ModelName,
        apiKey:    model.APIKey,
        endpoint:  endpoint,
        client:    &http.Client{Timeout: 60 * time.Second},
        dim:       1536, // text-embedding-ada-002
    }
}

func (e *OpenAIEmbedder) Encode(ctx context.Context, text string) ([]float32, error) {
    reqBody, _ := json.Marshal(map[string]interface{}{
        "input": text,
        "model": e.modelName,
    })
    
    req, _ := http.NewRequestWithContext(ctx, "POST", e.endpoint, bytes.NewReader(reqBody))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+e.apiKey)
    
    resp, err := e.client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var result struct {
        Data []struct {
            Embedding []float32 `json:"embedding"`
        } `json:"data"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }
    
    if len(result.Data) == 0 {
        return nil, ErrNoEmbeddingReturned
    }
    
    return result.Data[0].Embedding, nil
}

func (e *OpenAIEmbedder) EncodeBatch(ctx context.Context, texts []string) ([][]float32, error) {
    reqBody, _ := json.Marshal(map[string]interface{}{
        "input": texts,
        "model": e.modelName,
    })
    
    req, _ := http.NewRequestWithContext(ctx, "POST", e.endpoint, bytes.NewReader(reqBody))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+e.apiKey)
    
    resp, err := e.client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var result struct {
        Data []struct {
            Embedding []float32 `json:"embedding"`
        } `json:"data"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }
    
    embeddings := make([][]float32, len(result.Data))
    for i, d := range result.Data {
        embeddings[i] = d.Embedding
    }
    return embeddings, nil
}

func (e *OpenAIEmbedder) Dimensions() int { return e.dim }
func (e *OpenAIEmbedder) Name() string    { return e.modelName }

// OllamaEmbedder Ollama 本地嵌入模型
type OllamaEmbedder struct {
    endpoint string
    modelName string
    client   *http.Client
}

func NewOllamaEmbedder(model *AIModel) *OllamaEmbedder {
    return &OllamaEmbedder{
        endpoint:  model.Endpoint,
        modelName: model.ModelName,
        client:    &http.Client{Timeout: 120 * time.Second},
    }
}

func (e *OllamaEmbedder) Encode(ctx context.Context, text string) ([]float32, error) {
    reqBody, _ := json.Marshal(map[string]interface{}{
        "model": e.modelName,
        "prompt": text,
    })
    
    resp, err := e.client.Post(e.endpoint+"/api/embeddings", "application/json", bytes.NewReader(reqBody))
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var result struct {
        Embedding []float32 `json:"embedding"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }
    
    return result.Embedding, nil
}

func (e *OllamaEmbedder) EncodeBatch(ctx context.Context, texts []string) ([][]float32, error) {
    embeddings := make([][]float32, len(texts))
    for i, text := range texts {
        emb, err := e.Encode(ctx, text)
        if err != nil {
            return nil, err
        }
        embeddings[i] = emb
    }
    return embeddings, nil
}

func (e *OllamaEmbedder) Dimensions() int { return 0 } // 动态获取
func (e *OllamaEmbedder) Name() string    { return e.modelName }
```

#### 2.4.3 LLM 调用（支持多种提供商）

```go
// internal/ai/llm.go

type LLM interface {
    Generate(ctx context.Context, prompt string, options *GenerateOptions) (string, error)
    GenerateStream(ctx context.Context, prompt string, options *GenerateOptions) (<-chan string, error)
}

// LLMFactory LLM 工厂，根据配置创建对应的 LLM 实例
type LLMFactory struct {
    httpClient *http.Client
}

func (f *LLMFactory) GetLLM(model *AIModel) LLM {
    switch model.Provider {
    case ProviderOpenAI:
        return NewOpenAILLM(model)
    case ProviderAzure:
        return NewAzureLLM(model)
    case ProviderAnthropic:
        return NewAnthropicLLM(model)
    case ProviderOllama:
        return NewOllamaLLM(model)
    case ProviderLocal:
        return NewLocalLLM(model)
    default:
        return NewOpenAILLM(model)
    }
}

// OpenAILLM OpenAI LLM
type OpenAILLM struct {
    apiKey      string
    modelName   string
    endpoint    string
    maxTokens   int
    temperature float64
    client      *http.Client
}

func NewOpenAILLM(model *AIModel) *OpenAILLM {
    endpoint := model.Endpoint
    if endpoint == "" {
        endpoint = "https://api.openai.com/v1/chat/completions"
    }
    
    maxTokens := model.MaxTokens
    if maxTokens == 0 {
        maxTokens = 4096
    }
    
    return &OpenAILLM{
        apiKey:      model.APIKey,
        modelName:   model.ModelName,
        endpoint:    endpoint,
        maxTokens:   maxTokens,
        temperature: model.Temperature,
        client:      &http.Client{Timeout: 120 * time.Second},
    }
}

func (l *OpenAILLM) GenerateStream(ctx context.Context, prompt string, options *GenerateOptions) (<-chan string, error) {
    messages := []map[string]string{{"role": "user", "content": prompt}}
    
    temperature := l.temperature
    if options != nil && options.Temperature > 0 {
        temperature = options.Temperature
    }
    
    maxTokens := l.maxTokens
    if options != nil && options.MaxTokens > 0 {
        maxTokens = options.MaxTokens
    }
    
    reqBody := map[string]interface{}{
        "model": l.modelName,
        "messages": messages,
        "stream": true,
        "max_tokens": maxTokens,
        "temperature": temperature,
    }
    
    body, _ := json.Marshal(reqBody)
    req, _ := http.NewRequestWithContext(ctx, "POST", l.endpoint, bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+l.apiKey)
    
    resp, err := l.client.Do(req)
    if err != nil {
        return nil, err
    }
    
    ch := make(chan string)
    
    go func() {
        defer close(ch)
        defer resp.Body.Close()
        
        reader := bufio.NewReader(resp.Body)
        for {
            line, err := reader.ReadString('\n')
            if err != nil {
                break
            }
            
            if strings.HasPrefix(line, "data: ") {
                if strings.TrimSpace(line) == "data: [DONE]" {
                    break
                }
                
                var delta struct {
                    Choices []struct {
                        Delta struct {
                            Content string `json:"content"`
                        } `json:"delta"`
                    } `json:"choices"`
                }
                
                if json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &delta) == nil {
                    if len(delta.Choices) > 0 && delta.Choices[0].Delta.Content != "" {
                        ch <- delta.Choices[0].Delta.Content
                    }
                }
            }
        }
    }()
    
    return ch, nil
}

// OllamaLLM Ollama 本地 LLM
type OllamaLLM struct {
    endpoint    string
    modelName   string
    temperature float64
    maxTokens   int
    client      *http.Client
}

func NewOllamaLLM(model *AIModel) *OllamaLLM {
    endpoint := model.Endpoint
    if endpoint == "" {
        endpoint = "http://localhost:11434"
    }
    
    return &OllamaLLM{
        endpoint:    endpoint,
        modelName:   model.ModelName,
        temperature: model.Temperature,
        maxTokens:   model.MaxTokens,
        client:      &http.Client{Timeout: 300 * time.Second},
    }
}

func (l *OllamaLLM) GenerateStream(ctx context.Context, prompt string, options *GenerateOptions) (<-chan string, error) {
    temperature := l.temperature
    if options != nil && options.Temperature > 0 {
        temperature = options.Temperature
    }
    
    maxTokens := l.maxTokens
    if options != nil && options.MaxTokens > 0 {
        maxTokens = options.MaxTokens
    }
    
    reqBody, _ := json.Marshal(map[string]interface{}{
        "model": l.modelName,
        "prompt": prompt,
        "stream": true,
        "options": map[string]interface{}{
            "temperature": temperature,
            "num_predict": maxTokens,
        },
    })
    
    req, _ := http.NewRequestWithContext(ctx, "POST", l.endpoint+"/api/generate", bytes.NewReader(reqBody))
    req.Header.Set("Content-Type", "application/json")
    
    resp, err := l.client.Do(req)
    if err != nil {
        return nil, err
    }
    
    ch := make(chan string)
    
    go func() {
        defer close(ch)
        defer resp.Body.Close()
        
        reader := bufio.NewReader(resp.Body)
        decoder := json.NewDecoder(reader)
        
        for decoder.More() {
            var response struct {
                Response string `json:"response"`
                Done     bool   `json:"done"`
            }
            if err := decoder.Decode(&response); err != nil {
                break
            }
            ch <- response.Response
            if response.Done {
                break
            }
        }
    }()
    
    return ch, nil
}

// AzureLLM Azure OpenAI Service
type AzureLLM struct {
    apiKey      string
    endpoint    string
    modelName   string
    apiVersion  string
    maxTokens   int
    temperature float64
    client      *http.Client
}

func NewAzureLLM(model *AIModel) *AzureLLM {
    apiVersion := model.APIVersion
    if apiVersion == "" {
        apiVersion = "2024-02-01"
    }
    
    return &AzureLLM{
        apiKey:      model.APIKey,
        endpoint:    model.Endpoint,
        modelName:   model.ModelName,
        apiVersion:  apiVersion,
        maxTokens:   model.MaxTokens,
        temperature: model.Temperature,
        client:      &http.Client{Timeout: 120 * time.Second},
    }
}

func (l *AzureLLM) GenerateStream(ctx context.Context, prompt string, options *GenerateOptions) (<-chan string, error) {
    url := fmt.Sprintf("%s/openai/deployments/%s/chat/completions?api-version=%s",
        l.endpoint, l.modelName, l.apiVersion)
    
    messages := []map[string]string{{"role": "user", "content": prompt}}
    
    reqBody := map[string]interface{}{
        "messages": messages,
        "stream": true,
        "max_tokens": l.maxTokens,
        "temperature": l.temperature,
    }
    
    body, _ := json.Marshal(reqBody)
    req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("api-key", l.apiKey)
    
    resp, err := l.client.Do(req)
    if err != nil {
        return nil, err
    }
    
    ch := make(chan string)
    
    go func() {
        defer close(ch)
        defer resp.Body.Close()
        
        reader := bufio.NewReader(resp.Body)
        for {
            line, err := reader.ReadString('\n')
            if err != nil {
                break
            }
            
            if strings.HasPrefix(line, "data: ") {
                if strings.TrimSpace(line) == "data: [DONE]" {
                    break
                }
                
                var delta struct {
                    Choices []struct {
                        Delta struct {
                            Content string `json:"content"`
                        } `json:"delta"`
                    } `json:"choices"`
                }
                
                if json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &delta) == nil {
                    if len(delta.Choices) > 0 && delta.Choices[0].Delta.Content != "" {
                        ch <- delta.Choices[0].Delta.Content
                    }
                }
            }
        }
    }()
    
    return ch, nil
}
```

#### 2.4.4 模型配置热更新

```go
// internal/ai/config_reloader.go

// ConfigReloader 模型配置热更新器
type ConfigReloader struct {
    modelManager *ModelConfigManager
    ticker       *time.Ticker
    interval     time.Duration
    done         chan struct{}
}

func NewConfigReloader(mgr *ModelConfigManager, interval time.Duration) *ConfigReloader {
    return &ConfigReloader{
        modelManager: mgr,
        interval:     interval,
        done:        make(chan struct{}),
    }
}

func (r *ConfigReloader) Start(ctx context.Context) {
    r.ticker = time.NewTicker(r.interval)
    
    go func() {
        for {
            select {
            case <-r.ticker.C:
                if err := r.modelManager.LoadModels(ctx); err != nil {
                    log.Printf("Failed to reload AI models: %v", err)
                }
            case <-r.done:
                r.ticker.Stop()
                return
            case <-ctx.Done():
                r.ticker.Stop()
                return
            }
        }
    }()
}

func (r *ConfigReloader) Stop() {
    close(r.done)
}

// 模型配置变更时自动重载（通过数据库触发）
// 可以在 ai_models 表添加 trigger 或使用 CDC
```
```

### 2.5 协同编辑模块 (collab)

#### 2.5.1 WebSocket Hub

```go
// internal/collab/hub.go

type Hub struct {
    rooms      sync.Map       // map[string]*Room
    register   chan *Client
    unregister chan *Client
    broadcast  chan *Message
    mu         sync.RWMutex
}

type Client struct {
    Hub      *Hub
    Conn     *websocket.Conn
    Send     chan []byte
    UserID   string
    NoteID   string
    SessionID string
}

type Room struct {
    ID       string
    NoteID   string
    Clients  sync.Map  // map[string]*Client
    doc      *y.Doc    // Yjs document
    mu       sync.RWMutex
}

func (h *Hub) Run() {
    for {
        select {
        case client := <-h.register:
            h.addClientToRoom(client)
            
        case client := <-h.unregister:
            h.removeClientFromRoom(client)
            
        case message := <-h.broadcast:
            h.broadcastToRoom(message)
        }
    }
}

func (h *Hub) addClientToRoom(client *Client) {
    room, ok := h.rooms.Load(client.NoteID)
    if !ok {
        room = NewRoom(client.NoteID)
        h.rooms.Store(client.NoteID, room)
    }
    
    r := room.(*Room)
    r.AddClient(client)
    
    // 发送当前文档状态
    go r.sendState(client)
}

func (h *Hub) removeClientFromRoom(client *Client) {
    if room, ok := h.rooms.Load(client.NoteID); ok {
        r := room.(*Room)
        r.RemoveClient(client)
        
        // 广播用户离开
        r.BroadcastLeave(client.UserID)
        
        // 清理空房间
        if r.ClientCount() == 0 {
            h.rooms.Delete(client.NoteID)
        }
    }
}
```

#### 2.5.2 同步逻辑

```go
// internal/collab/sync.go

func (r *Room) HandleMessage(client *Client, msg *Message) error {
    switch msg.Type {
    case "sync":
        return r.handleSync(client, msg)
    case "awareness":
        return r.handleAwareness(client, msg)
    case "awareness_query":
        return r.handleAwarenessQuery(client)
    }
    return nil
}

func (r *Room) handleSync(client *Client, msg *Message) error {
    var syncMsg struct {
        Update []byte `json:"update"`
    }
    if err := json.Unmarshal(msg.Payload, &syncMsg); err != nil {
        return err
    }
    
    r.mu.Lock()
    // 应用更新到 Yjs 文档
    y.ApplyUpdate(r.doc, syncMsg.Update)
    r.mu.Unlock()
    
    // 广播给其他客户端
    r.Broadcast(client, msg.Payload, false)
    
    // 持久化更新
    go r.persistUpdate(syncMsg.Update)
    
    return nil
}

func (r *Room) handleAwareness(client *Client, msg *Message) error {
    var awareness struct {
        UserID   string    `json:"user_id"`
        Username string    `json:"username"`
        Cursor   *Cursor   `json:"cursor,omitempty"`
    }
    if err := json.Unmarshal(msg.Payload, &awareness); err != nil {
        return err
    }
    
    // 更新并广播
    r.Broadcast(client, msg.Payload, false)
    
    return nil
}

func (r *Room) Broadcast(source *Client, payload []byte, includeSource bool) {
    r.clients.Range(func _, client interface{}) bool {
        c := client.(*Client)
        if includeSource || c != source {
            select {
            case c.Send <- payload:
            default:
                // 客户端缓冲区满，关闭连接
                c.Hub.unregister <- c
            }
        }
        return true
    })
}
```

### 2.6 搜索模块 (search)

#### 2.6.1 Elasticsearch 索引

```go
// internal/search/indexer.go

type Indexer struct {
    client *elastic.Client
    index  string
}

func (i *Indexer) IndexNote(note *Note) error {
    doc := map[string]interface{}{
        "id":         note.ID,
        "title":      note.Title,
        "content":    note.Content,
        "owner_id":   note.OwnerID,
        "folder_id":  note.FolderID,
        "created_at": note.CreatedAt,
        "updated_at": note.UpdatedAt,
        "is_public":  note.IsPublic,
    }
    
    _, err := i.client.Index().
        Index(i.index).
        Id(note.ID).
        BodyJson(doc).
        Do(context.Background())
    
    return err
}

func (i *Indexer) Search(ctx context.Context, req *SearchRequest) (*SearchResult, error) {
    query := i.buildQuery(req)
    
    searchResult, err := i.client.Search().
        Index(i.index).
        Query(query).
        From((req.Page-1)*req.Size).
        Size(req.Size).
        Highlight(highlightConfig).
        Do(ctx)
    
    if err != nil {
        return nil, err
    }
    
    return i.parseResult(searchResult)
}

func (i *Indexer) buildQuery(req *SearchRequest) elastic.Query {
    must := []elastic.Query{
        elastic.NewMultiMatchQuery(req.Query, "title", "content").
            Type("best_fields").
            Fuzziness("AUTO"),
    }
    
    // 用户自己的笔记或公开笔记
    filter := []elastic.Query{
        elastic.NewTermQuery("owner_id", req.UserID),
    }
    
    if req.Type == "notes" {
        filter = append(filter, elastic.NewTermQuery("is_public", true))
    }
    
    return elastic.NewBoolQuery().
        Must(must...).
        Filter(filter...)
}
```

## 3. 异步任务处理

### 3.1 Asynq Worker

```go
// internal/worker/processor.go

type TaskProcessor struct {
    redis        *redis.Client
    minioClient  *minio.Client
    embedding    Embedder
    vectorStore  VectorStore
}

func (p *TaskProcessor) ProcessThumbnail(ctx context.Context, t *asynq.Task) error {
    var payload struct {
        FileID    string `json:"file_id"`
        ObjectKey string `json:"object_key"`
        Bucket    string `json:"bucket"`
    }
    if err := json.Unmarshal(t.Payload(), &payload); err != nil {
        return err
    }
    
    // 下载原图
    reader, err := p.minioClient.GetObject(ctx, payload.Bucket, payload.ObjectKey, minio.GetObjectOptions{})
    if err != nil {
        return err
    }
    defer reader.Close()
    
    // 生成缩略图
    img, err := imaging.Decode(reader)
    if err != nil {
        return err
    }
    
    sizes := map[string]imaging.ResizeOption{
        "small":  imaging.Resize(200, 0, imaging.Lanczos),
        "medium": imaging.Resize(800, 0, imaging.Lanczos),
        "large":  imaging.Resize(1600, 0, imaging.Lanczos),
    }
    
    for name, option := range sizes {
        thumb := imaging.Thumbnail(img, option.Width, option.Height, imaging.Lanczos)
        
        buf := new(bytes.Buffer)
        imaging.Encode(buf, thumb, imaging.JPEG, imaging.JPEGQuality(80))
        
        thumbKey := strings.Replace(payload.ObjectKey, "/original/", "/thumbnails/", 1)
        thumbKey = strings.Replace(thumbKey, path.Ext(thumbKey), "_"+name+".jpg", 1)
        
        _, err := p.minioClient.PutObject(ctx, payload.Bucket, thumbKey, buf, int64(buf.Len()), minio.PutObjectOptions{
            ContentType: "image/jpeg",
        })
        if err != nil {
            return err
        }
    }
    
    // 更新数据库
    return p.updateFileThumbnail(payload.FileID, thumbKey)
}
```

## 4. 中间件实现

### 4.1 认证中间件

```go
// internal/middleware/auth.go

func AuthMiddleware(jwtManager *auth.JWTManager, sessionManager *session.Manager) gin.HandlerFunc {
    return func(c *gin.Context) {
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            c.JSON(401, gin.H{"code": 1002, "message": "未认证"})
            c.Abort()
            return
        }
        
        parts := strings.Split(authHeader, " ")
        if len(parts) != 2 || parts[0] != "Bearer" {
            c.JSON(401, gin.H{"code": 1002, "message": "无效的认证格式"})
            c.Abort()
            return
        }
        
        claims, err := jwtManager.Validate(parts[1])
        if err != nil {
            c.JSON(401, gin.H{"code": 1002, "message": "无效的 token"})
            c.Abort()
            return
        }
        
        // 刷新 session
        sessionManager.Refresh(claims.SessionID)
        
        // 设置用户信息到上下文
        c.Set("user_id", claims.UserID)
        c.Set("role", claims.Role)
        c.Set("session_id", claims.SessionID)
        
        c.Next()
    }
}

func RequireRole(roles ...string) gin.HandlerFunc {
    return func(c *gin.Context) {
        userRole, exists := c.Get("role")
        if !exists {
            c.JSON(403, gin.H{"code": 1003, "message": "权限不足"})
            c.Abort()
            return
        }
        
        role := userRole.(string)
        for _, r := range roles {
            if r == role {
                c.Next()
                return
            }
        }
        
        c.JSON(403, gin.H{"code": 1003, "message": "权限不足"})
        c.Abort()
    }
}
```

### 4.2 限流中间件

```go
// internal/middleware/ratelimit.go

func RateLimitMiddleware(redis *redis.Client) gin.HandlerFunc {
    return func(c *gin.Context) {
        // 获取用户标识
        identifier := getIdentifier(c)
        
        key := fmt.Sprintf("ratelimit:%s", identifier)
        
        count, err := redis.Incr(key).Result()
        if err != nil {
            c.Next()
            return
        }
        
        if count == 1 {
            redis.Expire(key, time.Minute)
        }
        
        if count > 100 { // 100 次/分钟
            retryAfter, _ := redis.TTL(key).Result()
            c.JSON(429, gin.H{
                "code": 2001,
                "message": "请求过于频繁",
                "data": map[string]interface{}{
                    "retry_after": int(retryAfter.Seconds()),
                },
            })
            c.Abort()
            return
        }
        
        c.Next()
    }
}
```
