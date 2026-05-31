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
		ServiceName: "finops",
		Port:        "8105",
		DB:          dbConn,
	})

	store := NewFinopsStore(dbConn)
	srv := NewServer(store)

	httpSrv.HandleFunc("/api/v1/costs", srv.corsMiddleware(srv.handleCostsRoute))
	httpSrv.HandleFunc("/api/v1/costs/", srv.corsMiddleware(srv.handleCostsRoute))

	// Analytics endpoints for the FinOps Dashboard UI
	httpSrv.HandleFunc("/api/v1/finops/summary", srv.corsMiddleware(handleFinOpsSummary(dbConn)))
	httpSrv.HandleFunc("/api/v1/finops/cost-over-time", srv.corsMiddleware(handleCostOverTime(dbConn)))
	httpSrv.HandleFunc("/api/v1/finops/cost-by-model", srv.corsMiddleware(handleCostByModel(dbConn)))
	httpSrv.HandleFunc("/api/v1/finops/top-agents", srv.corsMiddleware(handleTopAgents(dbConn)))
	httpSrv.HandleFunc("/api/v1/finops/budget-utilization", srv.corsMiddleware(handleBudgetUtilization(dbConn)))
	httpSrv.HandleFunc("/api/v1/finops/teams", srv.corsMiddleware(handleTeams(dbConn)))
	httpSrv.HandleFunc("/api/v1/budgets", srv.corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/budgets" {
			return
		}
		if r.Method == http.MethodPost {
			srv.handleSetBudget(w, r)
		} else if r.Method == http.MethodGet {
			srv.handleListBudgets(w, r)
		} else {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	}))
	httpSrv.HandleFunc("/api/v1/budgets/", srv.corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/v1/budgets/")
		if path != "" {
			srv.handleBudgetTeam(w, r)
		}
	}))

	httpSrv.ListenAndServe()
}
