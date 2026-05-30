package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

type Server struct {
	store  *TaskStore
	react  *ReActEngine
}

func NewServer(store *TaskStore, react *ReActEngine) *Server {
	return &Server{store: store, react: react}
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
	go s.react.Run(task.ID, req.Steps)

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
