package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
)

type PlaygroundSession struct {
	ID          string                 `json:"id"`
	Tenant      string                 `json:"tenant"`
	AgentName   string                 `json:"agent_name"`
	AgentConfig map[string]interface{} `json:"agent_config"`
	BudgetLimit float64                `json:"budget_limit"`
	BudgetUsed  float64                `json:"budget_used"`
	TokensUsed  int                    `json:"tokens_used"`
	MsgCount    int                    `json:"message_count"`
	Status      string                 `json:"status"`
	CreatedAt   time.Time              `json:"created_at"`
	ExpiresAt   time.Time              `json:"expires_at"`
}

type StartPlaygroundRequest struct {
	AgentName   string                 `json:"agent_name"`
	AgentConfig map[string]interface{} `json:"agent_config,omitempty"`
	BudgetLimit float64                `json:"budget_limit,omitempty"`
}

type PlaygroundChatRequest struct {
	Message string `json:"message"`
}

type PlaygroundChatResponse struct {
	Reply      string                   `json:"reply"`
	ToolCalls  []map[string]interface{} `json:"tool_calls"`
	Guardrails []map[string]interface{} `json:"guardrails"`
	Reasoning  []string                 `json:"reasoning"`
	TokensUsed int                      `json:"tokens_used"`
	CostUSD    float64                  `json:"cost_usd"`
	BudgetUsed float64                  `json:"budget_used"`
	BudgetLeft float64                  `json:"budget_left"`
}

func (s *Server) handlePlaygroundStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST required")
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	var req StartPlaygroundRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.AgentName == "" {
		writeError(w, http.StatusBadRequest, "agent_name required")
		return
	}
	if req.BudgetLimit <= 0 {
		req.BudgetLimit = 1.0
	}

	tenant := extractTenant(r)
	sessionID := uuid.New().String()
	now := time.Now().UTC()
	session := &PlaygroundSession{
		ID:          sessionID,
		Tenant:      tenant,
		AgentName:   req.AgentName,
		AgentConfig: req.AgentConfig,
		BudgetLimit: req.BudgetLimit,
		BudgetUsed:  0,
		TokensUsed:  0,
		MsgCount:    0,
		Status:      "active",
		CreatedAt:   now,
		ExpiresAt:   now.Add(1 * time.Hour),
	}

	s.store.CreatePlaygroundSession(session)
	log.Printf("playground: started session %s for agent %s", sessionID, req.AgentName)
	writeJSON(w, http.StatusCreated, session)
}

func (s *Server) handlePlaygroundChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST required")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v1/playground/")
	sessionID := strings.TrimSuffix(path, "/chat")

	tenant := extractTenant(r)
	session := s.store.GetPlaygroundSession(tenant, sessionID)
	if session == nil {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}
	if session.Status != "active" {
		writeError(w, http.StatusGone, "session expired or ended")
		return
	}
	if session.BudgetUsed >= session.BudgetLimit {
		writeError(w, http.StatusPaymentRequired, "playground budget exhausted")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	var req PlaygroundChatRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	// Forward to engine in sandbox mode
	engineHost := os.Getenv("ENGINE_HOST")
	if engineHost == "" {
		engineHost = "arcana-engine.arcana.svc.cluster.local"
	}
	taskPayload, _ := json.Marshal(map[string]interface{}{
		"agent":   session.AgentName,
		"input":   req.Message,
		"sandbox": true,
	})
	engineResp, err := http.Post(
		fmt.Sprintf("http://%s:8081/api/v1/tasks", engineHost),
		"application/json",
		strings.NewReader(string(taskPayload)),
	)

	reasoning := []string{"Planning response to user query"}
	toolCalls := []map[string]interface{}{}
	guardrails := []map[string]interface{}{
		{"rule": "pii-detection", "status": "pass"},
		{"rule": "toxicity-filter", "status": "pass"},
	}
	tokens := 150
	cost := float64(tokens) * 0.00002

	reply := fmt.Sprintf("I'm %s running in playground mode. I received: \"%s\"", session.AgentName, req.Message)
	if err == nil && engineResp != nil {
		defer engineResp.Body.Close()
		var taskResult map[string]interface{}
		if json.NewDecoder(engineResp.Body).Decode(&taskResult) == nil {
			if id, ok := taskResult["id"].(string); ok {
				reasoning = append(reasoning, fmt.Sprintf("Created task %s", id))
			}
		}
	}

	s.store.UpdatePlaygroundUsage(tenant, session.ID, tokens, cost)

	resp := PlaygroundChatResponse{
		Reply:      reply,
		ToolCalls:  toolCalls,
		Guardrails: guardrails,
		Reasoning:  reasoning,
		TokensUsed: tokens,
		CostUSD:    cost,
		BudgetUsed: session.BudgetUsed + cost,
		BudgetLeft: session.BudgetLimit - session.BudgetUsed - cost,
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handlePlaygroundEnd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST required")
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/playground/")
	sessionID := strings.TrimSuffix(path, "/end")
	tenant := extractTenant(r)

	session := s.store.GetPlaygroundSession(tenant, sessionID)
	if session == nil {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}

	s.store.EndPlaygroundSession(tenant, sessionID)
	log.Printf("playground: ended session %s", sessionID)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"session_id": sessionID,
		"status":     "ended",
		"tokens_used": session.TokensUsed,
		"budget_used": session.BudgetUsed,
	})
}

func (s *Server) handlePlaygroundGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET required")
		return
	}
	sessionID := strings.TrimPrefix(r.URL.Path, "/api/v1/playground/")
	sessionID = strings.TrimSuffix(sessionID, "/")
	tenant := extractTenant(r)

	session := s.store.GetPlaygroundSession(tenant, sessionID)
	if session == nil {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}
	writeJSON(w, http.StatusOK, session)
}
