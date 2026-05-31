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
		ServiceName: "codex-scorer",
		Port:        "8093",
	})

	srv := &Server{store: NewScorerStore()}

	httpSrv.Handle("/api/v1/score", corsMiddleware(http.HandlerFunc(srv.handleScore)))
	httpSrv.Handle("/api/v1/contradictions", corsMiddleware(http.HandlerFunc(srv.handleContradictions)))
	httpSrv.Handle("/api/v1/supersession", corsMiddleware(http.HandlerFunc(srv.handleSupersession)))

	httpSrv.ListenAndServe()
}
