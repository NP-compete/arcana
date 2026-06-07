package main

import (
	"net/http"
	"strings"

	"github.com/NP-compete/arcana/pkg/server"
)

func main() {
	httpSrv := server.New(server.Config{
		ServiceName: "tools",
		Port:        "8096",
	})

	store := NewToolsStore()
	srv := NewServer(store)

	httpSrv.HandleFunc("/.well-known/mcp.json", srv.corsMiddleware(srv.handleMCPDiscovery))
	httpSrv.HandleFunc("/mcp/invoke", srv.corsMiddleware(srv.handleMCPInvoke))
	httpSrv.HandleFunc("/api/v1/tools/synthesize", srv.corsMiddleware(srv.handleSynthesizeTool))
	httpSrv.HandleFunc("/api/v1/tools/call", srv.corsMiddleware(srv.handleCallTool))
	httpSrv.HandleFunc("/api/v1/tools/translate", srv.corsMiddleware(srv.handleTranslate))
	httpSrv.HandleFunc("/api/v1/tools/", srv.corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		rest := strings.TrimPrefix(r.URL.Path, "/api/v1/tools/")
		if rest == "" || rest == "call" || rest == "translate" {
			return
		}
		srv.handleToolRoute(w, r)
	}))
	httpSrv.HandleFunc("/api/v1/tools", srv.corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/tools" {
			return
		}
		srv.handleListTools(w, r)
	}))

	httpSrv.ListenAndServe()
}
