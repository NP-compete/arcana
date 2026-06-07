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

	monitor := NewHealthMonitor(store, srv.k8s)
	monitor.Start()

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
	httpSrv.HandleFunc("/api/v1/agents/health", srv.corsMiddleware(srv.handleAgentsHealthOverview))
	httpSrv.HandleFunc("/api/v1/agents/", srv.corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/register") {
			return
		}
		if strings.HasSuffix(r.URL.Path, "/detail") {
			srv.handleAgentDetail(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/health") {
			srv.handleAgentHealth(w, r)
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

	httpSrv.HandleFunc("/api/v1/playground/start", srv.corsMiddleware(srv.handlePlaygroundStart))
	httpSrv.HandleFunc("/api/v1/playground/", srv.corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/chat") {
			srv.handlePlaygroundChat(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/end") {
			srv.handlePlaygroundEnd(w, r)
			return
		}
		srv.handlePlaygroundGet(w, r)
	}))

	httpSrv.HandleFunc("/api/v1/checkpoints", srv.corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			srv.handleCheckpointList(w, r)
		case http.MethodPost:
			srv.handleCheckpointCreate(w, r)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	}))
	httpSrv.HandleFunc("/api/v1/checkpoints/restore/", srv.corsMiddleware(srv.handleCheckpointRestore))
	httpSrv.HandleFunc("/api/v1/checkpoints/diff", srv.corsMiddleware(srv.handleCheckpointDiff))

	httpSrv.HandleFunc("/api/v1/dependencies", srv.corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "GET required")
			return
		}
		tenant := extractTenant(r)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"edges": srv.store.GetDependencyGraph(tenant),
		})
	}))
	httpSrv.HandleFunc("/api/v1/dependencies/impact", srv.corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "GET required")
			return
		}
		agent := r.URL.Query().Get("agent")
		if agent == "" {
			writeError(w, http.StatusBadRequest, "agent query param required")
			return
		}
		tenant := extractTenant(r)
		writeJSON(w, http.StatusOK, srv.store.GetCascadeImpact(tenant, agent))
	}))

	httpSrv.HandleFunc("/api/v1/messages", srv.corsMiddleware(srv.handleSendMessage))
	httpSrv.HandleFunc("/api/v1/messages/", srv.corsMiddleware(srv.handleGetMessages))
	httpSrv.HandleFunc("/api/v1/delegate", srv.corsMiddleware(srv.handleDelegate))

	httpSrv.ListenAndServe()
}
