package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8088"
	}

	store := NewBlueprintStore()
	engine := NewEngineClient()
	srv := NewServer(store, engine)

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

	http.HandleFunc("/api/v1/blueprints", srv.corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			srv.handleCreateBlueprint(w, r)
		case http.MethodGet:
			srv.handleListBlueprints(w, r)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	}))

	http.HandleFunc("/api/v1/blueprints/", srv.corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.HasSuffix(path, "/execute") {
			srv.handleExecuteBlueprint(w, r)
			return
		}
		switch r.Method {
		case http.MethodGet:
			srv.handleGetBlueprint(w, r)
		case http.MethodDelete:
			srv.handleDeleteBlueprint(w, r)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	}))

	log.Printf("arcana-blueprint starting on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
