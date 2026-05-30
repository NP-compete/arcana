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
		port = "8105"
	}

	store := NewFinopsStore()
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

	http.HandleFunc("/api/v1/costs", srv.corsMiddleware(srv.handleCostsRoute))
	http.HandleFunc("/api/v1/costs/", srv.corsMiddleware(srv.handleCostsRoute))
	http.HandleFunc("/api/v1/budgets", srv.corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/budgets" {
			return
		}
		if r.Method == http.MethodPost {
			srv.handleSetBudget(w, r)
		} else if r.Method == http.MethodGet {
			srv.handleListBudgets(w, r)
		} else {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	}))
	http.HandleFunc("/api/v1/budgets/", srv.corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/v1/budgets/")
		if path != "" {
			srv.handleBudgetTeam(w, r)
		}
	}))

	log.Printf("arcana-finops starting on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
