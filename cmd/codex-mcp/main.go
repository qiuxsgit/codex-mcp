package main

import (
	"embed"
	"flag"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/qiuxsgit/codex-mcp/internal/db"
	"github.com/qiuxsgit/codex-mcp/internal/git"
	"github.com/qiuxsgit/codex-mcp/internal/server"
)

// all: required so that _next/ (Next.js static assets) is embedded; Go excludes names starting with '_' by default.
//go:embed all:web/admin-dist
var embedAdminFS embed.FS

func main() {
	port := flag.String("port", "6688", "server port")
	dbPath := flag.String("db-path", "./data/codex-mcp.db", "SQLite database path")
	ignoreFilePath := flag.String("ignore-file-path", "./data/codex-ignore", "path to gitignore-format ignore file")
	flag.Parse()

	addr := ":" + *port

	// Ensure data directory exists
	dataDir := filepath.Dir(*dbPath)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatalf("mkdir data: %v", err)
	}

	if err := db.Open(*dbPath); err != nil {
		log.Fatalf("db open: %v", err)
	}
	defer db.Close()

	// Admin UI: Next.js SSG export embedded under web/admin-dist.
	adminSub, _ := fs.Sub(embedAdminFS, "web/admin-dist")
	adminFS := http.FS(adminSub)

	srv := server.New(addr, *ignoreFilePath, adminFS)
	go runGitScheduler()

	baseURL := "http://localhost:" + *port
	log.Printf("codex-mcp listening on %s db=%s", addr, *dbPath)
	log.Printf("Admin: %s/admin", baseURL)
	log.Printf("MCP (Streamable HTTP, Inspector): %s/mcp", baseURL)
	log.Printf("MCP (REST): %s/mcp/search_internal_codebase", baseURL)
	log.Printf("推荐使用 npx @modelcontextprotocol/inspector 测试 MCP，连接地址填 %s/mcp，协议选 streamable-http", baseURL)
	if err := http.ListenAndServe(addr, srv.Router()); err != nil {
		log.Fatalf("serve: %v", err)
	}
}

// runGitScheduler runs git pull for directories with auto-update enabled, every 60s.
func runGitScheduler() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		list, err := db.ListDirectoriesForGitUpdate(time.Now().UTC())
		if err != nil {
			log.Printf("[git] list for update: %v", err)
			continue
		}
		for _, d := range list {
			if !git.IsGitRepo(d.Path) {
				continue
			}
			if err := git.Pull(d.Path); err != nil {
				log.Printf("[git] pull %s: %v", d.Path, err)
				continue
			}
			if err := db.UpdateDirectoryGitLastUpdated(d.ID, time.Now().UTC()); err != nil {
				log.Printf("[git] update last_updated: %v", err)
			}
		}
	}
}
