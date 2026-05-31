package main

import (
	"encoding/json"
	"net/http"
	"os"
)

func corsOrigin() string {
	if origin := os.Getenv("CORS_ORIGIN"); origin != "" {
		return origin
	}
	return "http://arcana-api.arcana.svc.cluster.local:8080"
}

type Server struct {
	store *ScorerStore
}

func (s *Server) handleScore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req ScoreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if len(req.Results) == 0 {
		writeError(w, http.StatusBadRequest, "results array is required")
		return
	}
	profile := req.Profile
	if profile == "" {
		profile = "default"
	}
	scored := s.store.ScoreResults(req.Query, profile, req.Results)
	writeJSON(w, http.StatusOK, ScoreResponse{
		Query:   req.Query,
		Profile: profile,
		Results: scored,
	})
}

func (s *Server) handleContradictions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req ContradictionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.DocA == "" || req.DocB == "" {
		writeError(w, http.StatusBadRequest, "doc_a and doc_b are required")
		return
	}
	report := s.store.CheckContradiction(req.DocA, req.DocB)
	writeJSON(w, http.StatusOK, report)
}

func (s *Server) handleSupersession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req SupersessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.DocA == "" || req.DocB == "" {
		writeError(w, http.StatusBadRequest, "doc_a and doc_b are required")
		return
	}
	check := s.store.CheckSupersession(req.DocA, req.DocB)
	writeJSON(w, http.StatusOK, check)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", corsOrigin())
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
		w.Header().Set("Access-Control-Allow-Origin", corsOrigin())
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
