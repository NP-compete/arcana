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
		port = "8106"
	}

	store := NewGitopsStore()
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

	http.HandleFunc("/api/v1/promotions", srv.corsMiddleware(srv.handlePromotions))
	http.HandleFunc("/api/v1/promotions/", srv.corsMiddleware(srv.handlePromotions))

	log.Printf("arcana-gitops starting on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
