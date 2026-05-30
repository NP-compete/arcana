package main

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

type TaskStore struct {
	mu    sync.RWMutex
	tasks map[string]*AgentTask
}

func NewTaskStore() *TaskStore {
	return &TaskStore{tasks: make(map[string]*AgentTask)}
}

func (s *TaskStore) Create(agent string, input map[string]interface{}, model ModelConfig) *AgentTask {
	now := time.Now().UTC()
	task := &AgentTask{
		ID:        uuid.New().String(),
		Agent:     agent,
		Input:     input,
		Status:    TaskStatusPending,
		Model:     model,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if task.Input == nil {
		task.Input = make(map[string]interface{})
	}

	s.mu.Lock()
	s.tasks[task.ID] = task
	s.mu.Unlock()
	return task
}

func (s *TaskStore) Get(id string) (*AgentTask, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	task, ok := s.tasks[id]
	if !ok {
		return nil, false
	}
	copy := *task
	return &copy, true
}

func (s *TaskStore) Update(id string, fn func(*AgentTask)) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	task, ok := s.tasks[id]
	if !ok {
		return false
	}
	fn(task)
	task.UpdatedAt = time.Now().UTC()
	return true
}

func (s *TaskStore) List(agent, status, since string, limit, offset int) []AgentTask {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var sinceTime time.Time
	if since != "" {
		if t, err := time.Parse(time.RFC3339, since); err == nil {
			sinceTime = t
		}
	}

	var result []AgentTask
	for _, task := range s.tasks {
		if agent != "" && task.Agent != agent {
			continue
		}
		if status != "" && string(task.Status) != status {
			continue
		}
		if !sinceTime.IsZero() && task.CreatedAt.Before(sinceTime) {
			continue
		}
		result = append(result, *task)
	}

	if offset >= len(result) {
		return []AgentTask{}
	}
	end := offset + limit
	if end > len(result) {
		end = len(result)
	}
	if limit <= 0 {
		limit = 50
		end = offset + limit
		if end > len(result) {
			end = len(result)
		}
	}
	return result[offset:end]
}

func (s *TaskStore) Count(agent, status, since string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var sinceTime time.Time
	if since != "" {
		if t, err := time.Parse(time.RFC3339, since); err == nil {
			sinceTime = t
		}
	}

	count := 0
	for _, task := range s.tasks {
		if agent != "" && task.Agent != agent {
			continue
		}
		if status != "" && string(task.Status) != status {
			continue
		}
		if !sinceTime.IsZero() && task.CreatedAt.Before(sinceTime) {
			continue
		}
		count++
	}
	return count
}
