package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

type Server struct {
	store *MeshStore
}

func NewServer(store *MeshStore) *Server {
	return &Server{store: store}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, ErrorResponse{Error: msg})
}

func (s *Server) corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}

func (s *Server) handleRegisterAgent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req RegisterAgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	status := AgentStatusActive
	if req.Status != "" {
		status = AgentStatus(req.Status)
	}

	agent := s.store.RegisterAgent(req.Name, req.Capabilities, req.Protocols, status)
	writeJSON(w, http.StatusCreated, agent)
}

func (s *Server) handleListAgents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	agents := s.store.ListAgents()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"agents": agents,
		"total":  len(agents),
	})
}

func (s *Server) handleGetAgent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	name := strings.TrimPrefix(r.URL.Path, "/api/v1/agents/")
	if name == "" || strings.Contains(name, "/") {
		writeError(w, http.StatusBadRequest, "agent name required")
		return
	}

	agent, ok := s.store.GetAgent(name)
	if !ok {
		writeError(w, http.StatusNotFound, "agent not found")
		return
	}
	writeJSON(w, http.StatusOK, agent)
}

func (s *Server) handleSendMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.From == "" || req.To == "" {
		writeError(w, http.StatusBadRequest, "from and to are required")
		return
	}
	if req.Protocol != "a2a" && req.Protocol != "acp" {
		writeError(w, http.StatusBadRequest, "protocol must be a2a or acp")
		return
	}

	msg, err := s.store.SendMessage(req.From, req.To, req.Payload, req.Protocol)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, msg)
}

func (s *Server) handleGetMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	agentName := strings.TrimPrefix(r.URL.Path, "/api/v1/messages/")
	if agentName == "" {
		writeError(w, http.StatusBadRequest, "agent name required")
		return
	}

	messages := s.store.GetPendingMessages(agentName)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"agent":    agentName,
		"messages": messages,
		"count":    len(messages),
	})
}

func (s *Server) handleDelegate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req DelegateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.FromAgent == "" || req.ToAgent == "" {
		writeError(w, http.StatusBadRequest, "from_agent and to_agent are required")
		return
	}
	if req.TaskType == "" {
		writeError(w, http.StatusBadRequest, "task_type is required")
		return
	}

	task, err := s.store.CreateDelegation(req.FromAgent, req.ToAgent, req.TaskType, req.Payload)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	go s.processDelegation(task.ID)

	writeJSON(w, http.StatusAccepted, task)
}

func (s *Server) processDelegation(taskID string) {
	s.store.UpdateDelegation(taskID, func(t *DelegationTask) {
		t.Status = DelegationAccepted
	})

	time.Sleep(50 * time.Millisecond)

	s.store.UpdateDelegation(taskID, func(t *DelegationTask) {
		t.Status = DelegationCompleted
		t.Result = map[string]interface{}{
			"delegated": true,
			"message":   "Task accepted and processed by target agent",
		}
	})
}
