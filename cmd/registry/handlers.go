package main

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
)

func corsOrigin() string {
	if origin := os.Getenv("CORS_ORIGIN"); origin != "" {
		return origin
	}
	return "http://arcana-api.arcana.svc.cluster.local:8080"
}

type Server struct {
	store *RegistryStore
}

func NewServer(store *RegistryStore) *Server {
	return &Server{store: store}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", corsOrigin())
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, ErrorResponse{Error: msg})
}

func (s *Server) corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", corsOrigin())
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}

func (s *Server) handleCatalogList(w http.ResponseWriter, r *http.Request, catType CatalogType) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	entries := s.store.List(catType)
	writeJSON(w, http.StatusOK, map[string]interface{}{"entries": entries, "total": len(entries), "type": catType})
}

func (s *Server) handleCatalogRoute(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/catalog/")
	parts := strings.SplitN(path, "/", 2)

	if len(parts) == 1 && parts[0] == "stats" {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		writeJSON(w, http.StatusOK, s.store.Stats())
		return
	}

	if len(parts) == 1 {
		catType, ok := s.store.validType(parts[0])
		if !ok {
			writeError(w, http.StatusBadRequest, "invalid catalog type")
			return
		}
		if r.Method == http.MethodPost {
			var req RegisterRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeError(w, http.StatusBadRequest, "invalid JSON body")
				return
			}
			entry, err := s.store.Register(catType, req)
			if err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			writeJSON(w, http.StatusCreated, entry)
			return
		}
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	catType, ok := s.store.validType(parts[0])
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid catalog type")
		return
	}
	name := parts[1]

	switch r.Method {
	case http.MethodGet:
		s.handleCatalogList(w, r, catType)
	case http.MethodDelete:
		if !s.store.Deregister(catType, name) {
			writeError(w, http.StatusNotFound, "entry not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "deregistered", "name": name})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleCatalogType(w http.ResponseWriter, r *http.Request, catType CatalogType) {
	switch r.Method {
	case http.MethodGet:
		s.handleCatalogList(w, r, catType)
	case http.MethodPost:
		var req RegisterRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		entry, err := s.store.Register(catType, req)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, entry)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleCatalogAgents(w http.ResponseWriter, r *http.Request) {
	s.handleCatalogType(w, r, CatalogAgents)
}

func (s *Server) handleCatalogSkills(w http.ResponseWriter, r *http.Request) {
	s.handleCatalogType(w, r, CatalogSkills)
}

func (s *Server) handleCatalogTools(w http.ResponseWriter, r *http.Request) {
	s.handleCatalogType(w, r, CatalogTools)
}

func (s *Server) handleCatalogModels(w http.ResponseWriter, r *http.Request) {
	s.handleCatalogType(w, r, CatalogModels)
}

// handleVersions returns version history for a package.
// GET /api/v1/catalog/versions/{type}/{name}
func (s *Server) handleVersions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/catalog/versions/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		writeError(w, http.StatusBadRequest, "path must be /api/v1/catalog/versions/{type}/{name}")
		return
	}
	catType := parts[0]
	name := parts[1]
	if _, ok := s.store.validType(catType); !ok {
		writeError(w, http.StatusBadRequest, "invalid catalog type")
		return
	}
	versions := s.store.ListVersions(catType, name)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"package_type": catType,
		"package_name": name,
		"versions":     versions,
		"total":        len(versions),
	})
}

// handleApprovals lists pending approval requests.
// GET /api/v1/approvals?status=pending&type=skills&author=alice
func (s *Server) handleApprovals(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	q := r.URL.Query()
	status := q.Get("status")
	catType := q.Get("type")
	author := q.Get("author")
	approvals := s.store.ListApprovals(status, catType, author)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"approvals": approvals,
		"total":     len(approvals),
	})
}

// handleApprovalAction approves or rejects a pending package version.
// POST /api/v1/approvals/{id}/approve
// POST /api/v1/approvals/{id}/reject
func (s *Server) handleApprovalAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/approvals/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) < 2 || parts[0] == "" {
		writeError(w, http.StatusBadRequest, "path must be /api/v1/approvals/{id}/{action}")
		return
	}
	approvalID := parts[0]
	action := parts[1]
	if action != "approve" && action != "reject" {
		writeError(w, http.StatusBadRequest, "action must be approve or reject")
		return
	}

	var body struct {
		Reviewer string `json:"reviewer"`
		Comment  string `json:"comment"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if body.Reviewer == "" {
		body.Reviewer = "anonymous"
	}

	newStatus := "approved"
	if action == "reject" {
		newStatus = "rejected"
	}

	approval, err := s.store.UpdateApproval(approvalID, newStatus, body.Reviewer, body.Comment)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, err.Error())
		} else {
			writeError(w, http.StatusConflict, err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, approval)
}

// handleCatalogSearch performs full-text search across all packages.
// GET /api/v1/catalog/search?q=term&type=skill&sort=popular
func (s *Server) handleCatalogSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	q := r.URL.Query()
	query := q.Get("q")
	catType := q.Get("type")
	sortBy := q.Get("sort")
	results := s.store.Search(query, catType, sortBy)

	// Build simple suggestions from result names.
	seen := map[string]bool{}
	var suggestions []string
	for _, entry := range results {
		if !seen[entry.Name] && len(suggestions) < 5 {
			suggestions = append(suggestions, entry.Name)
			seen[entry.Name] = true
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"results":     results,
		"total":       len(results),
		"query":       query,
		"suggestions": suggestions,
	})
}

// handleSubmitVersion creates a new version for a package and optionally
// submits it for approval. This is a convenience so we have a way to
// populate the approval queue without a full publish pipeline.
// POST /api/v1/catalog/versions/{type}/{name}
func (s *Server) handleSubmitVersion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/catalog/versions/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		writeError(w, http.StatusBadRequest, "path must be /api/v1/catalog/versions/{type}/{name}")
		return
	}
	catType := parts[0]
	name := parts[1]
	if _, ok := s.store.validType(catType); !ok {
		writeError(w, http.StatusBadRequest, "invalid catalog type")
		return
	}

	var req struct {
		Version string `json:"version"`
		Author  string `json:"author"`
		Notes   string `json:"notes"`
		Approve bool   `json:"auto_approve"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Version == "" {
		writeError(w, http.StatusBadRequest, "version is required")
		return
	}
	if req.Author == "" {
		req.Author = "anonymous"
	}

	pv, approval, err := s.store.SubmitVersion(catType, name, req.Version, req.Author, req.Notes, req.Approve)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp := map[string]interface{}{
		"version": pv,
	}
	if approval != nil {
		resp["approval"] = approval
	}
	writeJSON(w, http.StatusCreated, resp)
}
