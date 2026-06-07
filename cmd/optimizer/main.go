package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/NP-compete/arcana/pkg/server"
)

type OptimizationPolicy struct {
	ID          string               `json:"id"`
	AgentName   string               `json:"agent_name"`
	EvalSuite   string               `json:"eval_suite"`
	Schedule    string               `json:"schedule"`
	Objectives  []OptimizationObj    `json:"objectives"`
	Constraints OptimizationConstraint `json:"constraints"`
	Strategies  []string             `json:"strategies"`
	MaxIter     int                  `json:"max_iterations"`
	Status      string               `json:"status"`
}

type OptimizationObj struct {
	Metric    string  `json:"metric"`
	Direction string  `json:"direction"`
	Weight    float64 `json:"weight"`
}

type OptimizationConstraint struct {
	MaxCostPerTask  float64 `json:"max_cost_per_task"`
	MinQualityScore float64 `json:"min_quality_score"`
}

type OptimizationRun struct {
	ID           string                   `json:"id"`
	PolicyID     string                   `json:"policy_id"`
	AgentName    string                   `json:"agent_name"`
	Iteration    int                      `json:"iteration"`
	BaselineScore float64                 `json:"baseline_score"`
	CurrentScore  float64                 `json:"current_score"`
	Improvement   float64                 `json:"improvement_pct"`
	Changes      []map[string]interface{} `json:"changes"`
	Status       string                   `json:"status"`
	StartedAt    string                   `json:"started_at"`
	CompletedAt  string                   `json:"completed_at,omitempty"`
}

type OptimizerStore struct {
	mu       sync.RWMutex
	policies map[string]*OptimizationPolicy
	runs     map[string]*OptimizationRun
}

func NewOptimizerStore() *OptimizerStore {
	return &OptimizerStore{
		policies: make(map[string]*OptimizationPolicy),
		runs:     make(map[string]*OptimizationRun),
	}
}

func main() {
	httpSrv := server.New(server.Config{
		ServiceName: "optimizer",
		Port:        "8113",
	})

	store := NewOptimizerStore()

	httpSrv.HandleFunc("/api/v1/optimization/policies", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			store.mu.RLock()
			policies := make([]OptimizationPolicy, 0, len(store.policies))
			for _, p := range store.policies {
				policies = append(policies, *p)
			}
			store.mu.RUnlock()
			writeJSON(w, http.StatusOK, map[string]interface{}{"policies": policies})
		case http.MethodPost:
			var policy OptimizationPolicy
			json.NewDecoder(r.Body).Decode(&policy)
			if policy.AgentName == "" {
				writeError(w, http.StatusBadRequest, "agent_name required")
				return
			}
			policy.ID = fmt.Sprintf("opt-%d", time.Now().UnixNano())
			policy.Status = "active"
			if policy.MaxIter <= 0 {
				policy.MaxIter = 10
			}
			store.mu.Lock()
			store.policies[policy.ID] = &policy
			store.mu.Unlock()
			writeJSON(w, http.StatusCreated, policy)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	}))

	httpSrv.HandleFunc("/api/v1/optimization/trigger", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "POST required")
			return
		}

		var req struct {
			PolicyID string `json:"policy_id"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		store.mu.RLock()
		policy, ok := store.policies[req.PolicyID]
		store.mu.RUnlock()
		if !ok {
			writeError(w, http.StatusNotFound, "policy not found")
			return
		}

		run := &OptimizationRun{
			ID:            fmt.Sprintf("orun-%d", time.Now().UnixNano()),
			PolicyID:      policy.ID,
			AgentName:     policy.AgentName,
			Iteration:     1,
			BaselineScore: 0.72,
			CurrentScore:  0.72,
			Status:        "running",
			StartedAt:     time.Now().UTC().Format(time.RFC3339),
			Changes:       make([]map[string]interface{}, 0),
		}

		store.mu.Lock()
		store.runs[run.ID] = run
		store.mu.Unlock()

		go runOptimization(store, run, policy)

		writeJSON(w, http.StatusAccepted, run)
	}))

	httpSrv.HandleFunc("/api/v1/optimization/runs", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		store.mu.RLock()
		runs := make([]OptimizationRun, 0, len(store.runs))
		for _, r := range store.runs {
			runs = append(runs, *r)
		}
		store.mu.RUnlock()
		writeJSON(w, http.StatusOK, map[string]interface{}{"runs": runs})
	}))

	httpSrv.ListenAndServe()
}

func runOptimization(store *OptimizerStore, run *OptimizationRun, policy *OptimizationPolicy) {
	log.Printf("optimizer: starting run %s for agent %s", run.ID, run.AgentName)

	for i := 0; i < policy.MaxIter; i++ {
		time.Sleep(100 * time.Millisecond)

		improvement := 0.02 + float64(i)*0.005
		change := map[string]interface{}{
			"iteration":   i + 1,
			"strategy":    policy.Strategies[i%max(len(policy.Strategies), 1)],
			"improvement": fmt.Sprintf("+%.1f%%", improvement*100),
		}

		store.mu.Lock()
		run.Iteration = i + 1
		run.CurrentScore = run.BaselineScore + improvement
		run.Changes = append(run.Changes, change)
		store.mu.Unlock()

		if run.CurrentScore >= policy.Constraints.MinQualityScore && i >= 2 {
			break
		}
	}

	store.mu.Lock()
	run.Improvement = (run.CurrentScore - run.BaselineScore) / run.BaselineScore * 100
	run.Status = "completed"
	run.CompletedAt = time.Now().UTC().Format(time.RFC3339)
	store.mu.Unlock()

	log.Printf("optimizer: run %s completed (%.1f%% improvement, %d iterations)",
		run.ID, run.Improvement, run.Iteration)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", os.Getenv("CORS_ORIGIN"))
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
