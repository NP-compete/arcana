package main

import (
	"encoding/json"
	"net/http"
	"time"
)

type Server struct {
	profiles *ProfileStore
	shards   *ShardRegistry
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Query == "" {
		writeError(w, http.StatusBadRequest, "query is required")
		return
	}

	profileName := req.Profile
	if profileName == "" {
		profileName = "default"
	}
	profile, ok := s.profiles.Get(profileName)
	if !ok {
		writeError(w, http.StatusBadRequest, "unknown fusion profile")
		return
	}

	topK := req.TopK
	if topK <= 0 {
		topK = 10
	}

	start := time.Now()
	shardIDs := s.shards.SelectForQuery(req.Query, req.Filters)
	route := routeForProfile(profile)
	hits := buildLocalHits(req.Query, shardIDs, topK, profile)

	result := SearchResult{
		Query:     req.Query,
		Profile:   profile.Name,
		Route:     route,
		ShardIDs:  shardIDs,
		Results:   hits,
		TotalHits: len(hits),
		ElapsedMs: time.Since(start).Milliseconds(),
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleProfiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"profiles": s.profiles.List()})
}

func (s *Server) handleShards(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"shards": s.shards.List()})
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
