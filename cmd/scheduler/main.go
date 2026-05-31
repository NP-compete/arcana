package main

import (
	"net/http"

	"github.com/NP-compete/arcana/pkg/db"
	"github.com/NP-compete/arcana/pkg/server"
)

func main() {
	dbConn := db.MustConnect()

	httpSrv := server.New(server.Config{
		ServiceName: "scheduler",
		Port:        "8103",
		DB:          dbConn,
	})

	store := NewSchedulerStore(dbConn)
	srv := NewServer(store)

	httpSrv.HandleFunc("/api/v1/agents", srv.corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/agents" {
			srv.handleListAgents(w, r)
		}
	}))
	httpSrv.HandleFunc("/api/v1/agents/", srv.corsMiddleware(srv.handleAgentAction))
	httpSrv.HandleFunc("/api/v1/snapshots", srv.corsMiddleware(srv.handleListSnapshots))
	httpSrv.HandleFunc("/api/v1/priority", srv.corsMiddleware(srv.handleSetPriority))

	httpSrv.ListenAndServe()
}
