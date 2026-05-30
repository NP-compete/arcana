package main

import "time"

type AgentStatus string

const (
	AgentStatusActive   AgentStatus = "active"
	AgentStatusIdle     AgentStatus = "idle"
	AgentStatusBusy     AgentStatus = "busy"
	AgentStatusOffline  AgentStatus = "offline"
)

type MeshAgent struct {
	Name         string      `json:"name"`
	Capabilities []string    `json:"capabilities"`
	Protocols    []string    `json:"protocols"`
	Status       AgentStatus `json:"status"`
	RegisteredAt time.Time   `json:"registered_at"`
}

type Message struct {
	ID        string                 `json:"id"`
	From      string                 `json:"from"`
	To        string                 `json:"to"`
	Payload   map[string]interface{} `json:"payload"`
	Protocol  string                 `json:"protocol"`
	CreatedAt time.Time              `json:"created_at"`
	Delivered bool                   `json:"delivered"`
}

type DelegationStatus string

const (
	DelegationPending   DelegationStatus = "pending"
	DelegationAccepted  DelegationStatus = "accepted"
	DelegationCompleted DelegationStatus = "completed"
	DelegationFailed    DelegationStatus = "failed"
)

type DelegationTask struct {
	ID          string                 `json:"id"`
	FromAgent   string                 `json:"from_agent"`
	ToAgent     string                 `json:"to_agent"`
	TaskType    string                 `json:"task_type"`
	Payload     map[string]interface{} `json:"payload"`
	Status      DelegationStatus       `json:"status"`
	Result      map[string]interface{} `json:"result,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

type RegisterAgentRequest struct {
	Name         string   `json:"name"`
	Capabilities []string `json:"capabilities"`
	Protocols    []string `json:"protocols"`
	Status       string   `json:"status,omitempty"`
}

type SendMessageRequest struct {
	From     string                 `json:"from"`
	To       string                 `json:"to"`
	Payload  map[string]interface{} `json:"payload"`
	Protocol string                 `json:"protocol"`
}

type DelegateRequest struct {
	FromAgent string                 `json:"from_agent"`
	ToAgent   string                 `json:"to_agent"`
	TaskType  string                 `json:"task_type"`
	Payload   map[string]interface{} `json:"payload"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
