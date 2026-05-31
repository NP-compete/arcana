package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"
)

func handleFinOpsSummary(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		period := r.URL.Query().Get("period")
		if period == "" {
			period = "monthly"
		}

		var totalSpend float64
		var totalTokens int64
		var activeAgents int
		err := db.QueryRow(`SELECT COALESCE(SUM(cost_usd), 0), COALESCE(SUM(tokens), 0), COUNT(DISTINCT agent) FROM cost_events`).
			Scan(&totalSpend, &totalTokens, &activeAgents)
		if err != nil {
			log.Printf("handleFinOpsSummary: %v", err)
		}

		var totalBudget float64
		err = db.QueryRow(`SELECT COALESCE(SUM(monthly_usd), 0) FROM budgets`).Scan(&totalBudget)
		if err != nil {
			log.Printf("handleFinOpsSummary budgets: %v", err)
		}

		avgCostPerAgent := 0.0
		if activeAgents > 0 {
			avgCostPerAgent = totalSpend / float64(activeAgents)
		}

		summary := map[string]interface{}{
			"total_spend_usd":      totalSpend,
			"budget_remaining_usd": totalBudget - totalSpend,
			"active_agents":        activeAgents,
			"total_tokens":         totalTokens,
			"avg_cost_per_agent_usd": avgCostPerAgent,
			"period":               period,
		}
		writeJSON(w, http.StatusOK, summary)
	}
}

func handleCostOverTime(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		days := 30
		now := time.Now().UTC()
		startDate := now.AddDate(0, 0, -(days - 1))

		rows, err := db.Query(`
			SELECT DATE(created_at) AS d, COALESCE(SUM(cost_usd), 0), COALESCE(SUM(tokens), 0)
			FROM cost_events
			WHERE created_at >= $1
			GROUP BY d ORDER BY d`, startDate.Format("2006-01-02"))
		if err != nil {
			log.Printf("handleCostOverTime: %v", err)
			writeJSON(w, http.StatusOK, map[string]interface{}{"series": []interface{}{}, "period": r.URL.Query().Get("period")})
			return
		}
		defer rows.Close()

		dayMap := make(map[string]map[string]interface{})
		for rows.Next() {
			var d string
			var cost float64
			var tokens int64
			if err := rows.Scan(&d, &cost, &tokens); err != nil {
				log.Printf("handleCostOverTime scan: %v", err)
				continue
			}
			dayMap[d] = map[string]interface{}{"date": d, "cost_usd": cost, "tokens": tokens}
		}

		series := make([]map[string]interface{}, 0, days)
		for i := days - 1; i >= 0; i-- {
			d := now.AddDate(0, 0, -i).Format("2006-01-02")
			if entry, ok := dayMap[d]; ok {
				series = append(series, entry)
			} else {
				series = append(series, map[string]interface{}{"date": d, "cost_usd": 0.0, "tokens": 0})
			}
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"series": series,
			"period": r.URL.Query().Get("period"),
		})
	}
}

func handleCostByModel(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		rows, err := db.Query(`
			SELECT model, COALESCE(SUM(cost_usd), 0), COALESCE(SUM(tokens), 0), COUNT(*)
			FROM cost_events
			GROUP BY model
			ORDER BY SUM(cost_usd) DESC`)
		if err != nil {
			log.Printf("handleCostByModel: %v", err)
			writeJSON(w, http.StatusOK, map[string]interface{}{"models": []interface{}{}, "period": r.URL.Query().Get("period")})
			return
		}
		defer rows.Close()

		var totalCost float64
		type modelRow struct {
			Model    string  `json:"model"`
			CostUSD  float64 `json:"cost_usd"`
			Tokens   int64   `json:"tokens"`
			Requests int64   `json:"requests"`
		}
		var modelRows []modelRow
		for rows.Next() {
			var mr modelRow
			if err := rows.Scan(&mr.Model, &mr.CostUSD, &mr.Tokens, &mr.Requests); err != nil {
				log.Printf("handleCostByModel scan: %v", err)
				continue
			}
			totalCost += mr.CostUSD
			modelRows = append(modelRows, mr)
		}

		models := make([]map[string]interface{}, 0, len(modelRows))
		for _, mr := range modelRows {
			pct := 0.0
			if totalCost > 0 {
				pct = mr.CostUSD / totalCost * 100
			}
			models = append(models, map[string]interface{}{
				"model":      mr.Model,
				"cost_usd":   mr.CostUSD,
				"tokens":     mr.Tokens,
				"requests":   mr.Requests,
				"percentage": fmt.Sprintf("%.1f", pct),
			})
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"models": models,
			"period": r.URL.Query().Get("period"),
		})
	}
}

func handleTopAgents(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		rows, err := db.Query(`
			SELECT agent, team, COALESCE(SUM(cost_usd), 0), COALESCE(SUM(tokens), 0), COUNT(*)
			FROM cost_events
			GROUP BY agent, team
			ORDER BY SUM(cost_usd) DESC
			LIMIT 10`)
		if err != nil {
			log.Printf("handleTopAgents: %v", err)
			writeJSON(w, http.StatusOK, map[string]interface{}{"agents": []interface{}{}, "period": r.URL.Query().Get("period")})
			return
		}
		defer rows.Close()

		agents := make([]map[string]interface{}, 0)
		for rows.Next() {
			var agent, team string
			var cost float64
			var tokens, tasks int64
			if err := rows.Scan(&agent, &team, &cost, &tokens, &tasks); err != nil {
				log.Printf("handleTopAgents scan: %v", err)
				continue
			}
			agents = append(agents, map[string]interface{}{
				"agent":    agent,
				"team":     team,
				"cost_usd": cost,
				"tokens":   tokens,
				"tasks":    tasks,
			})
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"agents": agents,
			"period": r.URL.Query().Get("period"),
		})
	}
}

func handleBudgetUtilization(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		now := time.Now().UTC()
		daysInMonth := 30.0
		daysElapsed := float64(now.Day())
		if daysElapsed == 0 {
			daysElapsed = 1
		}

		rows, err := db.Query(`
			SELECT b.team, b.monthly_usd, COALESCE(SUM(c.cost_usd), 0) AS used
			FROM budgets b
			LEFT JOIN cost_events c ON c.team = b.team
			GROUP BY b.team, b.monthly_usd
			ORDER BY b.team`)
		if err != nil {
			log.Printf("handleBudgetUtilization: %v", err)
			writeJSON(w, http.StatusOK, map[string]interface{}{"teams": []interface{}{}, "period": r.URL.Query().Get("period")})
			return
		}
		defer rows.Close()

		teams := make([]map[string]interface{}, 0)
		for rows.Next() {
			var team string
			var budget, used float64
			if err := rows.Scan(&team, &budget, &used); err != nil {
				log.Printf("handleBudgetUtilization scan: %v", err)
				continue
			}
			remaining := budget - used
			utilPct := 0.0
			if budget > 0 {
				utilPct = used / budget * 100
			}
			projected := used / daysElapsed * daysInMonth

			status := "on_track"
			if utilPct < 20 {
				status = "under_budget"
			} else if projected > budget {
				status = "over_budget"
			}

			teams = append(teams, map[string]interface{}{
				"team":            team,
				"budget_usd":      budget,
				"used_usd":        used,
				"remaining_usd":   remaining,
				"utilization_pct": fmt.Sprintf("%.1f", utilPct),
				"projected_usd":   fmt.Sprintf("%.2f", projected),
				"status":          status,
			})
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"teams":  teams,
			"period": r.URL.Query().Get("period"),
		})
	}
}

func handleTeams(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		rows, err := db.Query(`
			SELECT b.team, b.monthly_usd, COUNT(DISTINCT c.agent) AS agents
			FROM budgets b
			LEFT JOIN cost_events c ON c.team = b.team
			GROUP BY b.team, b.monthly_usd
			ORDER BY b.team`)
		if err != nil {
			log.Printf("handleTeams: %v", err)
			writeJSON(w, http.StatusOK, map[string]interface{}{"teams": []interface{}{}, "total": 0})
			return
		}
		defer rows.Close()

		teams := make([]map[string]interface{}, 0)
		for rows.Next() {
			var team string
			var budget float64
			var agents int
			if err := rows.Scan(&team, &budget, &agents); err != nil {
				log.Printf("handleTeams scan: %v", err)
				continue
			}
			teams = append(teams, map[string]interface{}{
				"name":       team,
				"budget_usd": budget,
				"agents":     agents,
			})
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"teams": teams,
			"total": len(teams),
		})
	}
}
