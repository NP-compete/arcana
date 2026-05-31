package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/NP-compete/arcana/pkg/db"
	"github.com/NP-compete/arcana/pkg/server"
)

func main() {
	dbConn := db.MustConnect()

	httpSrv := server.New(server.Config{
		ServiceName: "mesh",
		Port:        "8083",
		DB:          dbConn,
	})

	store := NewMeshStore(dbConn)
	srv := NewServer(store)

	httpSrv.HandleFunc("/.well-known/agent-card.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", corsOrigin())
		fmt.Fprint(w, `{"name":"arcana-mesh","version":"0.1.0","protocols":["a2a","acp"]}`)
	})

	httpSrv.HandleFunc("/api/v1/agents/register", srv.corsMiddleware(srv.handleRegisterAgent))
	httpSrv.HandleFunc("/api/v1/agents", srv.corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			srv.handleListAgents(w, r)
			return
		}
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}))
	httpSrv.HandleFunc("/api/v1/agents/", srv.corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/register") {
			return
		}
		if strings.HasSuffix(r.URL.Path, "/detail") {
			srv.handleAgentDetail(w, r)
			return
		}
		if r.Method == http.MethodDelete {
			srv.handleDeleteAgent(w, r)
			return
		}
		srv.handleGetAgent(w, r)
	}))

	httpSrv.HandleFunc("/api/v1/agents/suspend/", srv.corsMiddleware(srv.handleSuspendAgent))
	httpSrv.HandleFunc("/api/v1/agents/resume/", srv.corsMiddleware(srv.handleResumeAgent))

	httpSrv.HandleFunc("/api/v1/messages", srv.corsMiddleware(srv.handleSendMessage))
	httpSrv.HandleFunc("/api/v1/messages/", srv.corsMiddleware(srv.handleGetMessages))
	httpSrv.HandleFunc("/api/v1/delegate", srv.corsMiddleware(srv.handleDelegate))

	httpSrv.ListenAndServe()
}
