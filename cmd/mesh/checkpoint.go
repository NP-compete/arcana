package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

type Checkpoint struct {
	ID        string                 `json:"id"`
	Tenant    string                 `json:"tenant"`
	AgentName string                 `json:"agent_name"`
	Version   int                    `json:"version"`
	Type      string                 `json:"type"`
	Config    map[string]interface{} `json:"config,omitempty"`
	Skills    []string               `json:"skills,omitempty"`
	Tags      []string               `json:"tags,omitempty"`
	Message   string                 `json:"message"`
	CreatedAt time.Time              `json:"created_at"`
}

func (s *Server) handleCheckpointCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST required")
		return
	}

	var req struct {
		AgentName string   `json:"agent_name"`
		Message   string   `json:"message"`
		Tags      []string `json:"tags,omitempty"`
		Type      string   `json:"type,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.AgentName == "" {
		writeError(w, http.StatusBadRequest, "agent_name required")
		return
	}

	tenant := extractTenant(r)
	agent, ok := s.store.GetAgent(tenant, req.AgentName)
	if !ok {
		writeError(w, http.StatusNotFound, "agent not found")
		return
	}

	checkpointType := req.Type
	if checkpointType == "" {
		checkpointType = "manual"
	}

	version := s.store.NextCheckpointVersion(tenant, req.AgentName)

	var config map[string]interface{}
	if agent.DeepConfig != nil {
		configBytes, _ := json.Marshal(agent.DeepConfig)
		json.Unmarshal(configBytes, &config)
	}

	cp := &Checkpoint{
		ID:        fmt.Sprintf("cp-%s-v%d", req.AgentName, version),
		Tenant:    tenant,
		AgentName: req.AgentName,
		Version:   version,
		Type:      checkpointType,
		Config:    config,
		Skills:    agent.Capabilities,
		Tags:      req.Tags,
		Message:   req.Message,
		CreatedAt: time.Now().UTC(),
	}

	s.store.SaveCheckpoint(cp)
	log.Printf("checkpoint: created %s for agent %s (v%d)", cp.ID, req.AgentName, version)
	writeJSON(w, http.StatusCreated, cp)
}

func (s *Server) handleCheckpointList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET required")
		return
	}

	agentName := r.URL.Query().Get("agent")
	if agentName == "" {
		writeError(w, http.StatusBadRequest, "agent query param required")
		return
	}

	tenant := extractTenant(r)
	checkpoints := s.store.ListCheckpoints(tenant, agentName)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"checkpoints": checkpoints,
		"total":       len(checkpoints),
	})
}

func (s *Server) handleCheckpointRestore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST required")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v1/checkpoints/restore/")
	cpID := strings.TrimSuffix(path, "/")

	tenant := extractTenant(r)
	cp := s.store.GetCheckpoint(tenant, cpID)
	if cp == nil {
		writeError(w, http.StatusNotFound, "checkpoint not found")
		return
	}

	log.Printf("checkpoint: restoring agent %s to version %d", cp.AgentName, cp.Version)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"restored":    true,
		"checkpoint":  cp.ID,
		"agent":       cp.AgentName,
		"version":     cp.Version,
		"restored_at": time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) handleCheckpointDiff(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET required")
		return
	}

	cpA := r.URL.Query().Get("a")
	cpB := r.URL.Query().Get("b")
	if cpA == "" || cpB == "" {
		writeError(w, http.StatusBadRequest, "a and b query params required")
		return
	}

	tenant := extractTenant(r)
	a := s.store.GetCheckpoint(tenant, cpA)
	b := s.store.GetCheckpoint(tenant, cpB)
	if a == nil || b == nil {
		writeError(w, http.StatusNotFound, "checkpoint not found")
		return
	}

	addedSkills := diffSlices(a.Skills, b.Skills)
	removedSkills := diffSlices(b.Skills, a.Skills)
	addedTags := diffSlices(a.Tags, b.Tags)
	removedTags := diffSlices(b.Tags, a.Tags)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"from":           cpA,
		"to":             cpB,
		"version_from":   a.Version,
		"version_to":     b.Version,
		"skills_added":   addedSkills,
		"skills_removed": removedSkills,
		"tags_added":     addedTags,
		"tags_removed":   removedTags,
		"config_changed": !jsonEqual(a.Config, b.Config),
	})
}

func diffSlices(old, new []string) []string {
	oldSet := map[string]bool{}
	for _, s := range old {
		oldSet[s] = true
	}
	diff := []string{}
	for _, s := range new {
		if !oldSet[s] {
			diff = append(diff, s)
		}
	}
	return diff
}

func jsonEqual(a, b interface{}) bool {
	ja, _ := json.Marshal(a)
	jb, _ := json.Marshal(b)
	return string(ja) == string(jb)
}
