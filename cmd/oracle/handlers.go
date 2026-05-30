package main

import (
	"encoding/json"
	"net/http"
	"strings"
)

type Server struct {
	store     *PredictionStore
	predictor *Predictor
}

func NewServer(store *PredictionStore, predictor *Predictor) *Server {
	return &Server{store: store, predictor: predictor}
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

func (s *Server) handlePredict(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req PredictRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.ToolName == "" {
		writeError(w, http.StatusBadRequest, "tool_name is required")
		return
	}

	prediction := s.predictor.Predict(req.ToolName, req.Params, req.Context)
	writeJSON(w, http.StatusCreated, prediction)
}

func (s *Server) handleGetPrediction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/api/v1/predictions/")
	if id == "" {
		writeError(w, http.StatusBadRequest, "prediction id required")
		return
	}

	pred, ok := s.store.Get(id)
	if !ok {
		writeError(w, http.StatusNotFound, "prediction not found")
		return
	}
	writeJSON(w, http.StatusOK, pred)
}

func (s *Server) handleCalibrate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req CalibrateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.PredictionID == "" {
		writeError(w, http.StatusBadRequest, "prediction_id is required")
		return
	}
	if req.ActualOutput == nil {
		writeError(w, http.StatusBadRequest, "actual_output is required")
		return
	}

	pred, ok := s.store.Calibrate(req.PredictionID, req.ActualOutput)
	if !ok {
		writeError(w, http.StatusNotFound, "prediction not found")
		return
	}
	writeJSON(w, http.StatusOK, pred)
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	stats := s.store.Stats()
	writeJSON(w, http.StatusOK, stats)
}
