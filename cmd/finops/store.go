package main

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

type FinopsStore struct {
	mu     sync.RWMutex
	events []CostEvent
	budgets map[string]*Budget
}

func NewFinopsStore() *FinopsStore {
	return &FinopsStore{
		events:  make([]CostEvent, 0),
		budgets: make(map[string]*Budget),
	}
}

func (s *FinopsStore) RecordCost(req RecordCostRequest) CostEvent {
	s.mu.Lock()
	defer s.mu.Unlock()

	event := CostEvent{
		ID:        uuid.New().String(),
		Agent:     req.Agent,
		Model:     req.Model,
		Team:      req.Team,
		Tokens:    req.Tokens,
		CostUSD:   req.CostUSD,
		Timestamp: time.Now().UTC(),
	}
	s.events = append(s.events, event)
	return event
}

func (s *FinopsStore) SetBudget(req SetBudgetRequest) Budget {
	s.mu.Lock()
	defer s.mu.Unlock()

	budget := Budget{
		Team: req.Team, Period: req.Period,
		TotalUSD: req.TotalUSD, DailyUSD: req.DailyUSD,
		PerAgentUSD: req.PerAgentUSD, Alerts: req.Alerts,
		FallbackPolicy: req.FallbackPolicy,
	}
	if budget.Period == "" {
		budget.Period = "monthly"
	}
	s.budgets[req.Team] = &budget
	return budget
}

func (s *FinopsStore) ListBudgets() []Budget {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Budget, 0, len(s.budgets))
	for _, b := range s.budgets {
		result = append(result, *b)
	}
	return result
}

func (s *FinopsStore) GetBudgetStatus(team string) (*BudgetStatus, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	budget, ok := s.budgets[team]
	if !ok {
		return nil, false
	}

	var used, dailyUsed float64
	today := time.Now().UTC().Format("2006-01-02")
	agentCosts := make(map[string]float64)

	for _, e := range s.events {
		if e.Team != team {
			continue
		}
		used += e.CostUSD
		agentCosts[e.Agent] += e.CostUSD
		if e.Timestamp.Format("2006-01-02") == today {
			dailyUsed += e.CostUSD
		}
	}

	daysInMonth := 30.0
	daysElapsed := float64(time.Now().UTC().Day())
	projected := used
	if daysElapsed > 0 {
		projected = used / daysElapsed * daysInMonth
	}

	return &BudgetStatus{
		Team: team, Period: budget.Period,
		Used: used, Remaining: budget.TotalUSD - used,
		Projected: projected, DailyUsed: dailyUsed,
	}, true
}

func (s *FinopsStore) CostReport(team, agent, model, period string) CostReport {
	s.mu.RLock()
	defer s.mu.RUnlock()

	report := CostReport{
		Period:  period,
		ByTeam:  make(map[string]float64),
		ByAgent: make(map[string]float64),
		ByModel: make(map[string]float64),
	}

	for _, e := range s.events {
		if team != "" && e.Team != team {
			continue
		}
		if agent != "" && e.Agent != agent {
			continue
		}
		if model != "" && e.Model != model {
			continue
		}
		report.TotalUSD += e.CostUSD
		report.ByTeam[e.Team] += e.CostUSD
		report.ByAgent[e.Agent] += e.CostUSD
		report.ByModel[e.Model] += e.CostUSD
		report.Events++
	}
	if report.Period == "" {
		report.Period = "all"
	}
	return report
}

func (s *FinopsStore) CostTrends(team string, days int) []CostTrendPoint {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if days <= 0 {
		days = 7
	}

	points := make(map[string]*CostTrendPoint)
	now := time.Now().UTC()
	for i := days - 1; i >= 0; i-- {
		d := now.AddDate(0, 0, -i).Format("2006-01-02")
		points[d] = &CostTrendPoint{Date: d}
	}

	for _, e := range s.events {
		if team != "" && e.Team != team {
			continue
		}
		d := e.Timestamp.Format("2006-01-02")
		if p, ok := points[d]; ok {
			p.CostUSD += e.CostUSD
			p.Tokens += e.Tokens
		}
	}

	result := make([]CostTrendPoint, 0, len(points))
	for i := days - 1; i >= 0; i-- {
		d := now.AddDate(0, 0, -i).Format("2006-01-02")
		if p, ok := points[d]; ok {
			result = append(result, *p)
		}
	}
	return result
}
