# 部署架构设计

## 1. 开发环境部署

### 1.1 Docker Compose 配置

```yaml
# docker-compose.yml

version: '3.8'

services:
  # PostgreSQL 数据库
  postgres:
    image: postgres:15-alpine
    container_name: notekeeper-postgres
    environment:
      POSTGRES_DB: notekeeper
      POSTGRES_USER: notekeeper
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD:-notekeeper123}
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./migrations:/docker-entrypoint-initdb.d
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U notekeeper"]
      interval: 10s
      timeout: 5s
      retries: 5

  # Redis 缓存
  redis:
    image: redis:7-alpine
    container_name: notekeeper-redis
    command: redis-server --appendonly yes
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5

  # MinIO 对象存储
  minio:
    image: minio/minio:latest
    container_name: notekeeper-minio
    command: server /data --console-address ":9001"
    environment:
      MINIO_ROOT_USER: ${MINIO_ACCESS_KEY:-minioadmin}
      MINIO_ROOT_PASSWORD: ${MINIO_SECRET_KEY:-minioadmin}
    ports:
      - "9000:9000"   # API
      - "9001:9001"   # Console
    volumes:
      - minio_data:/data
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:9000/minio/health/live"]
      interval: 30s
      timeout: 20s
      retries: 3

  # Elasticsearch 全文检索
  elasticsearch:
    image: docker.elastic.co/elasticsearch/elasticsearch:8.11.0
    container_name: notekeeper-elasticsearch
    environment:
      - discovery.type=single-node
      - xpack.security.enabled=false
      - "ES_JAVA_OPTS=-Xms512m -Xmx512m"
      - indices.query.bool.max_clause_count=10240
    ports:
      - "9200:9200"
      - "9300:9300"
    volumes:
      - es_data:/usr/share/elasticsearch/data
    healthcheck:
      test: ["CMD-SHELL", "curl -f http://localhost:9200/_cluster/health || exit 1"]
      interval: 30s
      timeout: 10s
      retries: 5

  # Kibana (可选，ES 可视化)
  kibana:
    image: docker.elastic.co/kibana/kibana:8.11.0
    container_name: notekeeper-kibana
    environment:
      ELASTICSEARCH_HOSTS: '["http://elasticsearch:9200"]'
    ports:
      - "5601:5601"
    depends_on:
      elasticsearch:
        condition: service_healthy

  # Qdrant 向量数据库
  qdrant:
    image: qdrant/qdrant:latest
    container_name: notekeeper-qdrant
    ports:
      - "6333:6333"   # REST API
      - "6334:6334"   # gRPC API
    volumes:
      - qdrant_data:/qdrant/storage
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:6333/readyz"]
      interval: 10s
      timeout: 5s
      retries: 5

  # Go 后端服务
  server:
    build:
      context: ./backend
      dockerfile: Dockerfile.dev
    container_name: notekeeper-server
    environment:
      PORT: 8080
      DB_DSN: postgres://notekeeper:${POSTGRES_PASSWORD:-notekeeper123}@postgres:5432/notekeeper?sslmode=disable
      REDIS_ADDR: redis:6379
      MINIO_ENDPOINT: minio:9000
      MINIO_ACCESS_KEY: ${MINIO_ACCESS_KEY:-minioadmin}
      MINIO_SECRET_KEY: ${MINIO_SECRET_KEY:-minioadmin}
      OAUTH_GITHUB_CLIENT_ID: ${OAUTH_GITHUB_CLIENT_ID}
      OAUTH_GITHUB_CLIENT_SECRET: ${OAUTH_GITHUB_CLIENT_SECRET}
      OAUTH_GOOGLE_CLIENT_ID: ${OAUTH_GOOGLE_CLIENT_ID}
      OAUTH_GOOGLE_CLIENT_SECRET: ${OAUTH_GOOGLE_CLIENT_SECRET}
      JWT_SECRET: ${JWT_SECRET:-dev-secret-change-in-production}
      ELASTICSEARCH_URL: http://elasticsearch:9200
      QDRANT_HOST: qdrant
      QDRANT_PORT: 6334
    ports:
      - "8080:8080"
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
      minio:
        condition: service_healthy
    volumes:
      - ./backend:/app
    command: air

  # 异步任务 Worker
  worker:
    build:
      context: ./backend
      dockerfile: Dockerfile.dev
    container_name: notekeeper-worker
    environment:
      DB_DSN: postgres://notekeeper:${POSTGRES_PASSWORD:-notekeeper123}@postgres:5432/notekeeper?sslmode=disable
      REDIS_ADDR: redis:6379
      MINIO_ENDPOINT: minio:9000
      ELASTICSEARCH_URL: http://elasticsearch:9200
      QDRANT_HOST: qdrant
    depends_on:
      - server
    volumes:
      - ./backend:/app
    command: go run cmd/worker/main.go

  # Nginx 反向代理
  nginx:
    image: nginx:alpine
    container_name: notekeeper-nginx
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx/nginx.conf:/etc/nginx/nginx.conf:ro
      - ./nginx/ssl:/etc/nginx/ssl:ro
    depends_on:
      - server

volumes:
  postgres_data:
  redis_data:
  minio_data:
  es_data:
  qdrant_data:
```

### 1.2 环境变量文件

```bash
# .env

# 数据库
POSTGRES_PASSWORD=notekeeper123

# MinIO
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin123

# OAuth
OAUTH_GITHUB_CLIENT_ID=your_github_client_id
OAUTH_GITHUB_CLIENT_SECRET=your_github_client_secret
OAUTH_GOOGLE_CLIENT_ID=your_google_client_id
OAUTH_GOOGLE_CLIENT_SECRET=your_google_client_secret

# JWT
JWT_SECRET=your-super-secret-jwt-key-change-in-production

# LLM (可选)
LLM_API_KEY=your_openai_api_key
LLM_MODEL=gpt-4o-mini

# Embedding Model (可选)
EMBEDDING_MODEL_URL=http://embedding:8080/encode
```

### 1.3 Nginx 配置

```nginx
# nginx/nginx.conf

worker_processes auto;
error_log /var/log/nginx/error.log;
pid /var/run/nginx.pid;

events {
    worker_connections 1024;
    use epoll;
    multi_accept on;
}

http {
    include /etc/nginx/mime.types;
    default_type application/octet-stream;

    log_format main '$remote_addr - $remote_user [$time_local] "$request" '
                    '$status $body_bytes_sent "$http_referer" '
                    '"$http_user_agent" "$http_x_forwarded_for"';

    access_log /var/log/nginx/access.log main;

    sendfile on;
    tcp_nopush on;
    tcp_nodelay on;
    keepalive_timeout 65;
    types_hash_max_size 2048;
    client_max_body_size 100M;

    gzip on;
    gzip_vary on;
    gzip_proxied any;
    gzip_comp_level 6;
    gzip_types text/plain text/css text/xml text/javascript 
               application/json application/javascript application/xml+rss;

    upstream backend {
        server server:8080;
        keepalive 32;
    }

    server {
        listen 80;
        server_name api.notekeeper.com;

        # 重定向到 HTTPS
        return 301 https://$server_name$request_uri;
    }

    server {
        listen 443 ssl http2;
        server_name api.notekeeper.com;

        ssl_certificate /etc/nginx/ssl/cert.pem;
        ssl_certificate_key /etc/nginx/ssl/key.pem;
        ssl_protocols TLSv1.2 TLSv1.3;
        ssl_ciphers HIGH:!aNULL:!MD5;
        ssl_prefer_server_ciphers on;

        # API 代理
        location /api/ {
            proxy_pass http://backend;
            proxy_http_version 1.1;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
            proxy_set_header Connection "";

            proxy_connect_timeout 60s;
            proxy_send_timeout 60s;
            proxy_read_timeout 60s;
        }

        # WebSocket 代理
        location /ws/ {
            proxy_pass http://backend;
            proxy_http_version 1.1;
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection "upgrade";
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;

            proxy_read_timeout 86400;
        }

        # 健康检查
        location /health {
            proxy_pass http://backend;
            proxy_http_version 1.1;
            proxy_set_header Host $host;
        }
    }
}
```

## 2. 生产环境部署

### 2.1 Kubernetes 部署

```yaml
# k8s/namespace.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: notekeeper
  labels:
    app.kubernetes.io/name: notekeeper
```

```yaml
# k8s/configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: notekeeper-config
  namespace: notekeeper
data:
  API_BASE_URL: "https://api.notekeeper.com"
  WS_BASE_URL: "wss://api.notekeeper.com"
  ELASTICSEARCH_URL: "http://elasticsearch:9200"
  QDRANT_HOST: "qdrant"
  QDRANT_PORT: "6334"
```

```yaml
# k8s/secrets.yaml
apiVersion: v1
kind: Secret
metadata:
  name: notekeeper-secrets
  namespace: notekeeper
type: Opaque
stringData:
  DB_DSN: "postgres://user:password@postgres:5432/notekeeper?sslmode=require"
  REDIS_ADDR: "redis:6379"
  REDIS_PASSWORD: "redis-password"
  MINIO_ACCESS_KEY: "minio-access-key"
  MINIO_SECRET_KEY: "minio-secret-key"
  JWT_SECRET: "production-jwt-secret"
  OAUTH_GITHUB_CLIENT_ID: "github-client-id"
  OAUTH_GITHUB_CLIENT_SECRET: "github-client-secret"
  OAUTH_GOOGLE_CLIENT_ID: "google-client-id"
  OAUTH_GOOGLE_CLIENT_SECRET: "google-client-secret"
```

```yaml
# k8s/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: notekeeper-server
  namespace: notekeeper
  labels:
    app: notekeeper-server
spec:
  replicas: 3
  selector:
    matchLabels:
      app: notekeeper-server
  template:
    metadata:
      labels:
        app: notekeeper-server
    spec:
      containers:
        - name: server
          image: notekeeper/server:latest
          ports:
            - containerPort: 8080
          envFrom:
            - configMapRef:
                name: notekeeper-config
            - secretRef:
                name: notekeeper-secrets
          resources:
            requests:
              memory: "256Mi"
              cpu: "250m"
            limits:
              memory: "512Mi"
              cpu: "500m"
          livenessProbe:
            httpGet:
              path: /health
              port: 8080
            initialDelaySeconds: 10
            periodSeconds: 30
          readinessProbe:
            httpGet:
              path: /health
              port: 8080
            initialDelaySeconds: 5
            periodSeconds: 10
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
            - weight: 100
              podAffinityTerm:
                labelSelector:
                  matchLabels:
                    app: notekeeper-server
                topologyKey: kubernetes.io/hostname
```

```yaml
# k8s/service.yaml
apiVersion: v1
kind: Service
metadata:
  name: notekeeper-server
  namespace: notekeeper
spec:
  selector:
    app: notekeeper-server
  ports:
    - port: 80
      targetPort: 8080
  type: ClusterIP
```

```yaml
# k8s/ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: notekeeper-ingress
  namespace: notekeeper
  annotations:
    kubernetes.io/ingress.class: nginx
    cert-manager.io/cluster-issuer: letsencrypt-prod
    nginx.ingress.kubernetes.io/proxy-body-size: "100m"
    nginx.ingress.kubernetes.io/proxy-read-timeout: "3600"
    nginx.ingress.kubernetes.io/proxy-send-timeout: "3600"
    nginx.ingress.kubernetes.io/websocket-services: "notekeeper-server"
    nginx.ingress.kubernetes.io/upstream-hash-by: "$remote_addr"
spec:
  tls:
    - hosts:
        - api.notekeeper.com
      secretName: notekeeper-tls
  rules:
    - host: api.notekeeper.com
      http:
        paths:
          - path: /api
            pathType: Prefix
            backend:
              service:
                name: notekeeper-server
                port:
                  number: 80
          - path: /ws
            pathType: Prefix
            backend:
              service:
                name: notekeeper-server
                port:
                  number: 80
```

```yaml
# k8s/hpa.yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: notekeeper-server-hpa
  namespace: notekeeper
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: notekeeper-server
  minReplicas: 3
  maxReplicas: 20
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
    - type: Resource
      resource:
        name: memory
        target:
          type: Utilization
          averageUtilization: 80
  behavior:
    scaleUp:
      stabilizationWindowSeconds: 60
      policies:
        - type: Percent
          value: 100
          periodSeconds: 15
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
        - type: Percent
          value: 10
          periodSeconds: 60
```

```yaml
# k8s/worker-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: notekeeper-worker
  namespace: notekeeper
spec:
  replicas: 2
  selector:
    matchLabels:
      app: notekeeper-worker
  template:
    metadata:
      labels:
        app: notekeeper-worker
    spec:
      containers:
        - name: worker
          image: notekeeper/server:latest
          command: ["./worker"]
          envFrom:
            - configMapRef:
                name: notekeeper-config
            - secretRef:
                name: notekeeper-secrets
          resources:
            requests:
              memory: "512Mi"
              cpu: "500m"
            limits:
              memory: "1Gi"
              cpu: "1000m"
```

## 3. CI/CD 流程

### 3.1 GitHub Actions

```yaml
# .github/workflows/ci.yml

name: CI

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]

env:
  REGISTRY: ghcr.io
  IMAGE_NAME: ${{ github.repository }}

jobs:
  # 单元测试
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.21'
          
      - name: Cache Go modules
        uses: actions/cache@v3
        with:
          path: ~/go/pkg/mod
          key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}
          restore-keys: |
            ${{ runner.os }}-go-
      
      - name: Run tests
        run: |
          go test -v -race -coverprofile=coverage.out ./...
          
      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          file: coverage.out

  # Docker 构建
  build:
    needs: test
    runs-on: ubuntu-latest
    if: github.event_name == 'push'
    steps:
      - uses: actions/checkout@v4
      
      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v3
        
      - name: Login to Container Registry
        uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}
          
      - name: Extract metadata
        id: meta
        uses: docker/metadata-action@v5
        with:
          images: ghcr.io/${{ github.repository }}/server
          tags: |
            type=ref,event=branch
            type=sha,prefix={{branch}}-
            type=raw,value=latest,enable={{is_default_branch}}
            
      - name: Build and push server image
        uses: docker/build-push-action@v5
        with:
          context: ./backend
          push: true
          tags: ${{ steps.meta.outputs.tags }}
          labels: ${{ steps.meta.outputs.labels }}
          cache-from: type=gha
          cache-to: type=gha,mode=max

  # 部署到开发环境
  deploy-dev:
    needs: build
    runs-on: ubuntu-latest
    if: github.ref == 'refs/heads/develop'
    environment: dev
    steps:
      - name: Deploy to dev cluster
        uses: appleboy/ssh-action@master
        with:
          host: ${{ secrets.DEV_HOST }}
          username: ${{ secrets.DEV_USER }}
          key: ${{ secrets.DEV_SSH_KEY }}
          script: |
            kubectl set image deployment/notekeeper-server server=${{ needs.build.outputs.image }}
            kubectl rollout status deployment/notekeeper-server -n notekeeper
```

## 4. 监控与日志

### 4.1 Prometheus 配置

```yaml
# k8s/prometheus-config.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: prometheus-config
  namespace: monitoring
data:
  prometheus.yml: |
    global:
      scrape_interval: 15s
      evaluation_interval: 15s

    scrape_configs:
      - job_name: 'notekeeper'
        kubernetes_sd_configs:
          - role: pod
        relabel_configs:
          - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
            action: keep
            regex: true
```

### 4.2 Grafana Dashboard

```json
{
  "dashboard": {
    "title": "NoteKeeper Dashboard",
    "panels": [
      {
        "title": "API Request Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(http_requests_total{service=\"notekeeper\"}[5m])",
            "legendFormat": "{{method}} {{path}}"
          }
        ]
      },
      {
        "title": "API Latency (p99)",
        "type": "graph",
        "targets": [
          {
            "expr": "histogram_quantile(0.99, rate(http_request_duration_seconds_bucket{service=\"notekeeper\"}[5m]))",
            "legendFormat": "{{path}}"
          }
        ]
      },
      {
        "title": "Error Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(http_requests_total{service=\"notekeeper\",status=~\"5..\"}[5m])",
            "legendFormat": "5xx errors"
          }
        ]
      },
      {
        "title": "Active Sessions",
        "type": "stat",
        "targets": [
          {
            "expr": "redis_sessions_active{service=\"notekeeper\"}"
          }
        ]
      },
      {
        "title": "AI Query Latency",
        "type": "graph",
        "targets": [
          {
            "expr": "histogram_quantile(0.99, rate(ai_query_duration_seconds_bucket[5m]))",
            "legendFormat": "p99"
          }
        ]
      }
    ]
  }
}
```

## 5. 备份策略

### 5.1 PostgreSQL 备份脚本

```bash
#!/bin/bash
# scripts/backup.sh

set -e

BACKUP_DIR="/backups/postgres"
DATE=$(date +%Y%m%d_%H%M%S)
RETENTION_DAYS=30

# 创建备份目录
mkdir -p ${BACKUP_DIR}

# 执行备份
pg_dump -h postgres -U notekeeper -d notekeeper -Fc -f ${BACKUP_DIR}/notekeeper_${DATE}.dump

# 压缩
gzip ${BACKUP_DIR}/notekeeper_${DATE}.dump

# 删除旧备份
find ${BACKUP_DIR} -name "*.dump.gz" -mtime +${RETENTION_DAYS} -delete

# 上传到 S3 (可选)
# aws s3 cp ${BACKUP_DIR}/notekeeper_${DATE}.dump.gz s3://notekeeper-backups/

echo "Backup completed: notekeeper_${DATE}.dump.gz"
```

### 5.2 定时备份 Cron

```bash
# crontab -e

# 每天凌晨 3 点备份
0 3 * * * /opt/scripts/backup.sh

# 每小时备份增量日志
0 * * * * /opt/scripts/backup-wal.sh
```

## 6. 灾难恢复

### 6.1 数据库恢复流程

```bash
#!/bin/bash
# scripts/restore.sh

BACKUP_FILE=$1

if [ -z "$BACKUP_FILE" ]; then
    echo "Usage: $0 <backup_file>"
    exit 1
fi

# 停止服务
kubectl scale deployment notekeeper-server --replicas=0 -n notekeeper

# 恢复数据库
gunzip -c ${BACKUP_FILE} | pg_restore -h postgres -U notekeeper -d notekeeper --clean

# 重启服务
kubectl scale deployment notekeeper-server --replicas=3 -n notekeeper

echo "Restore completed from ${BACKUP_FILE}"
```

### 6.2 RTO/RPO 目标

| 场景 | RTO | RPO | 策略 |
|------|-----|-----|------|
| 数据库故障 | < 30 分钟 | < 24 小时 | 主从复制 + 每日备份 |
| 全部服务故障 | < 1 小时 | < 1 小时 | K8s 自动恢复 + 多副本 |
| 数据误删 | < 2 小时 | < 24 小时 | 每日备份 + 软删除 |
| 区域故障 | < 4 小时 | < 1 小时 | 跨区域备份 |
