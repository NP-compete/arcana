package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

type SchedulerStore struct {
	mu       sync.RWMutex
	agents   map[string]*AgentSchedule
	snapshots map[string]*Snapshot
}

func NewSchedulerStore() *SchedulerStore {
	s := &SchedulerStore{
		agents:    make(map[string]*AgentSchedule),
		snapshots: make(map[string]*Snapshot),
	}
	s.seedAgents()
	return s
}

func (s *SchedulerStore) seedAgents() {
	now := time.Now().UTC()
	idleSince := now.Add(-30 * time.Minute)
	s.agents["research-agent"] = &AgentSchedule{
		Name: "research-agent", Status: AgentRunning, Priority: PriorityStandard, LastActive: now,
	}
	s.agents["support-agent"] = &AgentSchedule{
		Name: "support-agent", Status: AgentIdle, Priority: PriorityBatch,
		IdleSince: &idleSince, LastActive: idleSince,
	}
}

func (s *SchedulerStore) ListAgents() []AgentSchedule {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]AgentSchedule, 0, len(s.agents))
	for _, a := range s.agents {
		result = append(result, *a)
	}
	return result
}

func (s *SchedulerStore) getOrCreate(name string) *AgentSchedule {
	if a, ok := s.agents[name]; ok {
		return a
	}
	now := time.Now().UTC()
	a := &AgentSchedule{Name: name, Status: AgentIdle, Priority: PriorityStandard, LastActive: now}
	s.agents[name] = a
	return a
}

func (s *SchedulerStore) Suspend(name string) (*AgentSchedule, *Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	a := s.getOrCreate(name)
	if a.Status == AgentSuspended {
		return nil, nil, fmt.Errorf("agent already suspended")
	}

	snapID := uuid.New().String()
	path := fmt.Sprintf("/snapshots/%s/%s.json", name, snapID)
	snap := &Snapshot{
		ID: snapID, AgentName: name, Path: path,
		CreatedAt: time.Now().UTC(), SizeBytes: 4096,
	}
	s.snapshots[snapID] = snap

	a.Status = AgentSuspended
	a.SnapshotPath = path
	a.LastActive = time.Now().UTC()
	copy := *a
	snapCopy := *snap
	return &copy, &snapCopy, nil
}

func (s *SchedulerStore) Resume(name string) (*AgentSchedule, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	a, ok := s.agents[name]
	if !ok {
		return nil, fmt.Errorf("agent not found")
	}
	if a.Status != AgentSuspended {
		return nil, fmt.Errorf("agent is not suspended")
	}

	a.Status = AgentRunning
	a.SnapshotPath = ""
	a.IdleSince = nil
	a.LastActive = time.Now().UTC()
	copy := *a
	return &copy, nil
}

func (s *SchedulerStore) SetPriority(name string, priority PriorityClass) (*AgentSchedule, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	a := s.getOrCreate(name)
	a.Priority = priority
	copy := *a
	return &copy, nil
}

func (s *SchedulerStore) ListSnapshots() []Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Snapshot, 0, len(s.snapshots))
	for _, snap := range s.snapshots {
		result = append(result, *snap)
	}
	return result
}
