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
		port = "8083"
	}

	store := NewMeshStore()
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

	http.HandleFunc("/.well-known/agent-card.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		fmt.Fprint(w, `{"name":"arcana-mesh","version":"0.1.0","protocols":["a2a","acp"]}`)
	})

	http.HandleFunc("/api/v1/agents/register", srv.corsMiddleware(srv.handleRegisterAgent))
	http.HandleFunc("/api/v1/agents", srv.corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			srv.handleListAgents(w, r)
			return
		}
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}))
	http.HandleFunc("/api/v1/agents/", srv.corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/register") {
			return
		}
		srv.handleGetAgent(w, r)
	}))

	http.HandleFunc("/api/v1/messages", srv.corsMiddleware(srv.handleSendMessage))
	http.HandleFunc("/api/v1/messages/", srv.corsMiddleware(srv.handleGetMessages))
	http.HandleFunc("/api/v1/delegate", srv.corsMiddleware(srv.handleDelegate))

	log.Printf("arcana-mesh starting on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
