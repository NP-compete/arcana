package main

import "time"

type ExecStatus string

const (
	ExecPending   ExecStatus = "pending"
	ExecRunning   ExecStatus = "running"
	ExecCompleted ExecStatus = "completed"
	ExecFailed    ExecStatus = "failed"
	ExecTimeout   ExecStatus = "timeout"
)

type ExecRequest struct {
	Language  string                 `json:"language"`
	Code      string                 `json:"code"`
	TimeoutMs int                    `json:"timeout_ms"`
	Inputs    map[string]interface{} `json:"inputs"`
}

type ExecResult struct {
	ID               string     `json:"id"`
	Status           ExecStatus `json:"status"`
	Stdout           string     `json:"stdout"`
	Stderr           string     `json:"stderr"`
	ExitCode         int        `json:"exit_code"`
	DurationMs       int64      `json:"duration_ms"`
	MemoryUsedBytes  int64      `json:"memory_used_bytes"`
	Language         string     `json:"language"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type execRecord struct {
	result    ExecResult
	logs      []string
	createdAt time.Time
}
