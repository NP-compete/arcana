package main

import "time"

type AgentStatus string

const (
	AgentStatusActive    AgentStatus = "active"
	AgentStatusIdle      AgentStatus = "idle"
	AgentStatusBusy      AgentStatus = "busy"
	AgentStatusOffline   AgentStatus = "offline"
	AgentStatusSuspended AgentStatus = "suspended"
)

type AgentType string

const (
	AgentTypeStandard AgentType = "create_agent"
	AgentTypeDeep     AgentType = "create_deep_agent"
)

type DeepAgentConfig struct {
	WorldModel     bool     `json:"world_model"`
	SkillGraph     bool     `json:"skill_graph"`
	Blueprint      string   `json:"blueprint,omitempty"`
	MemoryPolicy   string   `json:"memory_policy"`
	SubAgents      []string `json:"sub_agents,omitempty"`
	HITLEnabled    bool     `json:"hitl_enabled"`
	SelfImprove    bool     `json:"self_improve"`
	SystemPrompt   string   `json:"system_prompt,omitempty"`
	Temperature    float64  `json:"temperature"`
	MaxTokens      int      `json:"max_tokens,omitempty"`
	ModelCallLimit int      `json:"model_call_limit,omitempty"`
	ToolCallLimit  int      `json:"tool_call_limit,omitempty"`
	// Per-agent database mode: "shared" (platform DB), "dedicated" (per-agent DB), "external" (BYO)
	DBMode         string   `json:"db_mode,omitempty"`
	// External database URL (used when db_mode="external")
	ExternalDBURL  string   `json:"external_db_url,omitempty"`
}

type MeshAgent struct {
	Name         string           `json:"name"`
	Tenant       string           `json:"tenant"`
	AgentType    AgentType        `json:"agent_type"`
	Capabilities []string         `json:"capabilities"`
	Protocols    []string         `json:"protocols"`
	Status       AgentStatus      `json:"status"`
	RegisteredAt time.Time        `json:"registered_at"`
	DeepConfig   *DeepAgentConfig `json:"deep_config,omitempty"`
}

type Message struct {
	ID        string                 `json:"id"`
	Tenant    string                 `json:"tenant"`
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
	Tenant      string                 `json:"tenant"`
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
	Name         string           `json:"name"`
	AgentType    string           `json:"agent_type,omitempty"`
	Capabilities []string         `json:"capabilities"`
	Protocols    []string         `json:"protocols"`
	Status       string           `json:"status,omitempty"`
	DeepConfig   *DeepAgentConfig `json:"deep_config,omitempty"`
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
