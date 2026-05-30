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
		port = "8103"
	}

	store := NewSchedulerStore()
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

	http.HandleFunc("/api/v1/agents", srv.corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/agents" {
			srv.handleListAgents(w, r)
		}
	}))
	http.HandleFunc("/api/v1/agents/", srv.corsMiddleware(srv.handleAgentAction))
	http.HandleFunc("/api/v1/snapshots", srv.corsMiddleware(srv.handleListSnapshots))
	http.HandleFunc("/api/v1/priority", srv.corsMiddleware(srv.handleSetPriority))

	log.Printf("arcana-scheduler starting on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
