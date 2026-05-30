package main

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

type MeshStore struct {
	mu          sync.RWMutex
	agents      map[string]*MeshAgent
	messages    map[string][]Message
	delegations map[string]*DelegationTask
}

func NewMeshStore() *MeshStore {
	return &MeshStore{
		agents:      make(map[string]*MeshAgent),
		messages:    make(map[string][]Message),
		delegations: make(map[string]*DelegationTask),
	}
}

func (s *MeshStore) RegisterAgent(name string, capabilities, protocols []string, status AgentStatus) *MeshAgent {
	now := time.Now().UTC()
	agent := &MeshAgent{
		Name:         name,
		Capabilities: capabilities,
		Protocols:    protocols,
		Status:       status,
		RegisteredAt: now,
	}

	s.mu.Lock()
	s.agents[name] = agent
	s.mu.Unlock()
	return agent
}

func (s *MeshStore) GetAgent(name string) (*MeshAgent, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	a, ok := s.agents[name]
	if !ok {
		return nil, false
	}
	copy := *a
	return &copy, true
}

func (s *MeshStore) ListAgents() []MeshAgent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]MeshAgent, 0, len(s.agents))
	for _, a := range s.agents {
		result = append(result, *a)
	}
	return result
}

func (s *MeshStore) SendMessage(from, to string, payload map[string]interface{}, protocol string) (*Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.agents[to]; !ok {
		return nil, errAgentNotFound
	}

	msg := Message{
		ID:        uuid.New().String(),
		From:      from,
		To:        to,
		Payload:   payload,
		Protocol:  protocol,
		CreatedAt: time.Now().UTC(),
		Delivered: false,
	}
	if msg.Payload == nil {
		msg.Payload = make(map[string]interface{})
	}

	s.messages[to] = append(s.messages[to], msg)
	return &msg, nil
}

func (s *MeshStore) GetPendingMessages(agentName string) []Message {
	s.mu.RLock()
	defer s.mu.RUnlock()

	msgs := s.messages[agentName]
	pending := make([]Message, 0)
	for _, m := range msgs {
		if !m.Delivered {
			pending = append(pending, m)
		}
	}
	return pending
}

func (s *MeshStore) MarkMessagesDelivered(agentName string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.messages[agentName] {
		s.messages[agentName][i].Delivered = true
	}
}

func (s *MeshStore) CreateDelegation(fromAgent, toAgent, taskType string, payload map[string]interface{}) (*DelegationTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.agents[toAgent]; !ok {
		return nil, errAgentNotFound
	}

	now := time.Now().UTC()
	task := &DelegationTask{
		ID:        uuid.New().String(),
		FromAgent: fromAgent,
		ToAgent:   toAgent,
		TaskType:  taskType,
		Payload:   payload,
		Status:    DelegationPending,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if task.Payload == nil {
		task.Payload = make(map[string]interface{})
	}

	s.delegations[task.ID] = task
	return task, nil
}

func (s *MeshStore) GetDelegation(id string) (*DelegationTask, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.delegations[id]
	if !ok {
		return nil, false
	}
	copy := *t
	return &copy, true
}

func (s *MeshStore) UpdateDelegation(id string, fn func(*DelegationTask)) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	task, ok := s.delegations[id]
	if !ok {
		return false
	}
	fn(task)
	task.UpdatedAt = time.Now().UTC()
	return true
}

var errAgentNotFound = &agentNotFoundError{}

type agentNotFoundError struct{}

func (e *agentNotFoundError) Error() string {
	return "agent not found"
}
