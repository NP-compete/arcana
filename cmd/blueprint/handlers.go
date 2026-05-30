package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type Server struct {
	store  *BlueprintStore
	engine *EngineClient
}

func NewServer(store *BlueprintStore, engine *EngineClient) *Server {
	return &Server{store: store, engine: engine}
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

func (s *Server) handleCreateBlueprint(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req CreateBlueprintRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.YAML == "" {
		writeError(w, http.StatusBadRequest, "yaml field is required")
		return
	}

	bp, err := ParseBlueprintYAML(req.YAML)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	s.store.Save(bp)
	saved, _ := s.store.Get(bp.Name)
	writeJSON(w, http.StatusCreated, saved)
}

func (s *Server) handleListBlueprints(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	bps := s.store.List()
	writeJSON(w, http.StatusOK, BlueprintListResponse{
		Blueprints: bps,
		Total:      len(bps),
	})
}

func (s *Server) handleGetBlueprint(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	name := extractBlueprintName(r.URL.Path)
	bp, ok := s.store.Get(name)
	if !ok {
		writeError(w, http.StatusNotFound, "blueprint not found")
		return
	}
	writeJSON(w, http.StatusOK, bp)
}

func (s *Server) handleDeleteBlueprint(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	name := extractBlueprintName(r.URL.Path)
	if !s.store.Delete(name) {
		writeError(w, http.StatusNotFound, "blueprint not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"deleted": name})
}

func (s *Server) handleExecuteBlueprint(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v1/blueprints/")
	name := strings.TrimSuffix(path, "/execute")
	bp, ok := s.store.Get(name)
	if !ok {
		writeError(w, http.StatusNotFound, "blueprint not found")
		return
	}

	taskIDs := make(map[string]string)
	for _, node := range bp.Nodes {
		input := map[string]interface{}{
			"blueprint": bp.Name,
			"node_id":   node.ID,
			"node_type": node.Type,
			"tools":     node.Tools,
			"skills":    node.Skills,
		}
		agent := fmt.Sprintf("%s/%s", bp.Name, node.ID)
		taskID, err := s.engine.SubmitTask(agent, input, node.Model)
		if err != nil {
			writeError(w, http.StatusBadGateway, "failed to submit task for node "+node.ID+": "+err.Error())
			return
		}
		taskIDs[node.ID] = taskID
	}

	writeJSON(w, http.StatusAccepted, ExecuteBlueprintResponse{
		BlueprintName: bp.Name,
		TaskIDs:       taskIDs,
		Status:        "executing",
	})
}

func extractBlueprintName(path string) string {
	path = strings.TrimPrefix(path, "/api/v1/blueprints/")
	if idx := strings.Index(path, "/"); idx >= 0 {
		path = path[:idx]
	}
	return path
}
