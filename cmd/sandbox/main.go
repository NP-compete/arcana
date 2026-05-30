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
		port = "8097"
	}

	store := NewSandboxStore()
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

	http.HandleFunc("/api/v1/exec", srv.corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/exec" {
			return
		}
		srv.handleExec(w, r)
	}))
	http.HandleFunc("/api/v1/exec/", srv.corsMiddleware(srv.handleExecRoute))

	log.Printf("arcana-sandbox starting on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
