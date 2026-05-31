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
		ServiceName: "codex-router",
		Port:        "8090",
	})

	srv := &Server{
		profiles: NewProfileStore(),
		shards:   NewShardRegistry(),
	}

	httpSrv.Handle("/api/v1/search", corsMiddleware(http.HandlerFunc(srv.handleSearch)))
	httpSrv.Handle("/api/v1/profiles", corsMiddleware(http.HandlerFunc(srv.handleProfiles)))
	httpSrv.Handle("/api/v1/shards", corsMiddleware(http.HandlerFunc(srv.handleShards)))

	httpSrv.ListenAndServe()
}
