package main

import (
	"sync"
	"time"
)

type BlueprintStore struct {
	mu         sync.RWMutex
	blueprints map[string]*Blueprint
}

func NewBlueprintStore() *BlueprintStore {
	return &BlueprintStore{blueprints: make(map[string]*Blueprint)}
}

func (s *BlueprintStore) Save(bp *Blueprint) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	if existing, ok := s.blueprints[bp.Name]; ok {
		bp.CreatedAt = existing.CreatedAt
	} else {
		bp.CreatedAt = now
	}
	bp.UpdatedAt = now
	s.blueprints[bp.Name] = bp
}

func (s *BlueprintStore) Get(name string) (*Blueprint, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	bp, ok := s.blueprints[name]
	if !ok {
		return nil, false
	}
	copy := *bp
	return &copy, true
}

func (s *BlueprintStore) Delete(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.blueprints[name]; !ok {
		return false
	}
	delete(s.blueprints, name)
	return true
}

func (s *BlueprintStore) List() []Blueprint {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Blueprint, 0, len(s.blueprints))
	for _, bp := range s.blueprints {
		result = append(result, *bp)
	}
	return result
}
