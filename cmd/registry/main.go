package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8104"
	}

	store := NewRegistryStore()
	srv := NewServer(store)

	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	})

	http.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	})

	http.HandleFunc("/api/v1/catalog/agents", srv.corsMiddleware(srv.handleCatalogAgents))
	http.HandleFunc("/api/v1/catalog/skills", srv.corsMiddleware(srv.handleCatalogSkills))
	http.HandleFunc("/api/v1/catalog/tools", srv.corsMiddleware(srv.handleCatalogTools))
	http.HandleFunc("/api/v1/catalog/models", srv.corsMiddleware(srv.handleCatalogModels))
	http.HandleFunc("/api/v1/catalog/stats", srv.corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		writeJSON(w, http.StatusOK, store.Stats())
	}))
	http.HandleFunc("/api/v1/catalog/", srv.corsMiddleware(srv.handleCatalogRoute))

	log.Printf("arcana-registry starting on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
