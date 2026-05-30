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
		port = "8100"
	}

	store := NewAuditStore()
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

	http.HandleFunc("/api/v1/audit", srv.corsMiddleware(srv.handleAuditRoute))
	http.HandleFunc("/api/v1/audit/", srv.corsMiddleware(srv.handleAuditRoute))

	log.Printf("arcana-audit starting on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
