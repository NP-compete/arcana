package main

import "time"

type AgentStatus string

const (
	AgentRunning   AgentStatus = "running"
	AgentSuspended AgentStatus = "suspended"
	AgentIdle      AgentStatus = "idle"
)

type PriorityClass string

const (
	PriorityCritical PriorityClass = "critical"
	PriorityStandard PriorityClass = "standard"
	PriorityBatch    PriorityClass = "batch"
)

type AgentSchedule struct {
	Name         string        `json:"name"`
	Status       AgentStatus   `json:"status"`
	Priority     PriorityClass `json:"priority"`
	IdleSince    *time.Time    `json:"idle_since,omitempty"`
	SnapshotPath string        `json:"snapshot_path,omitempty"`
	LastActive   time.Time     `json:"last_active"`
}

type Snapshot struct {
	ID        string    `json:"id"`
	AgentName string    `json:"agent_name"`
	Path      string    `json:"path"`
	CreatedAt time.Time `json:"created_at"`
	SizeBytes int64     `json:"size_bytes"`
}

type SetPriorityRequest struct {
	Agent    string        `json:"agent"`
	Priority PriorityClass `json:"priority"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
