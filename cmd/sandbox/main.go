package main

import (
	"net/http"

	"github.com/NP-compete/arcana/pkg/server"
)

func main() {
	httpSrv := server.New(server.Config{
		ServiceName: "sandbox",
		Port:        "8097",
	})

	store := NewSandboxStore()
	srv := NewServer(store)

	httpSrv.HandleFunc("/api/v1/exec", srv.corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/exec" {
			return
		}
		srv.handleExec(w, r)
	}))
	httpSrv.HandleFunc("/api/v1/exec/", srv.corsMiddleware(srv.handleExecRoute))

	httpSrv.ListenAndServe()
}
