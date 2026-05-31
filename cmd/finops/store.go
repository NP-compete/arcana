package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
)

type FinopsStore struct {
	db *sql.DB
}

func NewFinopsStore(db *sql.DB) *FinopsStore {
	return &FinopsStore{db: db}
}

func (s *FinopsStore) RecordCost(req RecordCostRequest) CostEvent {
	now := time.Now().UTC()
	event := CostEvent{
		ID:        uuid.New().String(),
		Agent:     req.Agent,
		Model:     req.Model,
		Team:      req.Team,
		Tokens:    req.Tokens,
		CostUSD:   req.CostUSD,
		Timestamp: now,
	}

	_, err := s.db.Exec(`
		INSERT INTO cost_events (agent, team, model, tokens, cost_usd, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		event.Agent, event.Team, event.Model, event.Tokens, event.CostUSD, event.Timestamp,
	)
	if err != nil {
		log.Printf("RecordCost: insert: %v", err)
	}
	return event
}

func (s *FinopsStore) SetBudget(req SetBudgetRequest) Budget {
	budget := Budget{
		Team: req.Team, Period: req.Period,
		TotalUSD: req.TotalUSD, DailyUSD: req.DailyUSD,
		PerAgentUSD: req.PerAgentUSD, Alerts: req.Alerts,
		FallbackPolicy: req.FallbackPolicy,
	}
	if budget.Period == "" {
		budget.Period = "monthly"
	}

	id := uuid.New().String()
	_, err := s.db.Exec(`
		INSERT INTO budgets (id, team, daily_usd, monthly_usd, per_agent_daily_usd, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		ON CONFLICT (team) DO UPDATE SET
			daily_usd = EXCLUDED.daily_usd,
			monthly_usd = EXCLUDED.monthly_usd,
			per_agent_daily_usd = EXCLUDED.per_agent_daily_usd,
			updated_at = NOW()`,
		id, req.Team, req.DailyUSD, req.TotalUSD, req.PerAgentUSD,
	)
	if err != nil {
		log.Printf("SetBudget: upsert: %v", err)
	}
	return budget
}

func (s *FinopsStore) ListBudgets() []Budget {
	rows, err := s.db.Query(`
		SELECT team, daily_usd, monthly_usd, per_agent_daily_usd
		FROM budgets ORDER BY team`)
	if err != nil {
		log.Printf("ListBudgets: %v", err)
		return []Budget{}
	}
	defer rows.Close()

	result := make([]Budget, 0)
	for rows.Next() {
		var b Budget
		err := rows.Scan(&b.Team, &b.DailyUSD, &b.TotalUSD, &b.PerAgentUSD)
		if err != nil {
			log.Printf("ListBudgets scan: %v", err)
			continue
		}
		b.Period = "monthly"
		result = append(result, b)
	}
	if err := rows.Err(); err != nil {
		log.Printf("ListBudgets rows: %v", err)
	}
	return result
}

func (s *FinopsStore) GetBudgetStatus(team string) (*BudgetStatus, bool) {
	var b struct {
		team       string
		dailyUSD   float64
		monthlyUSD float64
	}
	err := s.db.QueryRow(`SELECT team, daily_usd, monthly_usd FROM budgets WHERE team = $1`, team).
		Scan(&b.team, &b.dailyUSD, &b.monthlyUSD)
	if err == sql.ErrNoRows {
		return nil, false
	}
	if err != nil {
		log.Printf("GetBudgetStatus %s: %v", team, err)
		return nil, false
	}

	var used float64
	err = s.db.QueryRow(`SELECT COALESCE(SUM(cost_usd), 0) FROM cost_events WHERE team = $1`, team).Scan(&used)
	if err != nil {
		log.Printf("GetBudgetStatus sum %s: %v", team, err)
	}

	var dailyUsed float64
	today := time.Now().UTC().Format("2006-01-02")
	err = s.db.QueryRow(`SELECT COALESCE(SUM(cost_usd), 0) FROM cost_events WHERE team = $1 AND DATE(created_at) = $2`, team, today).Scan(&dailyUsed)
	if err != nil {
		log.Printf("GetBudgetStatus daily %s: %v", team, err)
	}

	daysInMonth := 30.0
	daysElapsed := float64(time.Now().UTC().Day())
	projected := used
	if daysElapsed > 0 {
		projected = used / daysElapsed * daysInMonth
	}

	return &BudgetStatus{
		Team: team, Period: "monthly",
		Used: used, Remaining: b.monthlyUSD - used,
		Projected: projected, DailyUsed: dailyUsed,
	}, true
}

func (s *FinopsStore) CostReport(team, agent, model, period string) CostReport {
	report := CostReport{
		Period:  period,
		ByTeam:  make(map[string]float64),
		ByAgent: make(map[string]float64),
		ByModel: make(map[string]float64),
	}
	if report.Period == "" {
		report.Period = "all"
	}

	query := `SELECT agent, team, model, cost_usd FROM cost_events WHERE 1=1`
	var args []interface{}
	idx := 1

	if team != "" {
		query += fmt.Sprintf(" AND team = $%d", idx)
		args = append(args, team)
		idx++
	}
	if agent != "" {
		query += fmt.Sprintf(" AND agent = $%d", idx)
		args = append(args, agent)
		idx++
	}
	if model != "" {
		query += fmt.Sprintf(" AND model = $%d", idx)
		args = append(args, model)
		idx++
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		log.Printf("CostReport: %v", err)
		return report
	}
	defer rows.Close()

	for rows.Next() {
		var a, t, m string
		var cost float64
		if err := rows.Scan(&a, &t, &m, &cost); err != nil {
			log.Printf("CostReport scan: %v", err)
			continue
		}
		report.TotalUSD += cost
		report.ByTeam[t] += cost
		report.ByAgent[a] += cost
		report.ByModel[m] += cost
		report.Events++
	}
	return report
}

func (s *FinopsStore) CostTrends(team string, days int) []CostTrendPoint {
	if days <= 0 {
		days = 7
	}

	now := time.Now().UTC()
	startDate := now.AddDate(0, 0, -(days - 1))

	query := `
		SELECT DATE(created_at) AS d, COALESCE(SUM(cost_usd), 0), COALESCE(SUM(tokens), 0)
		FROM cost_events
		WHERE created_at >= $1`
	args := []interface{}{startDate.Format("2006-01-02")}

	if team != "" {
		query += ` AND team = $2`
		args = append(args, team)
	}
	query += ` GROUP BY d ORDER BY d`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		log.Printf("CostTrends: %v", err)
		return []CostTrendPoint{}
	}
	defer rows.Close()

	dayMap := make(map[string]*CostTrendPoint)
	for rows.Next() {
		var d string
		var cost float64
		var tokens int64
		if err := rows.Scan(&d, &cost, &tokens); err != nil {
			log.Printf("CostTrends scan: %v", err)
			continue
		}
		dayMap[d] = &CostTrendPoint{Date: d, CostUSD: cost, Tokens: tokens}
	}

	result := make([]CostTrendPoint, 0, days)
	for i := days - 1; i >= 0; i-- {
		d := now.AddDate(0, 0, -i).Format("2006-01-02")
		if p, ok := dayMap[d]; ok {
			result = append(result, *p)
		} else {
			result = append(result, CostTrendPoint{Date: d})
		}
	}
	return result
}

