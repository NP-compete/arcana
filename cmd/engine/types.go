package main

import "time"

type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCancelled TaskStatus = "cancelled"
)

type ModelConfig struct {
	Provider  string  `json:"provider,omitempty"`
	Model     string  `json:"model,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
	MaxTokens int     `json:"max_tokens,omitempty"`
}

type AgentTask struct {
	ID          string                 `json:"id"`
	Agent       string                 `json:"agent"`
	Input       map[string]interface{} `json:"input"`
	Status      TaskStatus             `json:"status"`
	Result      map[string]interface{} `json:"result,omitempty"`
	Model       ModelConfig            `json:"model"`
	TokensUsed  int                    `json:"tokens_used"`
	Cost        float64                `json:"cost"`
	CurrentStep int                    `json:"current_step"`
	Error       string                 `json:"error,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

type SubmitTaskRequest struct {
	Agent  string                 `json:"agent"`
	Input  map[string]interface{} `json:"input"`
	Model  ModelConfig            `json:"model"`
	Steps  int                    `json:"steps,omitempty"`
}

type TaskListResponse struct {
	Tasks  []AgentTask `json:"tasks"`
	Total  int         `json:"total"`
	Limit  int         `json:"limit"`
	Offset int         `json:"offset"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
