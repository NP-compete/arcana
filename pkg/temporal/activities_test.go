package temporal

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestActivities creates an Activities instance with a short timeout
// for testing. The caller must set the endpoint URLs after creation.
func newTestActivities() *Activities {
	return &Activities{
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
	}
}

// ---------------------------------------------------------------------------
// CheckGuardrails
// ---------------------------------------------------------------------------

func TestCheckGuardrails_Allowed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/evaluate", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var req guardrailRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		assert.Equal(t, "safe input", req.Input)

		json.NewEncoder(w).Encode(guardrailResponse{Allowed: true})
	}))
	defer srv.Close()

	acts := newTestActivities()
	acts.Endpoints.WardURL = srv.URL

	allowed, err := acts.CheckGuardrails(context.Background(), "safe input")
	require.NoError(t, err)
	assert.True(t, allowed)
}

func TestCheckGuardrails_Rejected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(guardrailResponse{Allowed: false, Reason: "policy violation"})
	}))
	defer srv.Close()

	acts := newTestActivities()
	acts.Endpoints.WardURL = srv.URL

	allowed, err := acts.CheckGuardrails(context.Background(), "unsafe input")
	require.NoError(t, err)
	assert.False(t, allowed)
}

func TestCheckGuardrails_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer srv.Close()

	acts := newTestActivities()
	acts.Endpoints.WardURL = srv.URL

	_, err := acts.CheckGuardrails(context.Background(), "input")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

// ---------------------------------------------------------------------------
// ExecuteTask
// ---------------------------------------------------------------------------

func TestExecuteTask_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/execute", r.URL.Path)

		var req executeTaskRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		assert.Equal(t, "agent-1", req.AgentID)
		assert.Equal(t, "summarize", req.TaskType)

		json.NewEncoder(w).Encode(executeTaskResponse{Result: "summary output"})
	}))
	defer srv.Close()

	acts := newTestActivities()
	acts.Endpoints.EngineURL = srv.URL

	result, err := acts.ExecuteTask(context.Background(), "agent-1", "summarize", "text to summarize")
	require.NoError(t, err)
	assert.Equal(t, "summary output", result)
}

func TestExecuteTask_RemoteError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(executeTaskResponse{Error: "model unavailable"})
	}))
	defer srv.Close()

	acts := newTestActivities()
	acts.Endpoints.EngineURL = srv.URL

	_, err := acts.ExecuteTask(context.Background(), "agent-1", "summarize", "input")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "model unavailable")
}

// ---------------------------------------------------------------------------
// StoreMemory
// ---------------------------------------------------------------------------

func TestStoreMemory_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/memories", r.URL.Path)

		var req storeMemoryRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		assert.Equal(t, "agent-1", req.AgentID)
		assert.Equal(t, "the result", req.Content)

		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	acts := newTestActivities()
	acts.Endpoints.MemoryURL = srv.URL

	err := acts.StoreMemory(context.Background(), "agent-1", "the result")
	require.NoError(t, err)
}

func TestStoreMemory_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("service unavailable"))
	}))
	defer srv.Close()

	acts := newTestActivities()
	acts.Endpoints.MemoryURL = srv.URL

	err := acts.StoreMemory(context.Background(), "agent-1", "result")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "503")
}

// ---------------------------------------------------------------------------
// GetSkill
// ---------------------------------------------------------------------------

func TestGetSkill_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/skills/code-review", r.URL.Path)

		json.NewEncoder(w).Encode(SkillDefinition{
			Name:        "code-review",
			Description: "Reviews code",
			Version:     "2.0",
		})
	}))
	defer srv.Close()

	acts := newTestActivities()
	acts.Endpoints.SkillsURL = srv.URL

	skill, err := acts.GetSkill(context.Background(), "code-review")
	require.NoError(t, err)
	assert.Equal(t, "code-review", skill.Name)
	assert.Equal(t, "2.0", skill.Version)
}

func TestGetSkill_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("skill not found"))
	}))
	defer srv.Close()

	acts := newTestActivities()
	acts.Endpoints.SkillsURL = srv.URL

	_, err := acts.GetSkill(context.Background(), "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "404")
}

// ---------------------------------------------------------------------------
// RunEvaluation
// ---------------------------------------------------------------------------

func TestRunEvaluation_Pass(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/evaluate", r.URL.Path)

		var req evalRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		assert.Equal(t, "agent-1", req.AgentID)
		assert.Equal(t, "code-review", req.Skill.Name)

		json.NewEncoder(w).Encode(EvaluationResult{
			Passed:  true,
			Score:   0.95,
			Details: "all checks passed",
		})
	}))
	defer srv.Close()

	acts := newTestActivities()
	acts.Endpoints.ProbeURL = srv.URL

	result, err := acts.RunEvaluation(context.Background(), "agent-1", SkillDefinition{Name: "code-review"})
	require.NoError(t, err)
	assert.True(t, result.Passed)
	assert.Equal(t, 0.95, result.Score)
}

func TestRunEvaluation_Fail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(EvaluationResult{
			Passed:  false,
			Score:   0.3,
			Details: "below threshold",
		})
	}))
	defer srv.Close()

	acts := newTestActivities()
	acts.Endpoints.ProbeURL = srv.URL

	result, err := acts.RunEvaluation(context.Background(), "agent-1", SkillDefinition{Name: "test"})
	require.NoError(t, err)
	assert.False(t, result.Passed)
}

// ---------------------------------------------------------------------------
// CheckBudget
// ---------------------------------------------------------------------------

func TestCheckBudget_Approved(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Contains(t, r.URL.Path, "/api/v1/budget/check")
		assert.Equal(t, "agent-1", r.URL.Query().Get("agent_id"))

		json.NewEncoder(w).Encode(BudgetCheckResult{
			Approved:       true,
			RemainingBudget: 1000.0,
		})
	}))
	defer srv.Close()

	acts := newTestActivities()
	acts.Endpoints.FinOpsURL = srv.URL

	result, err := acts.CheckBudget(context.Background(), "agent-1")
	require.NoError(t, err)
	assert.True(t, result.Approved)
	assert.Equal(t, 1000.0, result.RemainingBudget)
}

func TestCheckBudget_Denied(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(BudgetCheckResult{
			Approved: false,
			Reason:   "exceeded monthly limit",
		})
	}))
	defer srv.Close()

	acts := newTestActivities()
	acts.Endpoints.FinOpsURL = srv.URL

	result, err := acts.CheckBudget(context.Background(), "agent-1")
	require.NoError(t, err)
	assert.False(t, result.Approved)
	assert.Equal(t, "exceeded monthly limit", result.Reason)
}

// ---------------------------------------------------------------------------
// UpdateStatus
// ---------------------------------------------------------------------------

func TestUpdateStatus_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/agents/status", r.URL.Path)

		var req statusPatch
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		assert.Equal(t, "agent-1", req.AgentID)
		assert.Equal(t, "promoted", req.Status)

		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	acts := newTestActivities()
	acts.Endpoints.OperatorURL = srv.URL

	err := acts.UpdateStatus(context.Background(), "agent-1", "promoted")
	require.NoError(t, err)
}

func TestUpdateStatus_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("CRD update failed"))
	}))
	defer srv.Close()

	acts := newTestActivities()
	acts.Endpoints.OperatorURL = srv.URL

	err := acts.UpdateStatus(context.Background(), "agent-1", "promoted")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

// ---------------------------------------------------------------------------
// EndpointsFromEnv
// ---------------------------------------------------------------------------

func TestEndpointsFromEnv_Defaults(t *testing.T) {
	// Clear any env vars that might be set.
	for _, key := range []string{"WARD_URL", "ENGINE_URL", "MEMORY_URL", "SKILLS_URL", "PROBE_URL", "FINOPS_URL", "OPERATOR_URL"} {
		t.Setenv(key, "")
	}

	ep := EndpointsFromEnv()
	assert.Contains(t, ep.WardURL, "ward.arcana.svc.cluster.local")
	assert.Contains(t, ep.EngineURL, "engine.arcana.svc.cluster.local")
	assert.Contains(t, ep.MemoryURL, "oracle.arcana.svc.cluster.local")
	assert.Contains(t, ep.SkillsURL, "registry.arcana.svc.cluster.local")
	assert.Contains(t, ep.ProbeURL, "audit.arcana.svc.cluster.local")
	assert.Contains(t, ep.FinOpsURL, "finops.arcana.svc.cluster.local")
	assert.Contains(t, ep.OperatorURL, "operator.arcana.svc.cluster.local")
}

func TestEndpointsFromEnv_Override(t *testing.T) {
	t.Setenv("WARD_URL", "http://localhost:9000")
	t.Setenv("FINOPS_URL", "http://localhost:9001")

	ep := EndpointsFromEnv()
	assert.Equal(t, "http://localhost:9000", ep.WardURL)
	assert.Equal(t, "http://localhost:9001", ep.FinOpsURL)
	// Others should be defaults.
	assert.Contains(t, ep.EngineURL, "engine.arcana.svc.cluster.local")
}

// ---------------------------------------------------------------------------
// NewActivities
// ---------------------------------------------------------------------------

func TestNewActivities(t *testing.T) {
	acts := NewActivities()
	assert.NotNil(t, acts.HTTPClient)
	assert.Equal(t, 10*time.Second, acts.HTTPClient.Timeout)
	assert.NotEmpty(t, acts.Endpoints.WardURL)
}

// ---------------------------------------------------------------------------
// HTTP helper edge cases
// ---------------------------------------------------------------------------

func TestPostJSON_ContextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response — the test cancels the context before this returns.
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	acts := newTestActivities()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	err := acts.postJSON(ctx, srv.URL+"/api/v1/test", map[string]string{"key": "value"}, nil)
	assert.Error(t, err)
}
