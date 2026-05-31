package main

import (
	"net/http"

	"github.com/NP-compete/arcana/pkg/db"
	"github.com/NP-compete/arcana/pkg/server"
)

func main() {
	dbConn := db.MustConnect()

	httpSrv := server.New(server.Config{
		ServiceName: "registry",
		Port:        "8104",
		DB:          dbConn,
	})

	store := NewRegistryStore(dbConn)
	srv := NewServer(store)

	httpSrv.HandleFunc("/api/v1/catalog/agents", srv.corsMiddleware(srv.handleCatalogAgents))
	httpSrv.HandleFunc("/api/v1/catalog/skills", srv.corsMiddleware(srv.handleCatalogSkills))
	httpSrv.HandleFunc("/api/v1/catalog/tools", srv.corsMiddleware(srv.handleCatalogTools))
	httpSrv.HandleFunc("/api/v1/catalog/models", srv.corsMiddleware(srv.handleCatalogModels))
	httpSrv.HandleFunc("/api/v1/catalog/stats", srv.corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		writeJSON(w, http.StatusOK, store.Stats())
	}))
	httpSrv.HandleFunc("/api/v1/catalog/search", srv.corsMiddleware(srv.handleCatalogSearch))
	httpSrv.HandleFunc("/api/v1/catalog/versions/", srv.corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			srv.handleVersions(w, r)
		case http.MethodPost:
			srv.handleSubmitVersion(w, r)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	}))
	httpSrv.HandleFunc("/api/v1/approvals", srv.corsMiddleware(srv.handleApprovals))
	httpSrv.HandleFunc("/api/v1/approvals/", srv.corsMiddleware(srv.handleApprovalAction))
	httpSrv.HandleFunc("/api/v1/catalog/", srv.corsMiddleware(srv.handleCatalogRoute))

	httpSrv.ListenAndServe()
}
