package main

import (
	"fmt"
	"sync"
	"time"
)

type RegistryStore struct {
	mu      sync.RWMutex
	entries map[string]map[string]*CatalogEntry
}

func NewRegistryStore() *RegistryStore {
	s := &RegistryStore{
		entries: map[string]map[string]*CatalogEntry{
			string(CatalogAgents): {},
			string(CatalogSkills): {},
			string(CatalogTools):  {},
			string(CatalogModels): {},
		},
	}
	s.seedDefaults()
	return s
}

func (s *RegistryStore) seedDefaults() {
	now := time.Now().UTC()
	s.entries[string(CatalogAgents)]["research-agent"] = &CatalogEntry{
		Name: "research-agent", Type: CatalogAgents, Version: "1.0.0",
		Description: "Research and analysis agent", Metadata: map[string]interface{}{"team": "platform"},
		CreatedAt: now,
	}
	s.entries[string(CatalogSkills)]["summarize"] = &CatalogEntry{
		Name: "summarize", Type: CatalogSkills, Version: "2.1.0",
		Description: "Text summarization skill",
		Metadata: map[string]interface{}{"tier": "L2", "badge": "gold", "transfer": true},
		CreatedAt: now,
	}
	s.entries[string(CatalogTools)]["web_search"] = &CatalogEntry{
		Name: "web_search", Type: CatalogTools, Version: "1.0.0",
		Description: "Web search tool", Metadata: map[string]interface{}{"server": "web"},
		CreatedAt: now,
	}
	s.entries[string(CatalogModels)]["llama-3-8b"] = &CatalogEntry{
		Name: "llama-3-8b", Type: CatalogModels, Version: "1.0.0",
		Description: "Fine-tuned Llama 3 8B", Metadata: map[string]interface{}{"framework": "pytorch"},
		CreatedAt: now,
	}
}

func (s *RegistryStore) validType(t string) (CatalogType, bool) {
	switch CatalogType(t) {
	case CatalogAgents, CatalogSkills, CatalogTools, CatalogModels:
		return CatalogType(t), true
	default:
		return "", false
	}
}

func (s *RegistryStore) List(catType CatalogType) []CatalogEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	bucket := s.entries[string(catType)]
	result := make([]CatalogEntry, 0, len(bucket))
	for _, e := range bucket {
		result = append(result, *e)
	}
	return result
}

func (s *RegistryStore) Register(catType CatalogType, req RegisterRequest) (*CatalogEntry, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	version := req.Version
	if version == "" {
		version = "1.0.0"
	}
	entry := &CatalogEntry{
		Name: req.Name, Type: catType, Version: version,
		Description: req.Description, Metadata: req.Metadata,
		CreatedAt: time.Now().UTC(),
	}
	if entry.Metadata == nil {
		entry.Metadata = make(map[string]interface{})
	}
	s.entries[string(catType)][req.Name] = entry
	copy := *entry
	return &copy, nil
}

func (s *RegistryStore) Deregister(catType CatalogType, name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	bucket := s.entries[string(catType)]
	if _, ok := bucket[name]; !ok {
		return false
	}
	delete(bucket, name)
	return true
}

func (s *RegistryStore) Stats() CatalogStats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	stats := CatalogStats{}
	for t, bucket := range s.entries {
		count := len(bucket)
		stats.Total += count
		switch CatalogType(t) {
		case CatalogAgents:
			stats.Agents = count
		case CatalogSkills:
			stats.Skills = count
		case CatalogTools:
			stats.Tools = count
		case CatalogModels:
			stats.Models = count
		}
	}
	return stats
}
