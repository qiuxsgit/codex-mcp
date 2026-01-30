package mcp

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/qiuxsgit/codex-mcp/internal/search"
)

const maxLimit = 20
const defaultLimit = 10

// SearchRequest is the JSON body for POST /mcp/search_internal_codebase.
type SearchRequest struct {
	Query     string `json:"query"`
	Language  string `json:"language"`
	PathHint  string `json:"path_hint"`
	Limit     int    `json:"limit"`
}

// SearchResponse is the JSON response.
type SearchResponse struct {
	Matches []search.Match `json:"matches"`
}

// Handler holds dependencies for the MCP search endpoint.
type Handler struct {
	IgnoreFilePath string
}

// ServeSearch handles POST /mcp/search_internal_codebase.
func (h *Handler) ServeSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if req.Query == "" {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(SearchResponse{Matches: []search.Match{}})
		return
	}
	limit := req.Limit
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	log.Printf("[search] limit=%d", limit)
	params := search.Params{
		Query:      req.Query,
		Language:   req.Language,
		PathHint:   req.PathHint,
		Limit:      limit,
		IgnorePath: h.IgnoreFilePath,
	}
	matches, err := search.Search(params)
	if err != nil {
		log.Printf("[search] error: %v", err)
		http.Error(w, "search failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(SearchResponse{Matches: matches}); err != nil {
		log.Printf("[search] encode error: %v", err)
	}
}
