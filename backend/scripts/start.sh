#!/usr/bin/env bash
# 启动已编译的二进制（需先执行 scripts/build.sh）
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BIN="$ROOT/bin/krasis-server"

if [[ ! -x "$BIN" ]]; then
  echo "未找到可执行文件: $BIN" >&2
  echo "请先运行: $ROOT/scripts/build.sh" >&2
  exit 1
fi

cd "$ROOT"
: "${SERVER_PORT:=18081}"
export SERVER_PORT
exec "$BIN" "$@"
