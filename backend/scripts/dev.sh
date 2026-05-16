#!/usr/bin/env bash
# 开发模式：不生成 bin，直接用 go run（适合改代码即测）
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
: "${SERVER_PORT:=9091}"
export SERVER_PORT

# 关闭已有的 krasis 服务进程
echo "=> checking for existing krasis server..."
PID="$(lsof -ti ":$SERVER_PORT" 2>/dev/null || true)"
if [ -n "$PID" ]; then
  echo "=> stopping existing krasis server (PID=$PID) on port $SERVER_PORT..."
  kill "$PID" 2>/dev/null || true
  # 等待进程完全退出
  for i in $(seq 1 10); do
    if ! kill -0 "$PID" 2>/dev/null; then
      break
    fi
    sleep 0.3
  done
  # 强制关闭
  kill -9 "$PID" 2>/dev/null || true
  echo "=> stopped"
fi

echo "=> running pending migrations..."
go run ./cmd/migrate --baseline

echo "=> starting krasis server..."
go run ./cmd/server "$@"
