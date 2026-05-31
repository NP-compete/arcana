package main

import (
	"github.com/NP-compete/arcana/pkg/db"
	"github.com/NP-compete/arcana/pkg/server"
)

func main() {
	dbConn := db.MustConnect()

	httpSrv := server.New(server.Config{
		ServiceName: "audit",
		Port:        "8100",
		DB:          dbConn,
	})

	store := NewAuditStore(dbConn)
	srv := NewServer(store)

	httpSrv.HandleFunc("/api/v1/audit", srv.corsMiddleware(srv.handleAuditRoute))
	httpSrv.HandleFunc("/api/v1/audit/", srv.corsMiddleware(srv.handleAuditRoute))

	httpSrv.ListenAndServe()
}
