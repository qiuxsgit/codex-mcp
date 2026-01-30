# codex-mcp Makefile — 简化从源码构建与运行

BINARY     := codex-mcp
GO_PKG     := ./cmd/codex-mcp
ADMIN_DIST := cmd/codex-mcp/web/admin-dist
WEB_ADMIN  := web-admin

.PHONY: all build admin run clean help

# 默认目标：先构建 Admin，再构建 Go 二进制
all: build

# 构建 Go 二进制（依赖 admin，确保首次 make 即可完成全量构建）
build: admin
	@test -d "$(ADMIN_DIST)/_next" || (echo "ERROR: $(ADMIN_DIST)/_next missing, run 'make admin' first"; exit 1)
	go build -o $(BINARY) $(GO_PKG)
	@echo "Built: $(BINARY)"

# 构建 Admin（Next.js SSG）并复制到嵌入目录
admin:
	@if [ ! -d "$(WEB_ADMIN)/node_modules" ]; then \
		echo "Installing web-admin deps..."; \
		cd $(WEB_ADMIN) && npm ci --no-audit --no-fund && cd ..; \
	fi
	cd $(WEB_ADMIN) && npx next build
	rm -rf $(ADMIN_DIST)
	mkdir -p $(ADMIN_DIST)
	cp -R $(WEB_ADMIN)/out/. $(ADMIN_DIST)/
	@echo "Admin built -> $(ADMIN_DIST)"

# 启动服务（默认端口 6688）
run: build
	./$(BINARY)

# 清理：二进制、Admin 构建产物、Next 缓存
clean:
	rm -f $(BINARY)
	rm -rf $(ADMIN_DIST)
	rm -rf $(WEB_ADMIN)/.next $(WEB_ADMIN)/out
	@echo "Cleaned."

# 列出常用目标
help:
	@echo "codex-mcp Makefile"
	@echo ""
	@echo "  make          - 构建 Admin + Go 二进制（默认）"
	@echo "  make build    - 同上"
	@echo "  make admin    - 仅构建 Admin 静态资源"
	@echo "  make run      - 构建并启动服务（端口 6688）"
	@echo "  make clean    - 删除二进制与构建产物"
	@echo "  make help     - 显示此帮助"
