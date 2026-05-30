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
		port = "8081"
	}

	store := NewTaskStore()
	react := NewReActEngine(store)
	srv := NewServer(store, react)

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

	http.HandleFunc("/api/v1/tasks", srv.corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			srv.handleSubmitTask(w, r)
		case http.MethodGet:
			srv.handleListTasks(w, r)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	}))

	http.HandleFunc("/api/v1/tasks/", srv.corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.HasSuffix(path, "/cancel") {
			srv.handleCancelTask(w, r)
			return
		}
		srv.handleGetTask(w, r)
	}))

	log.Printf("arcana-engine starting on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
