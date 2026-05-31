package main

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"strings"
)

func corsOrigin() string {
	if origin := os.Getenv("CORS_ORIGIN"); origin != "" {
		return origin
	}
	return "http://arcana-api.arcana.svc.cluster.local:8080"
}

type Server struct {
	store *AuditStore
}

func NewServer(store *AuditStore) *Server {
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

func (s *Server) handleAppend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req AppendAuditRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Tenant == "" || req.Agent == "" || req.Action == "" {
		writeError(w, http.StatusBadRequest, "tenant, agent, and action are required")
		return
	}
	entry := s.store.Append(req)
	writeJSON(w, http.StatusCreated, entry)
}

func (s *Server) handleQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	q := AuditQuery{
		Agent:   r.URL.Query().Get("agent"),
		Action:  r.URL.Query().Get("action"),
		Since:   r.URL.Query().Get("since"),
		Until:   r.URL.Query().Get("until"),
		Verdict: r.URL.Query().Get("verdict"),
	}
	if l := r.URL.Query().Get("limit"); l != "" {
		q.Limit, _ = strconv.Atoi(l)
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		q.Offset, _ = strconv.Atoi(o)
	}
	entries, total := s.store.Query(q)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"entries": entries,
		"total":   total,
		"limit":   q.Limit,
		"offset":  q.Offset,
	})
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	stats := s.store.Stats()
	writeJSON(w, http.StatusOK, stats)
}

func (s *Server) handleGetByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/audit/")
	if id == "" || strings.Contains(id, "/") {
		writeError(w, http.StatusBadRequest, "audit entry id required")
		return
	}
	entry, ok := s.store.Get(id)
	if !ok {
		writeError(w, http.StatusNotFound, "audit entry not found")
		return
	}
	writeJSON(w, http.StatusOK, entry)
}

func (s *Server) handleAuditRoute(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/audit")
	switch {
	case path == "" || path == "/":
		if r.Method == http.MethodPost {
			s.handleAppend(w, r)
		} else if r.Method == http.MethodGet {
			s.handleQuery(w, r)
		} else {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	case path == "/stats":
		s.handleStats(w, r)
	default:
		s.handleGetByID(w, r)
	}
}
