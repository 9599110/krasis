#!/usr/bin/env bash
# 编译并启动服务（等价于 build.sh 后 start.sh）
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
: "${SERVER_PORT:=18081}"
export SERVER_PORT
"$ROOT/scripts/build.sh"
exec "$ROOT/scripts/start.sh" "$@"
