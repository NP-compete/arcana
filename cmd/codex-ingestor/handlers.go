package main

import (
	"encoding/json"
	"net/http"
	"strings"
)

type Server struct {
	store *JobStore
}

func (s *Server) handleIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v1/ingest")
	path = strings.Trim(path, "/")

	if path == "batch" {
		s.handleBatchIngest(w, r)
		return
	}
	if path != "" {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	var req IngestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	job, doc, err := s.store.ProcessSingle(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]any{
		"job_id":  job.ID,
		"status":  job.Status,
		"doc_id":  doc.ID,
		"chunks":  len(doc.Chunks),
		"message": "document ingested successfully",
	})
}

func (s *Server) handleBatchIngest(w http.ResponseWriter, r *http.Request) {
	var req BatchIngestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if len(req.Documents) == 0 {
		writeError(w, http.StatusBadRequest, "documents array is required")
		return
	}

	job, docIDs := s.store.ProcessBatch(req.Documents)
	writeJSON(w, http.StatusAccepted, map[string]any{
		"job_id":     job.ID,
		"status":     job.Status,
		"doc_ids":    docIDs,
		"processed":  job.DocumentsProcessed,
		"chunks":     job.ChunksGenerated,
		"embeddings": job.EmbeddingsGenerated,
		"errors":     job.Errors,
	})
}

func (s *Server) handleJobs(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/ingest/jobs")
	path = strings.Trim(path, "/")

	if path == "" {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"jobs": s.store.ListJobs()})
		return
	}

	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	job, ok := s.store.GetJob(path)
	if !ok {
		writeError(w, http.StatusNotFound, "job not found")
		return
	}
	writeJSON(w, http.StatusOK, job)
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
