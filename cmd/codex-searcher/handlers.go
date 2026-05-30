package main

import (
	"encoding/json"
	"net/http"
)

type Server struct {
	store *DocumentStore
}

func (s *Server) handleSemantic(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req SemanticSearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	hits := s.store.SemanticSearch(req.Embedding, req.TopK, req.ShardIDs)
	writeJSON(w, http.StatusOK, SearchResponse{Hits: hits, TotalHits: len(hits), SearchType: "semantic"})
}

func (s *Server) handleKeyword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req KeywordSearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Query == "" {
		writeError(w, http.StatusBadRequest, "query is required")
		return
	}
	hits := s.store.KeywordSearch(req.Query, req.TopK, req.ShardIDs)
	writeJSON(w, http.StatusOK, SearchResponse{Hits: hits, TotalHits: len(hits), SearchType: "keyword"})
}

func (s *Server) handleEntity(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req EntitySearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if len(req.Entities) == 0 {
		writeError(w, http.StatusBadRequest, "entities is required")
		return
	}
	hits := s.store.EntitySearch(req.Entities, req.TopK)
	writeJSON(w, http.StatusOK, SearchResponse{Hits: hits, TotalHits: len(hits), SearchType: "entity"})
}

func (s *Server) handleHybrid(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req HybridSearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Query == "" {
		writeError(w, http.StatusBadRequest, "query is required")
		return
	}
	hits := s.store.HybridSearch(req.Query, req.Embedding, req.Weights, req.TopK, req.ShardIDs)
	writeJSON(w, http.StatusOK, SearchResponse{Hits: hits, TotalHits: len(hits), SearchType: "hybrid"})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
