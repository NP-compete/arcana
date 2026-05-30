package main

import "time"

type AuditEntry struct {
	ID          string    `json:"id"`
	Timestamp   time.Time `json:"timestamp"`
	Tenant      string    `json:"tenant"`
	Agent       string    `json:"agent"`
	Action      string    `json:"action"`
	Tool        string    `json:"tool,omitempty"`
	InputHash   string    `json:"input_hash"`
	OutputHash  string    `json:"output_hash"`
	WardVerdict string    `json:"ward_verdict"`
	OPAVerdict  string    `json:"opa_verdict"`
	CostUSD     float64   `json:"cost_usd"`
	Tokens      int64     `json:"tokens"`
	Model       string    `json:"model,omitempty"`
	SessionID   string    `json:"session_id,omitempty"`
	WorkflowID  string    `json:"workflow_id,omitempty"`
	PrevHash    string    `json:"prev_hash"`
	EntryHash   string    `json:"entry_hash"`
}

type AuditQuery struct {
	Agent   string `json:"agent,omitempty"`
	Action  string `json:"action,omitempty"`
	Since   string `json:"since,omitempty"`
	Until   string `json:"until,omitempty"`
	Verdict string `json:"verdict,omitempty"`
	Limit   int    `json:"limit,omitempty"`
	Offset  int    `json:"offset,omitempty"`
}

type AuditStats struct {
	TotalEntries int64              `json:"total_entries"`
	ByAgent      map[string]int64   `json:"by_agent"`
	ByVerdict    map[string]int64   `json:"by_verdict"`
	TotalCost    float64            `json:"total_cost"`
}

type AppendAuditRequest struct {
	Timestamp   time.Time `json:"timestamp"`
	Tenant      string    `json:"tenant"`
	Agent       string    `json:"agent"`
	Action      string    `json:"action"`
	Tool        string    `json:"tool,omitempty"`
	InputHash   string    `json:"input_hash"`
	OutputHash  string    `json:"output_hash"`
	WardVerdict string    `json:"ward_verdict"`
	OPAVerdict  string    `json:"opa_verdict"`
	CostUSD     float64   `json:"cost_usd"`
	Tokens      int64     `json:"tokens"`
	Model       string    `json:"model,omitempty"`
	SessionID   string    `json:"session_id,omitempty"`
	WorkflowID  string    `json:"workflow_id,omitempty"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
