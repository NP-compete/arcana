package main

import (
	"encoding/json"
	"net/http"
	"strings"
)

type Server struct {
	store *ToolsStore
}

func NewServer(store *ToolsStore) *Server {
	return &Server{store: store}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
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
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}

func (s *Server) handleListTools(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	servers := s.store.ListServers()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"servers": servers,
		"total":   len(servers),
	})
}

func (s *Server) handleToolRoute(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/tools/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		writeError(w, http.StatusBadRequest, "server and tool name required")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	schema, ok := s.store.GetToolSchema(parts[0], parts[1])
	if !ok {
		writeError(w, http.StatusNotFound, "tool not found")
		return
	}
	writeJSON(w, http.StatusOK, schema)
}

func (s *Server) handleCallTool(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req ToolCallRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.ToolName == "" || req.Server == "" {
		writeError(w, http.StatusBadRequest, "tool_name and server are required")
		return
	}
	call, err := s.store.CallTool(req.ToolName, req.Server, req.Params, req.AgentID)
	if err != nil {
		if err == errServerNotFound || err == errToolNotFound {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, call)
}

func (s *Server) handleTranslate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req TranslateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.SourceVersion == "" || req.TargetVersion == "" {
		writeError(w, http.StatusBadRequest, "source_version and target_version are required")
		return
	}
	if req.ToolSchema == nil {
		req.ToolSchema = make(map[string]interface{})
	}
	translated := s.store.TranslateSchema(req.SourceVersion, req.TargetVersion, req.ToolSchema)
	writeJSON(w, http.StatusOK, TranslateResponse{
		TranslatedSchema: translated,
		SourceVersion:    req.SourceVersion,
		TargetVersion:    req.TargetVersion,
	})
}
