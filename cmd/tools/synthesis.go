package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type SynthesisRequest struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
	AgentID     string                 `json:"agent_id"`
}

type SynthesizedTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
	ServerCode  string                 `json:"server_code"`
	Status      string                 `json:"status"`
	TestResult  string                 `json:"test_result"`
	CreatedAt   string                 `json:"created_at"`
}

func (s *Server) handleSynthesizeTool(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST required")
		return
	}

	var req SynthesisRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Name == "" || req.Description == "" {
		writeError(w, http.StatusBadRequest, "name and description required")
		return
	}

	serverCode := fmt.Sprintf(`# Auto-generated MCP server for tool: %s
# Description: %s

from fastapi import FastAPI
app = FastAPI()

@app.post("/invoke")
async def invoke(params: dict):
    # TODO: implement %s logic
    return {"result": "synthesized tool output", "tool": "%s"}
`, req.Name, req.Description, req.Name, req.Name)

	tool := SynthesizedTool{
		Name:        req.Name,
		Description: req.Description,
		InputSchema: req.InputSchema,
		ServerCode:  serverCode,
		Status:      "synthesized",
		TestResult:  "sandbox_pending",
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
	}

	s.store.RegisterSynthesized(req.Name, tool)
	log.Printf("synthesis: generated tool %s for agent %s", req.Name, req.AgentID)

	writeJSON(w, http.StatusCreated, tool)
}
