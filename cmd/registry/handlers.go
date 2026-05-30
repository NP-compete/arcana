package main

import (
	"encoding/json"
	"net/http"
	"strings"
)

type Server struct {
	store *RegistryStore
}

func NewServer(store *RegistryStore) *Server {
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
