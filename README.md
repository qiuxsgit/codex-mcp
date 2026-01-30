# codex-mcp

**codex-mcp** is a deterministic Model Context Protocol (MCP) server that exposes real, local codebases to AI agents through simple, grep-like search tools.

It allows agents such as Cursor, Claude Code, or other MCP-compatible tools to query existing code instead of guessing or re-implementing logic that already exists.

> **Don’t guess the codebase. Query it.**

---

## Why codex-mcp

Modern AI coding agents are powerful, but they are fundamentally blind to:
- internal frameworks
- shared utilities
- company-specific abstractions
- proprietary libraries shipped as jars or binaries

As a result, agents often:
- reinvent existing logic
- violate internal conventions
- generate parallel implementations of the same capability

**codex-mcp solves this by acting as a ground-truth code search layer for AI agents.**

It exposes real source code from configured directories using deterministic, verifiable search results, similar to tools like `grep` or `ripgrep`, but designed for agent consumption.

---

## Design Principles

codex-mcp is intentionally minimal and conservative.

- **Deterministic**
  - No reasoning
  - No speculation
  - No code generation

- **Read-only**
  - Never modifies indexed files
  - Never executes project code

- **Tool-like**
  - Behaves like a native codebase utility
  - Not a conversational assistant

- **Agent-first**
  - Output is structured for machines, not humans
  - Optimized for frequent, safe invocation

If an answer cannot be verified directly from the codebase, codex-mcp will not invent one.

---

## What codex-mcp Is Not

- ❌ Not a chatbot
- ❌ Not a semantic reasoning engine
- ❌ Not a documentation generator
- ❌ Not a code indexer for humans

codex-mcp is closer in spirit to:
- `grep`
- `ripgrep`
- `ctags`

than to an AI assistant.

---

## Architecture Overview

```text
+------------------+
|  AI Agent       |
|  (Cursor, etc.) |
+--------+---------+
         |
         | MCP Tool Calls
         v
+------------------+
|   codex-mcp     |
|-----------------|
| - MCP Server    |
| - Search Engine |
| - Config Store  |
+--------+---------+
         |
         v
+------------------+
|  Local Codebase |
+------------------+
```

---

## 下载预构建二进制

在 [Releases](https://github.com/qiuxsgit/codex-mcp/releases) 页面可下载各平台二进制，无需安装 Go。

| 平台 | 架构 | 文件名 |
|------|------|--------|
| Linux | x86_64 (amd64) | `codex-mcp-linux-amd64` |
| Linux | ARM64 | `codex-mcp-linux-arm64` |
| Windows | x86_64 (amd64) | `codex-mcp-windows-amd64.exe` |
| Windows | ARM64 | `codex-mcp-windows-arm64.exe` |
| macOS | x86_64 (Intel) | `codex-mcp-darwin-amd64` |
| macOS | ARM64 (Apple Silicon) | `codex-mcp-darwin-arm64` |

下载后赋予可执行权限（Linux/macOS）：`chmod +x codex-mcp-*`，然后运行：`./codex-mcp-<平台>-<架构>`（Windows 下直接运行 `.exe`）。Admin 管理页已内置于二进制，无需额外 `web` 目录即可访问 `/admin`。

---

## Quick Start

### Run the server

从源码运行：

```bash
go run ./cmd/codex-mcp
```

Defaults: port `6688`, DB `./data/codex-mcp.db`, ignore file `./data/codex-ignore`.

Override with flags:

```bash
go run ./cmd/codex-mcp --port=8081 --db-path=./data.db --ignore-file-path=./data/codex-ignore
```

Search supports two backends: if [ripgrep](https://github.com/BurntSushi/ripgrep) (`rg`) is installed, it is used for better performance; otherwise built-in (pure Go) search is used, and a one-time log suggests installing `rg` to improve performance.

### Add code directories

1. Open **Admin UI**: http://localhost:6688/admin  
2. Add a directory: name, **absolute path** to a code root, language, role.  
3. Optionally edit **Ignore rules** (gitignore format); save applies immediately (hot reload).

### MCP Inspector (Streamable HTTP)

使用 `npx @modelcontextprotocol/inspector` 测试时：

- **连接地址**：`http://localhost:6688/mcp`
- **协议**：选择 **streamable-http**

Inspector 会通过 JSON-RPC 2.0 调用 `initialize`、`tools/list`、`tools/call`，本服务在 `POST /mcp` 实现上述协议。

### 各工具配置 MCP

先启动 codex-mcp（`go run ./cmd/codex-mcp`），再在对应工具中填入以下配置。默认端口 `6688`，若用 `--port` 改了端口，请把下面 URL 里的端口一并修改。

#### Cursor

在 Cursor 设置中打开 **MCP** 配置（或编辑项目/用户目录下的 MCP 配置文件），添加 Streamable HTTP 服务器：

```json
{
  "mcpServers": {
    "codex-mcp": {
      "url": "http://localhost:6688/mcp",
      "transport": "streamable-http"
    }
  }
}
```

或使用 Cursor 的「Add new MCP server」→ 选择 **Streamable HTTP**，URL 填：`http://localhost:6688/mcp`。

#### Claude Desktop（桌面端）

在 Claude Desktop 的 MCP 配置文件中（例如 `~/Library/Application Support/Claude/claude_desktop_config.json`（macOS）或 `%APPDATA%\Claude\claude_desktop_config.json`（Windows））加入：

```json
{
  "mcpServers": {
    "codex-mcp": {
      "url": "http://localhost:6688/mcp",
      "transport": "streamable-http"
    }
  }
}
```

保存后重启 Claude Desktop。

#### Claude Code / Claude for VS Code

在扩展的 MCP 设置里添加 Streamable HTTP 服务器，URL 填：`http://localhost:6688/mcp`，transport 选 `streamable-http`（若扩展提供该选项）。具体入口以扩展说明为准。

#### Windsurf (Codeium)

在 Windsurf 的 MCP 配置中新增服务器，类型选 **Streamable HTTP** 或 **HTTP**，URL 填：`http://localhost:6688/mcp`。

#### 通用（mcp.json 等）

若工具使用统一的 `mcp.json` 或类似结构，可加入：

```json
{
  "mcpServers": {
    "codex-mcp": {
      "type": "streamable-http",
      "url": "http://localhost:6688/mcp"
    }
  }
}
```

部分客户端用 `"transport": "streamable-http"` 代替 `"type"`，按工具文档为准。

---

### 直接调用 (REST)

不经过 MCP 协议、直接请求搜索接口时：

- **URL**: `http://localhost:6688/mcp/search_internal_codebase`
- **Method**: POST
- **Body (JSON)**: `{"query":"string", "language":"optional", "path_hint":"optional", "limit":10}`
- **Response**: `{"matches":[{ "path", "line_start", "line_end", "snippet", "match_reason" }]}`
