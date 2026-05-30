package main

import (
	"encoding/json"
	"net/http"
	"strings"
)

type Server struct {
	store *SandboxStore
}

func NewServer(store *SandboxStore) *Server {
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

func (s *Server) handleExec(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req ExecRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Language == "" || req.Code == "" {
		writeError(w, http.StatusBadRequest, "language and code are required")
		return
	}
	result := s.store.Execute(req)
	writeJSON(w, http.StatusAccepted, result)
}

func (s *Server) handleExecRoute(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/exec/")
	parts := strings.SplitN(path, "/", 2)
	id := parts[0]
	if id == "" {
		writeError(w, http.StatusBadRequest, "execution id required")
		return
	}

	if len(parts) == 2 && parts[1] == "logs" {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		logs, ok := s.store.GetLogs(id)
		if !ok {
			writeError(w, http.StatusNotFound, "execution not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"id": id, "logs": logs})
		return
	}

	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	result, ok := s.store.GetResult(id)
	if !ok {
		writeError(w, http.StatusNotFound, "execution not found")
		return
	}
	writeJSON(w, http.StatusOK, result)
}
