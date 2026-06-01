package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/NP-compete/arcana/pkg/db"
	"github.com/NP-compete/arcana/pkg/server"
	"github.com/NP-compete/arcana/pkg/temporal"
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

	// Start engine-local Temporal worker (ReAct loop workflows). Falls back
	// to in-memory execution if Temporal is unreachable.
	startWorker(store, react)

	// Start platform-level Temporal worker for cross-service workflows
	// (RunAgent, EvaluateSkill, PromoteAgent). Runs in a background
	// goroutine with graceful shutdown via context cancellation.
	workerCtx, workerCancel := context.WithCancel(context.Background())
	workerDone := make(chan struct{})
	go func() {
		defer close(workerDone)
		addr := os.Getenv("TEMPORAL_ADDRESS")
		if err := temporal.StartWorker(workerCtx, addr); err != nil {
			if workerCtx.Err() == nil {
				// Only log if the context was not cancelled (i.e. not a
				// graceful shutdown).
				log.Printf("WARNING: platform temporal worker stopped: %v", err)
			}
		}
	}()

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

	// Graceful shutdown: on SIGTERM/SIGINT, stop the platform worker and
	// let the HTTP server drain before exiting.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sigCh
		log.Println("shutdown signal received, stopping platform temporal worker")
		workerCancel()
		<-workerDone
	}()

	httpSrv.ListenAndServe()
}
