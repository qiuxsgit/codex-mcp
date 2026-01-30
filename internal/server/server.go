package server

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/qiuxsgit/codex-mcp/internal/config"
	"github.com/qiuxsgit/codex-mcp/internal/db"
	"github.com/qiuxsgit/codex-mcp/internal/git"
	"github.com/qiuxsgit/codex-mcp/internal/mcp"
)

// Server holds config and serves HTTP.
type Server struct {
	Addr           string
	IgnoreFilePath string
	AdminFS        http.FileSystem
	mcpHandler     *mcp.Handler
}

// New creates a new Server.
func New(addr, ignoreFilePath string, adminFS http.FileSystem) *Server {
	return &Server{
		Addr:           addr,
		IgnoreFilePath: ignoreFilePath,
		AdminFS:        adminFS,
		mcpHandler:     &mcp.Handler{IgnoreFilePath: ignoreFilePath},
	}
}

// Router returns the HTTP handler (chi or plain mux). We use plain net/http.
func (s *Server) Router() http.Handler {
	mux := http.NewServeMux()

	// MCP Streamable HTTP (for inspector: npx @modelcontextprotocol/inspector, transport streamable-http, URL http://localhost:PORT/mcp)
	mux.HandleFunc("POST /mcp", s.mcpHandler.ServeStreamableHTTP)
	// MCP REST endpoint (direct POST to tool)
	mux.HandleFunc("POST /mcp/search_internal_codebase", s.mcpHandler.ServeSearch)

	// Admin UI: /admin -> index.html; /admin/* -> static files via FileServer (CSS/JS must get correct Content-Type)
	mux.HandleFunc("GET /admin", s.serveAdminIndex)
	mux.Handle("GET /admin/", http.StripPrefix("/admin", s.adminStaticHandler()))

	// API: directories
	mux.HandleFunc("GET /api/directories", s.apiListDirectories)
	mux.HandleFunc("POST /api/directories", s.apiAddDirectory)
	mux.HandleFunc("DELETE /api/directories/{id}", s.apiDeleteDirectory)
	mux.HandleFunc("PATCH /api/directories/{id}/enabled", s.apiSetDirectoryEnabled)
	mux.HandleFunc("PATCH /api/directories/{id}/git", s.apiSetDirectoryGitInterval)
	mux.HandleFunc("POST /api/directories/{id}/git/pull", s.apiDirectoryGitPull)

	// API: ignore file (gitignore format)
	mux.HandleFunc("GET /api/ignore-file", s.apiGetIgnoreFile)
	mux.HandleFunc("PUT /api/ignore-file", s.apiPutIgnoreFile)

	return mux
}

// serveAdminIndex serves index.html for GET /admin (no trailing slash).
func (s *Server) serveAdminIndex(w http.ResponseWriter, r *http.Request) {
	if s.AdminFS == nil {
		http.Error(w, "admin not configured", http.StatusNotFound)
		return
	}
	f, err := s.AdminFS.Open("index.html")
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	defer f.Close()
	data, _ := io.ReadAll(f)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data)
}

// stripPrefixFS wraps http.FileSystem so Open("/_next/...") becomes Open("_next/...").
// Go's FileServer passes paths with leading slash; embed.FS expects no leading slash.
type stripPrefixFS struct{ http.FileSystem }

func (f stripPrefixFS) Open(name string) (http.File, error) {
	name = strings.TrimPrefix(name, "/")
	if name == "" {
		name = "index.html"
	}
	return f.FileSystem.Open(name)
}

// adminStaticHandler serves files under /admin/ (after StripPrefix: path is "/" or "/_next/...").
// Use FileServer so CSS/JS get correct Content-Type from Go's built-in type map.
func (s *Server) adminStaticHandler() http.Handler {
	if s.AdminFS == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "admin not configured", http.StatusNotFound)
		})
	}
	fs := stripPrefixFS{FileSystem: s.AdminFS}
	h := http.FileServer(fs)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "" || path == "/" {
			r = r.Clone(r.Context())
			r.URL = &url.URL{Path: "/index.html"}
		}
		if strings.Contains(r.URL.Path, "..") {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		h.ServeHTTP(w, r)
	})
}

func (s *Server) apiListDirectories(w http.ResponseWriter, r *http.Request) {
	list, err := db.ListDirectories()
	if err != nil {
		log.Printf("[api] list directories: %v", err)
		http.Error(w, "list failed", http.StatusInternalServerError)
		return
	}
	if list == nil {
		list = []db.Directory{}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(list)
}

func (s *Server) apiAddDirectory(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name     string `json:"name"`
		Path     string `json:"path"`
		Language string `json:"language"`
		Role     string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if body.Name == "" || body.Path == "" {
		http.Error(w, "name and path required", http.StatusBadRequest)
		return
	}
	if body.Role != "" && !db.IsValidRole(body.Role) {
		http.Error(w, "role must be one of: 前端业务, 后端业务, 前端框架, 后端框架", http.StatusBadRequest)
		return
	}
	id, err := db.AddDirectory(body.Name, body.Path, body.Language, body.Role)
	if err != nil {
		log.Printf("[api] add directory: %v", err)
		http.Error(w, "add failed", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]int64{"id": id})
}

func (s *Server) apiDeleteDirectory(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	if idStr == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := db.DeleteDirectory(id); err != nil {
		log.Printf("[api] delete directory: %v", err)
		http.Error(w, "delete failed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) apiSetDirectoryEnabled(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	if idStr == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var body struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if err := db.SetDirectoryEnabled(id, body.Enabled); err != nil {
		log.Printf("[api] set enabled: %v", err)
		http.Error(w, "update failed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) apiGetIgnoreFile(w http.ResponseWriter, r *http.Request) {
	if s.IgnoreFilePath == "" {
		http.Error(w, "ignore file not configured", http.StatusNotFound)
		return
	}
	data, err := config.ReadIgnoreFile(s.IgnoreFilePath)
	if err != nil {
		log.Printf("[api] read ignore file: %v", err)
		http.Error(w, "read failed", http.StatusInternalServerError)
		return
	}
	if data == nil {
		data = []byte{}
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write(data)
}

func (s *Server) apiPutIgnoreFile(w http.ResponseWriter, r *http.Request) {
	if s.IgnoreFilePath == "" {
		http.Error(w, "ignore file not configured", http.StatusNotFound)
		return
	}
	data, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "body read failed", http.StatusBadRequest)
		return
	}
	if err := config.WriteIgnoreFile(s.IgnoreFilePath, data); err != nil {
		log.Printf("[api] write ignore file: %v", err)
		http.Error(w, "write failed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) apiSetDirectoryGitInterval(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	if idStr == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var body struct {
		AutoUpdateIntervalSec int `json:"auto_update_interval_sec"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if err := db.SetDirectoryGitInterval(id, body.AutoUpdateIntervalSec); err != nil {
		log.Printf("[api] set git interval: %v", err)
		http.Error(w, "update failed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) apiDirectoryGitPull(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	if idStr == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	dir, err := db.GetDirectoryByID(id)
	if err != nil {
		log.Printf("[api] get directory: %v", err)
		http.Error(w, "get failed", http.StatusInternalServerError)
		return
	}
	if dir == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if !git.IsGitRepo(dir.Path) {
		http.Error(w, "not a git repository", http.StatusBadRequest)
		return
	}
	if err := git.Pull(dir.Path); err != nil {
		log.Printf("[api] git pull %s: %v", dir.Path, err)
		http.Error(w, "git pull failed", http.StatusInternalServerError)
		return
	}
	now := time.Now().UTC()
	if err := db.UpdateDirectoryGitLastUpdated(id, now); err != nil {
		log.Printf("[api] update git last updated: %v", err)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"git_last_updated_at": now.Format(time.RFC3339)})
}
