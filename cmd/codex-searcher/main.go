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
		ServiceName: "codex-searcher",
		Port:        "8091",
	})

	srv := &Server{store: NewDocumentStore()}

	httpSrv.Handle("/api/v1/search/semantic", corsMiddleware(http.HandlerFunc(srv.handleSemantic)))
	httpSrv.Handle("/api/v1/search/keyword", corsMiddleware(http.HandlerFunc(srv.handleKeyword)))
	httpSrv.Handle("/api/v1/search/entity", corsMiddleware(http.HandlerFunc(srv.handleEntity)))
	httpSrv.Handle("/api/v1/search/hybrid", corsMiddleware(http.HandlerFunc(srv.handleHybrid)))

	httpSrv.ListenAndServe()
}
