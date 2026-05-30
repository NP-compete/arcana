package main

import "time"

type CostEvent struct {
	ID        string    `json:"id"`
	Agent     string    `json:"agent"`
	Model     string    `json:"model"`
	Team      string    `json:"team"`
	Tokens    int64     `json:"tokens"`
	CostUSD   float64   `json:"cost_usd"`
	Timestamp time.Time `json:"timestamp"`
}

type Budget struct {
	Team           string                 `json:"team"`
	Period         string                 `json:"period"`
	TotalUSD       float64                `json:"total_usd"`
	DailyUSD       float64                `json:"daily_usd"`
	PerAgentUSD    float64                `json:"per_agent_usd"`
	Alerts         []string               `json:"alerts"`
	FallbackPolicy map[string]interface{} `json:"fallback_policy"`
}

type BudgetStatus struct {
	Team      string  `json:"team"`
	Period    string  `json:"period"`
	Used      float64 `json:"used"`
	Remaining float64 `json:"remaining"`
	Projected float64 `json:"projected"`
	DailyUsed float64 `json:"daily_used"`
}

type CostReport struct {
	Period    string             `json:"period"`
	TotalUSD  float64            `json:"total_usd"`
	ByTeam    map[string]float64 `json:"by_team"`
	ByAgent   map[string]float64 `json:"by_agent"`
	ByModel   map[string]float64 `json:"by_model"`
	Events    int                `json:"events"`
}

type CostTrendPoint struct {
	Date    string  `json:"date"`
	CostUSD float64 `json:"cost_usd"`
	Tokens  int64   `json:"tokens"`
}

type RecordCostRequest struct {
	Agent   string  `json:"agent"`
	Model   string  `json:"model"`
	Team    string  `json:"team"`
	Tokens  int64   `json:"tokens"`
	CostUSD float64 `json:"cost_usd"`
}

type SetBudgetRequest struct {
	Team           string                 `json:"team"`
	Period         string                 `json:"period"`
	TotalUSD       float64                `json:"total_usd"`
	DailyUSD       float64                `json:"daily_usd"`
	PerAgentUSD    float64                `json:"per_agent_usd"`
	Alerts         []string               `json:"alerts"`
	FallbackPolicy map[string]interface{} `json:"fallback_policy"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
