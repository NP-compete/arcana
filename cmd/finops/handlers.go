package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

type Server struct {
	store *FinopsStore
}

func NewServer(store *FinopsStore) *Server {
	return &Server{store: store}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
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

func (s *Server) handleCosts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	q := r.URL.Query()
	report := s.store.CostReport(q.Get("team"), q.Get("agent"), q.Get("model"), q.Get("period"))
	writeJSON(w, http.StatusOK, report)
}

func (s *Server) handleRecordCost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req RecordCostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Agent == "" || req.Model == "" {
		writeError(w, http.StatusBadRequest, "agent and model are required")
		return
	}
	event := s.store.RecordCost(req)
	writeJSON(w, http.StatusCreated, event)
}

func (s *Server) handleCostTrends(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	days := 7
	if d := r.URL.Query().Get("days"); d != "" {
		days, _ = strconv.Atoi(d)
	}
	trends := s.store.CostTrends(r.URL.Query().Get("team"), days)
	writeJSON(w, http.StatusOK, map[string]interface{}{"trends": trends})
}

func (s *Server) handleSetBudget(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req SetBudgetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Team == "" {
		writeError(w, http.StatusBadRequest, "team is required")
		return
	}
	budget := s.store.SetBudget(req)
	writeJSON(w, http.StatusCreated, budget)
}

func (s *Server) handleListBudgets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"budgets": s.store.ListBudgets()})
}

func (s *Server) handleBudgetTeam(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	team := strings.TrimPrefix(r.URL.Path, "/api/v1/budgets/")
	if team == "" {
		writeError(w, http.StatusBadRequest, "team required")
		return
	}
	status, ok := s.store.GetBudgetStatus(team)
	if !ok {
		writeError(w, http.StatusNotFound, "budget not found for team")
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (s *Server) handleCostsRoute(w http.ResponseWriter, r *http.Request) {
	if strings.HasSuffix(r.URL.Path, "/record") {
		s.handleRecordCost(w, r)
		return
	}
	if strings.HasSuffix(r.URL.Path, "/trends") {
		s.handleCostTrends(w, r)
		return
	}
	s.handleCosts(w, r)
}
