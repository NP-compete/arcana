package main

import (
	"net/http"
	"os"

	"github.com/NP-compete/arcana/pkg/server"
)

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	httpSrv := server.New(server.Config{
		ServiceName: "codex-ingestor",
		Port:        "8092",
	})

	srv := &Server{store: NewJobStore()}

	httpSrv.Handle("/api/v1/ingest", corsMiddleware(http.HandlerFunc(srv.handleIngest)))
	httpSrv.Handle("/api/v1/ingest/batch", corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		srv.handleBatchIngest(w, r)
	})))
	httpSrv.Handle("/api/v1/ingest/jobs/", corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		srv.handleJobs(w, r)
	})))
	httpSrv.Handle("/api/v1/ingest/jobs", corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"jobs": srv.store.ListJobs()})
	})))

	httpSrv.ListenAndServe()
}
