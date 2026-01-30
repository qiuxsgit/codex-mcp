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
