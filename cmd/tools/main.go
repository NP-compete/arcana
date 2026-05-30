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
		port = "8096"
	}

	store := NewToolsStore()
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

	http.HandleFunc("/api/v1/tools/call", srv.corsMiddleware(srv.handleCallTool))
	http.HandleFunc("/api/v1/tools/translate", srv.corsMiddleware(srv.handleTranslate))
	http.HandleFunc("/api/v1/tools/", srv.corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		rest := strings.TrimPrefix(r.URL.Path, "/api/v1/tools/")
		if rest == "" || rest == "call" || rest == "translate" {
			return
		}
		srv.handleToolRoute(w, r)
	}))
	http.HandleFunc("/api/v1/tools", srv.corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/tools" {
			return
		}
		srv.handleListTools(w, r)
	}))

	log.Printf("arcana-tools starting on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
