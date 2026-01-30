#!/usr/bin/env bash
# Build Next.js admin (SSG) and copy to cmd/codex-mcp/web/admin-dist for Go embed.
set -e
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT/web-admin"
npm ci --no-audit --no-fund
npx next build
rm -rf "$ROOT/cmd/codex-mcp/web/admin-dist"
mkdir -p "$ROOT/cmd/codex-mcp/web/admin-dist"
cp -R out/. "$ROOT/cmd/codex-mcp/web/admin-dist/"
echo "Admin built -> cmd/codex-mcp/web/admin-dist"
