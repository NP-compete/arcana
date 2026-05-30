package main

import "time"

type ToolCallStatus string

const (
	ToolCallPending   ToolCallStatus = "pending"
	ToolCallCompleted ToolCallStatus = "completed"
	ToolCallFailed    ToolCallStatus = "failed"
)

type ToolSchema struct {
	Name        string                 `json:"name"`
	Server      string                 `json:"server"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

type ToolServer struct {
	Name            string       `json:"name"`
	Tools           []ToolSchema `json:"tools"`
	ProtocolVersion string       `json:"protocol_version"`
}

type ToolCall struct {
	ID        string                 `json:"id"`
	Tool      string                 `json:"tool"`
	Server    string                 `json:"server"`
	Params    map[string]interface{} `json:"params"`
	AgentID   string                 `json:"agent_id"`
	Result    map[string]interface{} `json:"result,omitempty"`
	Status    ToolCallStatus         `json:"status"`
	LatencyMs int64                  `json:"latency_ms"`
}

type ToolCallRequest struct {
	ToolName string                 `json:"tool_name"`
	Server   string                 `json:"server"`
	Params   map[string]interface{} `json:"params"`
	AgentID  string                 `json:"agent_id"`
}

type TranslateRequest struct {
	SourceVersion string                 `json:"source_version"`
	TargetVersion string                 `json:"target_version"`
	ToolSchema    map[string]interface{} `json:"tool_schema"`
}

type TranslateResponse struct {
	TranslatedSchema map[string]interface{} `json:"translated_schema"`
	SourceVersion    string                 `json:"source_version"`
	TargetVersion    string                 `json:"target_version"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type toolCallRecord struct {
	call      ToolCall
	createdAt time.Time
}
