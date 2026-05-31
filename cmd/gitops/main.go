package main

import (
	"github.com/NP-compete/arcana/pkg/server"
)

func main() {
	httpSrv := server.New(server.Config{
		ServiceName: "gitops",
		Port:        "8106",
	})

	store := NewGitopsStore()
	srv := NewServer(store)

	httpSrv.HandleFunc("/api/v1/promotions", srv.corsMiddleware(srv.handlePromotions))
	httpSrv.HandleFunc("/api/v1/promotions/", srv.corsMiddleware(srv.handlePromotions))

	httpSrv.ListenAndServe()
}
