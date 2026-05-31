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
		ServiceName: "blueprint",
		Port:        "8088",
		DB:          dbConn,
	})

	store := NewBlueprintStore(dbConn)
	engine := NewEngineClient()
	srv := NewServer(store, engine)

	httpSrv.HandleFunc("/api/v1/blueprints", srv.corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			srv.handleCreateBlueprint(w, r)
		case http.MethodGet:
			srv.handleListBlueprints(w, r)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	}))

	httpSrv.HandleFunc("/api/v1/blueprints/", srv.corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.HasSuffix(path, "/execute") {
			srv.handleExecuteBlueprint(w, r)
			return
		}
		switch r.Method {
		case http.MethodGet:
			srv.handleGetBlueprint(w, r)
		case http.MethodDelete:
			srv.handleDeleteBlueprint(w, r)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	}))

	httpSrv.ListenAndServe()
}
