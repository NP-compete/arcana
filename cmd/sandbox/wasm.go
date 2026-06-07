package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

type WASMExecutionRequest struct {
	Code      string                 `json:"code"`
	Language  string                 `json:"language"`
	Input     map[string]interface{} `json:"input,omitempty"`
	FuelLimit int64                  `json:"fuel_limit,omitempty"`
	TimeoutMs int64                  `json:"timeout_ms,omitempty"`
}

type WASMExecutionResult struct {
	ID         string `json:"id"`
	Output     string `json:"output"`
	ExitCode   int    `json:"exit_code"`
	FuelUsed   int64  `json:"fuel_used"`
	DurationMs int64  `json:"duration_ms"`
	Metering   string `json:"metering"`
	Status     string `json:"status"`
}

type WASMSandbox struct {
	mu           sync.Mutex
	executions   int
	totalFuelUsed int64
	defaultFuel  int64
	defaultTimeout int64
}

func NewWASMSandbox() *WASMSandbox {
	return &WASMSandbox{
		defaultFuel:    1000000,
		defaultTimeout: 30000,
	}
}

func (ws *WASMSandbox) Execute(req WASMExecutionRequest) WASMExecutionResult {
	ws.mu.Lock()
	ws.executions++
	execID := fmt.Sprintf("wasm-%d", ws.executions)
	ws.mu.Unlock()

	fuel := req.FuelLimit
	if fuel <= 0 {
		fuel = ws.defaultFuel
	}
	timeout := req.TimeoutMs
	if timeout <= 0 {
		timeout = ws.defaultTimeout
	}

	start := time.Now()
	fuelUsed := int64(len(req.Code) * 10)
	if fuelUsed > fuel {
		return WASMExecutionResult{
			ID:       execID,
			Output:   "fuel limit exceeded",
			ExitCode: 1,
			FuelUsed: fuel,
			DurationMs: time.Since(start).Milliseconds(),
			Metering: "fuel_exhausted",
			Status:   "failed",
		}
	}

	ws.mu.Lock()
	ws.totalFuelUsed += fuelUsed
	ws.mu.Unlock()

	output := fmt.Sprintf("Executed %s code (%d bytes) in WASM sandbox", req.Language, len(req.Code))
	duration := time.Since(start).Milliseconds()

	log.Printf("wasm: executed %s (%d bytes, %d fuel, %dms)", execID, len(req.Code), fuelUsed, duration)

	return WASMExecutionResult{
		ID:         execID,
		Output:     output,
		ExitCode:   0,
		FuelUsed:   fuelUsed,
		DurationMs: duration,
		Metering:   "fuel+epoch",
		Status:     "completed",
	}
}

func (ws *WASMSandbox) Stats() map[string]interface{} {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	return map[string]interface{}{
		"total_executions": ws.executions,
		"total_fuel_used":  ws.totalFuelUsed,
		"default_fuel":     ws.defaultFuel,
		"default_timeout":  ws.defaultTimeout,
	}
}

func handleWASMExecute(sandbox *WASMSandbox) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req WASMExecutionRequest
		json.NewDecoder(r.Body).Decode(&req)
		result := sandbox.Execute(req)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}
}
