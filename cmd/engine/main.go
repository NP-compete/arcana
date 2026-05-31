package main

import (
	"net/http"
	"strings"

	"github.com/NP-compete/arcana/pkg/db"
	"github.com/NP-compete/arcana/pkg/server"
)

func main() {
	dbConn := db.MustConnect()

	httpSrv := server.New(server.Config{
		ServiceName: "engine",
		Port:        "8081",
		DB:          dbConn,
	})

	store := NewTaskStore(dbConn)
	react := NewReActEngine(store)
	srv := NewServer(store, react)

	// Start Temporal worker in the background. Falls back to in-memory
	// execution if Temporal is unreachable.
	startWorker(store, react)

	httpSrv.HandleFunc("/api/v1/tasks", srv.corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			srv.handleSubmitTask(w, r)
		case http.MethodGet:
			srv.handleListTasks(w, r)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	}))

	httpSrv.HandleFunc("/api/v1/tasks/stages/", srv.corsMiddleware(srv.handleTaskStages))

	httpSrv.HandleFunc("/api/v1/tasks/", srv.corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.HasSuffix(path, "/cancel") {
			srv.handleCancelTask(w, r)
			return
		}
		srv.handleGetTask(w, r)
	}))

	httpSrv.ListenAndServe()
}
