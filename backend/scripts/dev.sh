#!/usr/bin/env bash
# 开发模式：不生成 bin，直接用 go run（适合改代码即测）
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
: "${SERVER_PORT:=18081}"
export SERVER_PORT

echo "=> running pending migrations..."
go run ./cmd/migrate --baseline

echo "=> starting krasis server..."
exec go run ./cmd/server "$@"
