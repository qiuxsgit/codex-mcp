package main

import (
	"embed"
	"flag"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/qiuxsgit/codex-mcp/internal/db"
	"github.com/qiuxsgit/codex-mcp/internal/server"
)

//go:embed web/admin.html
var embedWebFS embed.FS

var _ http.FileSystem = (*emptyFS)(nil)

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

	// Prefer web dir next to the executable (for built binary); fallback to cwd (for go run).
	var adminFS http.FileSystem
	if exe, err := os.Executable(); err == nil {
		if resolved, err := filepath.EvalSymlinks(exe); err == nil {
			exe = resolved
		}
		exeDir := filepath.Dir(exe)
		webDir := filepath.Join(exeDir, "web")
		if _, err := os.Stat(webDir); err == nil {
			adminFS = http.FS(os.DirFS(webDir))
		}
	}
	if adminFS == nil {
		if wd, err := os.Getwd(); err == nil {
			webDir := filepath.Join(wd, "web")
			if _, err := os.Stat(webDir); err == nil {
				adminFS = http.FS(os.DirFS(webDir))
			}
		}
	}
	if adminFS == nil {
		// Use embedded admin.html (binary is self-contained)
		if sub, err := fs.Sub(embedWebFS, "web"); err == nil {
			adminFS = http.FS(sub)
		} else {
			adminFS = &emptyFS{}
		}
	}

	srv := server.New(addr, *ignoreFilePath, adminFS)
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

// emptyFS is an empty http.FileSystem so Admin UI is optional (no web dir).
type emptyFS struct{}

func (e *emptyFS) Open(name string) (http.File, error) {
	return nil, fs.ErrNotExist
}
