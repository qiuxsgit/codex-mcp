package mcp

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/qiuxsgit/codex-mcp/internal/db"
	"github.com/qiuxsgit/codex-mcp/internal/search"
)

// SupportedLanguages is the list of language values accepted by search_internal_codebase (language param).
// Value is passed to ripgrep -t; label is for display.
var SupportedLanguages = []struct {
	Value string `json:"value"`
	Label string `json:"label"`
}{
	{Value: "java", Label: "Java"},
	{Value: "js", Label: "React/JSX"},
	{Value: "py", Label: "Python"},
	{Value: "go", Label: "Go"},
	{Value: "ts", Label: "TypeScript"},
	{Value: "javascript", Label: "JavaScript"},
	{Value: "csharp", Label: "C#"},
	{Value: "cpp", Label: "C++"},
	{Value: "rust", Label: "Rust"},
	{Value: "vue", Label: "Vue"},
	{Value: "swift", Label: "Swift"},
	{Value: "kotlin", Label: "Kotlin"},
	{Value: "ruby", Label: "Ruby"},
	{Value: "php", Label: "PHP"},
}

// SearchRoleOptions: role param for search_internal_codebase. 前端 = 前端业务+前端框架, 后端 = 后端业务+后端框架.
var SearchRoleOptions = []struct {
	Value       string   `json:"value"`
	Label       string   `json:"label"`
	DirectoryRoles []string `json:"directory_roles"` // directory tags that map to this search scope
}{
	{Value: "前端", Label: "Frontend", DirectoryRoles: []string{"前端业务", "前端框架"}},
	{Value: "后端", Label: "Backend", DirectoryRoles: []string{"后端业务", "后端框架"}},
}

const protocolVersion = "2024-11-05"
const serverName = "codex-mcp"
const serverVersion = "0.1.0"

// JSON-RPC 2.0 request
type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// JSON-RPC 2.0 response
type jsonRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *jsonRPCErr `json:"error,omitempty"`
}

type jsonRPCErr struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// MCP initialize params (client -> server)
type initParams struct {
	ProtocolVersion string `json:"protocolVersion"`
	Capabilities    struct{} `json:"capabilities"`
	ClientInfo      struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"clientInfo"`
}

// MCP initialize result (server -> client)
type initResult struct {
	ProtocolVersion string   `json:"protocolVersion"`
	Capabilities    initCaps `json:"capabilities"`
	ServerInfo      struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"serverInfo"`
}

type initCaps struct {
	Tools struct{} `json:"tools"`
}

// MCP tools/list result
type toolsListResult struct {
	Tools []toolDef `json:"tools"`
}

type toolDef struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	InputSchema inputSchema `json:"inputSchema"`
}

type inputSchema struct {
	Type       string            `json:"type"`
	Properties map[string]propDef `json:"properties"`
	Required   []string          `json:"required"`
}

type propDef struct {
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
}

// MCP tools/call params
type toolsCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

// MCP tools/call result
type toolsCallResult struct {
	Content  []contentItem `json:"content"`
	IsError  bool          `json:"isError,omitempty"`
}

type contentItem struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// ServeStreamableHTTP handles POST /mcp for MCP Streamable HTTP (JSON-RPC 2.0).
// Enables npx @modelcontextprotocol/inspector with transport "streamable-http" and URL http://localhost:PORT/mcp.
func (h *Handler) ServeStreamableHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req jsonRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONRPCError(w, nil, -32700, "Parse error")
		return
	}
	if req.JSONRPC != "2.0" || req.Method == "" {
		writeJSONRPCError(w, req.ID, -32600, "Invalid Request")
		return
	}

	var resp interface{}
	switch req.Method {
	case "initialize":
		resp = h.handleInitialize(req.Params)
	case "initialized":
		// notification, no response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		return
	case "tools/list":
		resp = h.handleToolsList()
	case "tools/call":
		resp = h.handleToolsCall(req.Params)
	default:
		writeJSONRPCError(w, req.ID, -32601, "Method not found")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  resp,
	})
}

func writeJSONRPCError(w http.ResponseWriter, id interface{}, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &jsonRPCErr{Code: code, Message: msg},
	})
}

func (h *Handler) handleInitialize(params json.RawMessage) *initResult {
	_ = params // optional client info
	return &initResult{
		ProtocolVersion: protocolVersion,
		Capabilities:    initCaps{},
		ServerInfo: struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		}{Name: serverName, Version: serverVersion},
	}
}

func (h *Handler) handleToolsList() *toolsListResult {
	return &toolsListResult{
		Tools: []toolDef{
			{
				Name: "search_internal_codebase",
				Description: "Search the configured codebase for exact text matches. Use this before implementing or refactoring: find where logic already exists, how APIs are used, or which files contain a pattern. Returns file path, line range, and snippet. Read-only and deterministic—no code generation. Prefer querying the codebase over guessing. Use get_supported_languages and get_supported_roles to get valid values for language and role params.",
				InputSchema: inputSchema{
					Type: "object",
					Properties: map[string]propDef{
						"query":     {Type: "string", Description: "Required. The exact string or pattern to search for in source files. Use concrete identifiers (e.g. function name, type name, error message) for best results."},
						"language":  {Type: "string", Description: "Optional. Filter by language. Call get_supported_languages for valid values (e.g. go, py, java, js, ts). Omit to search all languages."},
						"path_hint": {Type: "string", Description: "Optional. Substring that must appear in the file path (e.g. package name, directory). Use to restrict search to a specific module or layer."},
						"role":      {Type: "string", Description: "Optional. Limit scope to frontend or backend. Call get_supported_roles for valid values (前端 or 后端). Omit to search all."},
						"limit":     {Type: "number", Description: "Optional. Max number of matches to return. Default 10, max 20."},
					},
					Required: []string{"query"},
				},
			},
			{
				Name:        "get_supported_languages",
				Description: "Returns the list of language values accepted by search_internal_codebase (language parameter). Use these exact values when calling search to avoid empty results. Each item has value (pass this to search) and label (human-readable).",
				InputSchema: inputSchema{
					Type:       "object",
					Properties: map[string]propDef{},
					Required:   []string{},
				},
			},
			{
				Name:        "get_supported_roles",
				Description: "Returns the list of role values accepted by search_internal_codebase (role parameter). Use these exact values (前端 or 后端) when calling search to limit to frontend or backend code. Also returns directory_roles for reference (how directories are tagged).",
				InputSchema: inputSchema{
					Type:       "object",
					Properties: map[string]propDef{},
					Required:   []string{},
				},
			},
		},
	}
}

func (h *Handler) handleToolsCall(params json.RawMessage) *toolsCallResult {
	var p toolsCallParams
	if err := json.Unmarshal(params, &p); err != nil {
		return &toolsCallResult{
			Content: []contentItem{{Type: "text", Text: "invalid params"}},
			IsError: true,
		}
	}
	switch p.Name {
	case "search_internal_codebase":
		return h.handleSearch(p.Arguments)
	case "get_supported_languages":
		return h.handleGetSupportedLanguages()
	case "get_supported_roles":
		return h.handleGetSupportedRoles()
	default:
		return &toolsCallResult{
			Content: []contentItem{{Type: "text", Text: "unknown tool: " + p.Name}},
			IsError: true,
		}
	}
}

func (h *Handler) handleSearch(args json.RawMessage) *toolsCallResult {
	var reqArgs SearchRequest
	if len(args) > 0 {
		if err := json.Unmarshal(args, &reqArgs); err != nil {
			return &toolsCallResult{
				Content: []contentItem{{Type: "text", Text: "invalid arguments"}},
				IsError: true,
			}
		}
	}
	limit := reqArgs.Limit
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	log.Printf("[search] limit=%d", limit)
	searchParams := search.Params{
		Query:      reqArgs.Query,
		Language:   reqArgs.Language,
		PathHint:   reqArgs.PathHint,
		Role:       reqArgs.Role,
		Limit:      limit,
		IgnorePath: h.IgnoreFilePath,
	}
	matches, err := search.Search(searchParams)
	if err != nil {
		log.Printf("[search] error: %v", err)
		return &toolsCallResult{
			Content: []contentItem{{Type: "text", Text: "search failed: " + err.Error()}},
			IsError: true,
		}
	}
	out := SearchResponse{Matches: matches}
	text, _ := json.Marshal(out)
	return &toolsCallResult{
		Content: []contentItem{{Type: "text", Text: string(text)}},
	}
}

func (h *Handler) handleGetSupportedLanguages() *toolsCallResult {
	out := map[string]interface{}{"languages": SupportedLanguages}
	text, _ := json.Marshal(out)
	return &toolsCallResult{
		Content: []contentItem{{Type: "text", Text: string(text)}},
	}
}

func (h *Handler) handleGetSupportedRoles() *toolsCallResult {
	// directory_roles: full list of tags used when adding directories (for reference)
	out := map[string]interface{}{
		"search_roles":    SearchRoleOptions,
		"directory_roles": db.ValidRoles,
	}
	text, _ := json.Marshal(out)
	return &toolsCallResult{
		Content: []contentItem{{Type: "text", Text: string(text)}},
	}
}
