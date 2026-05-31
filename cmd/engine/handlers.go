package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"go.temporal.io/sdk/client"
)

func corsOrigin() string {
	if origin := os.Getenv("CORS_ORIGIN"); origin != "" {
		return origin
	}
	return "http://arcana-api.arcana.svc.cluster.local:8080"
}

type Server struct {
	store  *TaskStore
	react  *ReActEngine
}

func NewServer(store *TaskStore, react *ReActEngine) *Server {
	return &Server{store: store, react: react}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", corsOrigin())
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
		w.Header().Set("Access-Control-Allow-Origin", corsOrigin())
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}

func (s *Server) handleSubmitTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req SubmitTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Agent == "" {
		writeError(w, http.StatusBadRequest, "agent is required")
		return
	}

	task := s.store.Create(req.Agent, req.Input, req.Model)

	if temporalClient != nil {
		// Durable execution via Temporal workflow.
		taskReq := TaskRequest{
			ID:    task.ID,
			Agent: req.Agent,
			Goal:  goalFromInput(req.Input),
			Input: req.Input,
		}
		wo := client.StartWorkflowOptions{
			ID:        "engine-task-" + task.ID,
			TaskQueue: taskQueue,
		}
		_, err := temporalClient.ExecuteWorkflow(r.Context(), wo, AgentTaskWorkflow, taskReq)
		if err != nil {
			log.Printf("WARNING: Temporal workflow start failed, falling back to in-memory: %v", err)
			go s.react.Run(task.ID, req.Steps)
		}
	} else {
		// In-memory fallback when Temporal is unavailable.
		go s.react.Run(task.ID, req.Steps)
	}

	writeJSON(w, http.StatusAccepted, task)
}

func (s *Server) handleGetTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	id := extractPathParam(r.URL.Path, "/api/v1/tasks/")
	if id == "" {
		writeError(w, http.StatusBadRequest, "task id required")
		return
	}

	task, ok := s.store.Get(id)
	if !ok {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func (s *Server) handleListTasks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	q := r.URL.Query()
	agent := q.Get("agent")
	status := q.Get("status")
	since := q.Get("since")
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	if limit <= 0 {
		limit = 50
	}

	tasks := s.store.List(agent, status, since, limit, offset)
	total := s.store.Count(agent, status, since)

	writeJSON(w, http.StatusOK, TaskListResponse{
		Tasks:  tasks,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	})
}

func (s *Server) handleCancelTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v1/tasks/")
	id := strings.TrimSuffix(path, "/cancel")
	if id == "" {
		writeError(w, http.StatusBadRequest, "task id required")
		return
	}

	task, ok := s.store.Get(id)
	if !ok {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}

	if task.Status == TaskStatusCompleted || task.Status == TaskStatusFailed {
		writeError(w, http.StatusConflict, "task already finished")
		return
	}
	if task.Status == TaskStatusCancelled {
		writeJSON(w, http.StatusOK, task)
		return
	}

	s.store.Update(id, func(t *AgentTask) {
		t.Status = TaskStatusCancelled
		t.Result = map[string]interface{}{"cancelled": true}
	})

	updated, _ := s.store.Get(id)
	writeJSON(w, http.StatusOK, updated)
}

func extractPathParam(path, prefix string) string {
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	rest := strings.TrimPrefix(path, prefix)
	if idx := strings.Index(rest, "/"); idx >= 0 {
		rest = rest[:idx]
	}
	return rest
}

// handleTaskStages returns execution stages for a running task.
// GET /api/v1/tasks/stages/{id}
func (s *Server) handleTaskStages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	taskID := extractPathParam(r.URL.Path, "/api/v1/tasks/stages/")
	if taskID == "" {
		writeError(w, http.StatusBadRequest, "task id required")
		return
	}

	task, ok := s.store.Get(taskID)
	if !ok {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}

	stages := []map[string]interface{}{
		{"step": 1, "name": "identify", "status": "pending", "duration_ms": 0},
		{"step": 2, "name": "plan", "status": "pending", "duration_ms": 0},
		{"step": 3, "name": "execute", "status": "pending", "duration_ms": 0},
		{"step": 4, "name": "format", "status": "pending", "duration_ms": 0},
	}

	currentStep := task.CurrentStep
	if currentStep == 0 {
		// Derive from status when CurrentStep has not been explicitly set.
		switch task.Status {
		case TaskStatusPending:
			currentStep = 0
		case TaskStatusRunning:
			currentStep = 2
		case TaskStatusCompleted:
			currentStep = 4
		case TaskStatusFailed, TaskStatusCancelled:
			currentStep = 2
		}
	}

	switch task.Status {
	case TaskStatusCompleted:
		for i := range stages {
			stages[i]["status"] = "completed"
			stages[i]["duration_ms"] = 150 + i*700
		}
	case TaskStatusFailed, TaskStatusCancelled:
		label := "failed"
		if task.Status == TaskStatusCancelled {
			label = "skipped"
		}
		for i := range stages {
			step := i + 1
			if step < currentStep {
				stages[i]["status"] = "completed"
				stages[i]["duration_ms"] = 150 + i*700
			} else if step == currentStep {
				stages[i]["status"] = label
			} else {
				stages[i]["status"] = "skipped"
			}
		}
	default:
		// pending or running
		for i := range stages {
			step := i + 1
			if step < currentStep {
				stages[i]["status"] = "completed"
				stages[i]["duration_ms"] = 150 + i*700
			} else if step == currentStep {
				stages[i]["status"] = "running"
			}
			// else stays "pending"
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"task_id":      taskID,
		"stages":       stages,
		"current_step": currentStep,
	})
}

// goalFromInput extracts a human-readable goal string from the task input map.
func goalFromInput(input map[string]interface{}) string {
	if input == nil {
		return ""
	}
	if g, ok := input["goal"]; ok {
		if s, ok := g.(string); ok {
			return s
		}
	}
	if t, ok := input["text"]; ok {
		if s, ok := t.(string); ok {
			return s
		}
	}
	return ""
}
