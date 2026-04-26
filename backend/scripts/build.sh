#!/usr/bin/env bash
# 编译 HTTP 服务到 backend/bin/krasis-server
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

mkdir -p bin
go build -buildvcs=false -trimpath -o bin/krasis-server ./cmd/server

echo "Built: $ROOT/bin/krasis-server"
