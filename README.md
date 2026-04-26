# Krasis - 智能笔记系统

企业级智能笔记系统，支持 AI 知识库、全文检索、实时协同编辑。

## 项目结构

```
krasis/
├── backend/                       # Go 后端 (Gin + pgx + Redis)
│   ├── cmd/server/                # 入口
│   ├── internal/                  # 业务模块
│   │   ├── admin/                 # 管理后台
│   │   ├── ai/                    # AI/RAG 系统
│   │   ├── auth/                  # 认证 (OAuth + JWT)
│   │   ├── collab/                # WebSocket 协同编辑
│   │   ├── config/                # 配置加载
│   │   ├── file/                  # MinIO 文件上传
│   │   ├── folder/                # 文件夹管理
│   │   ├── middleware/            # JWT / RBAC / CORS
│   │   ├── note/                  # 笔记 CRUD + 版本
│   │   ├── search/                # PostgreSQL 全文搜索
│   │   ├── server/                # HTTP 路由
│   │   ├── share/                 # 分享 + 审核
│   │   └── user/                  # 用户管理
│   ├── migrations/                # 数据库迁移
│   ├── Dockerfile                 # 生产镜像
│   └── Dockerfile.dev             # 开发镜像
├── sdk/
│   ├── javascript/                # TypeScript SDK (@krasis/sdk)
│   ├── flutter/                   # Flutter/Dart SDK
│   ├── go/                        # Go SDK (github.com/krasis/krasis/sdk/go)
│   └── python/                    # Python SDK (krasis-sdk)
├── frontend/                      # Flutter 客户端
├── docker/
│   └── nginx/nginx.conf           # 反向代理 (WebSocket)
├── docker-compose.yml             # 全栈部署
├── docker-compose.dev.yml         # 仅基础设施
└── .github/workflows/ci.yml       # CI/CD
```

## 技术栈

| 层 | 技术 |
|---|------|
| 后端 | Go 1.25, Gin, pgxpool, Redis |
| 数据库 | PostgreSQL 15 (全文搜索), Redis 7 (会话/缓存), Qdrant (向量), MinIO (对象存储) |
| AI | RAG 管道, 支持 OpenAI/Azure/Anthropic/Ollama |
| 实时协同 | WebSocket (gorilla/websocket) |
| SDK | TypeScript (ESM/CJS), Flutter/Dart, Go, Python (aiohttp + websockets) |
| 前端 | Flutter 3.16+, Riverpod, go_router, flutter_quill |
| 部署 | Docker, Docker Compose, GitHub Actions |

## 快速开始

### 开发环境 (推荐)

```bash
# 1. 启动基础设施
docker compose -f docker-compose.dev.yml up -d

# 2. 运行数据库迁移 (示例)
psql -h localhost -U krasis -d krasis -f backend/migrations/*.up.sql

# 3. 启动后端
cd backend
go mod download
CONFIG_FILE=config.yaml go run cmd/server/main.go

# 4. 启动 Flutter 应用
cd frontend
flutter pub get
flutter run
```

### Docker 全栈部署

```bash
cp .env.example .env
# 编辑 .env 填写配置
docker compose up --build -d
```

### CI/CD

推送代码到 main 或 develop 分支自动触发:
1. Go 测试 + 构建
2. Docker 镜像构建并推送 GHCR

## SDK 使用

### TypeScript
```bash
npm install @krasis/sdk
```
```typescript
import { Client } from '@krasis/sdk';
const client = new Client('http://localhost:8080', { token: 'xxx' });
const notes = await client.notes.list();
```

### Go
```bash
go get github.com/krasis/krasis/sdk/go
```
```go
import "github.com/krasis/krasis/sdk/go/krasis"
client := krasis.NewClient(krasis.Config{BaseURL: "http://localhost:8080", Token: "xxx"})
notes, _ := client.Notes.List(nil)
```

### Python
```bash
pip install krasis-sdk
```
```python
from krasis import Client
async with Client(base_url="http://localhost:8080") as c:
    c.set_token("xxx")
    notes = await c.notes.list()
```

## API 端点

| 方法 | 路径 | 描述 |
|------|------|------|
| **认证** | | |
| GET | `/auth/github/login` | GitHub OAuth |
| GET | `/auth/google/login` | Google OAuth |
| POST | `/auth/login` | 邮箱登录 |
| POST | `/auth/register` | 注册 |
| POST | `/auth/logout` | 登出 |
| GET | `/auth/me` | 获取当前用户 |
| **用户** | | |
| GET | `/user/sessions` | 设备列表 |
| DELETE | `/user/sessions/:id` | 设备下线 |
| PUT | `/user/profile` | 更新资料 |
| **笔记** | | |
| GET | `/notes` | 笔记列表 |
| POST | `/notes` | 创建笔记 |
| GET | `/notes/:id` | 笔记详情 |
| PUT | `/notes/:id` | 更新笔记 (If-Match) |
| DELETE | `/notes/:id` | 删除笔记 |
| GET | `/notes/:id/versions` | 版本历史 |
| POST | `/notes/:id/restore` | 恢复版本 |
| **文件夹** | | |
| GET/POST | `/folders` | 列表/创建 |
| PUT/DELETE | `/folders/:id` | 更新/删除 |
| **分享** | | |
| POST/GET/DELETE | `/notes/:id/share` | 生成/查询/取消 |
| GET | `/share/:token` | 公开访问 |
| **搜索** | | |
| GET | `/search?q=xxx` | 全文搜索 |
| **AI** | | |
| POST | `/ai/ask` | RAG 问答 |
| POST | `/ai/ask/stream` | 流式问答 (SSE) |
| GET | `/ai/conversations` | 对话列表 |
| **协同** | | |
| GET | `/ws/collab?note_id=&token=` | WebSocket 协同编辑 |
| **管理后台** (需要 admin) | | |
| GET/POST/PUT/DELETE | `/admin/users*` | 用户管理 |
| POST | `/admin/users/batch/disable` | 批量禁用 |
| POST | `/admin/users/export` | 导出 CSV |
| GET | `/admin/stats/overview` | 系统统计 |
| GET/POST/PUT/DELETE | `/admin/shares/*` | 分享审核 |
| GET/POST/PUT/DELETE/POST | `/admin/ai/models*` | 模型管理 (含测试) |
| GET/PUT | `/admin/ai/config` | AI 系统配置 |
