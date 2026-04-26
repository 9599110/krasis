# Go 后端项目

本目录包含 NoteKeeper 后端服务的完整实现。

## 项目结构

```
backend/
├── cmd/
│   ├── server/           # API 服务入口
│   │   └── main.go
│   └── worker/           # 异步任务 worker
│       └── main.go
├── internal/
│   ├── config/           # 配置管理
│   ├── server/           # HTTP/WebSocket 服务器
│   ├── middleware/       # 中间件
│   ├── auth/             # 认证模块
│   ├── user/             # 用户模块
│   ├── note/             # 笔记模块
│   ├── folder/           # 文件夹模块
│   ├── share/            # 分享模块
│   ├── file/             # 文件模块
│   ├── search/           # 搜索模块
│   ├── ai/               # AI 模块
│   └── collab/           # 协同编辑模块
├── pkg/
│   ├── database/         # 数据库客户端
│   ├── cache/            # 缓存
│   ├── storage/          # 对象存储
│   ├── vector/           # 向量数据库
│   ├── search/           # 搜索引擎
│   └── worker/           # 任务队列
├── migrations/           # 数据库迁移
├── config.yaml          # 配置文件
├── Dockerfile
├── Dockerfile.dev
└── go.mod
```

## 快速开始

### 1. 安装依赖

```bash
go mod download
```

### 2. 配置

```bash
cp config.yaml.example config.yaml
# 编辑 config.yaml
```

### 3. 运行

```bash
# 开发模式 (需要 air 热重载)
air

# 生产模式
go build -o server ./cmd/server
./server

# 运行 worker
go build -o worker ./cmd/worker
./worker
```

### 4. 运行测试

```bash
go test -v ./...
```

## 配置说明

### config.yaml

```yaml
app:
  name: notekeeper
  env: development
  port: 8080
  host: 0.0.0.0

database:
  dsn: postgres://user:pass@localhost:5432/notekeeper?sslmode=disable
  max_open_conns: 25
  max_idle_conns: 5
  conn_max_lifetime: 5m

redis:
  addr: localhost:6379
  password: ""
  db: 0
  pool_size: 10

minio:
  endpoint: localhost:9000
  access_key: minioadmin
  secret_key: minioadmin
  bucket: notes
  use_ssl: false

elasticsearch:
  url: http://localhost:9200
  index_prefix: notekeeper

qdrant:
  host: localhost
  port: 6334
  collection: note_chunks

oauth:
  github:
    client_id: ""
    client_secret: ""
    redirect_uri: http://localhost:8080/auth/github/callback
  google:
    client_id: ""
    client_secret: ""
    redirect_uri: http://localhost:8080/auth/google/callback

jwt:
  secret: your-secret-key
  expiration: 7d

ai:
  model: gpt-4
  embedding_url: http://localhost:8081/encode
  temperature: 0.7
  max_tokens: 2000

storage:
  max_file_size: 100MB
  allowed_types:
    - image
    - video
    - audio
    - archive
    - document
```

## 环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| PORT | 服务端口 | 8080 |
| DB_DSN | 数据库连接字符串 | - |
| REDIS_ADDR | Redis 地址 | localhost:6379 |
| JWT_SECRET | JWT 密钥 | - |

## API 端点

### 认证
- `GET /api/v1/auth/:provider/login` - OAuth 登录
- `GET /api/v1/auth/:provider/callback` - OAuth 回调
- `POST /api/v1/auth/logout` - 登出
- `GET /api/v1/auth/me` - 获取当前用户

### 笔记
- `GET /api/v1/notes` - 笔记列表
- `GET /api/v1/notes/:id` - 笔记详情
- `POST /api/v1/notes` - 创建笔记
- `PUT /api/v1/notes/:id` - 更新笔记
- `DELETE /api/v1/notes/:id` - 删除笔记

### AI
- `POST /api/v1/ai/ask` - AI 问答

### 搜索
- `GET /api/v1/search` - 全文搜索

### WebSocket
- `WS /ws/collab?note_id=&token=` - 协同编辑

## License

MIT
