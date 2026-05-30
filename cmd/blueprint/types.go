package main

import "time"

type Node struct {
	ID        string   `json:"id"`
	Type      string   `json:"type"`
	Model     string   `json:"model,omitempty"`
	Tools     []string `json:"tools,omitempty"`
	Skills    []string `json:"skills,omitempty"`
	DependsOn []string `json:"dependsOn,omitempty"`
}

type Edge struct {
	From        string `json:"from"`
	To          string `json:"to"`
	Passthrough bool   `json:"passthrough,omitempty"`
	Condition   string `json:"condition,omitempty"`
}

type Blueprint struct {
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Nodes       []Node    `json:"nodes"`
	Edges       []Edge    `json:"edges,omitempty"`
	Fallback    string    `json:"fallback,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CreateBlueprintRequest struct {
	YAML string `json:"yaml"`
}

type ExecuteBlueprintResponse struct {
	BlueprintName string            `json:"blueprint_name"`
	TaskIDs       map[string]string `json:"task_ids"`
	Status        string            `json:"status"`
}

type BlueprintListResponse struct {
	Blueprints []Blueprint `json:"blueprints"`
	Total      int         `json:"total"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
