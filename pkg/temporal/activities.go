package temporal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// ServiceEndpoints holds the base URLs for the platform services that
// activities call. All values come from environment variables with
// cluster-internal defaults.
type ServiceEndpoints struct {
	WardURL    string // Ward guardrail service
	EngineURL  string // Engine task execution service
	MemoryURL  string // Memory storage service
	SkillsURL  string // Skills registry service
	ProbeURL   string // Probe evaluation service
	FinOpsURL  string // FinOps budget service
	OperatorURL string // Operator CRD management service
}

// EndpointsFromEnv reads service endpoints from environment variables,
// falling back to cluster-internal DNS defaults.
func EndpointsFromEnv() ServiceEndpoints {
	return ServiceEndpoints{
		WardURL:     envOrDefault("WARD_URL", "http://ward.arcana.svc.cluster.local:8080"),
		EngineURL:   envOrDefault("ENGINE_URL", "http://engine.arcana.svc.cluster.local:8081"),
		MemoryURL:   envOrDefault("MEMORY_URL", "http://oracle.arcana.svc.cluster.local:8080"),
		SkillsURL:   envOrDefault("SKILLS_URL", "http://registry.arcana.svc.cluster.local:8080"),
		ProbeURL:    envOrDefault("PROBE_URL", "http://audit.arcana.svc.cluster.local:8080"),
		FinOpsURL:   envOrDefault("FINOPS_URL", "http://finops.arcana.svc.cluster.local:8080"),
		OperatorURL: envOrDefault("OPERATOR_URL", "http://operator.arcana.svc.cluster.local:8080"),
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// Activities holds shared dependencies for all Temporal activity functions.
// Methods on this struct are registered with the Temporal worker; the workflow
// references them by method pointer (e.g. acts.CheckGuardrails).
type Activities struct {
	Endpoints  ServiceEndpoints
	HTTPClient *http.Client
}

// NewActivities creates an Activities instance with production defaults.
func NewActivities() *Activities {
	return &Activities{
		Endpoints: EndpointsFromEnv(),
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// ---------------------------------------------------------------------------
// Activity: CheckGuardrails
// ---------------------------------------------------------------------------

// guardrailRequest is the payload sent to the Ward service.
type guardrailRequest struct {
	Input string `json:"input"`
}

// guardrailResponse is the response from the Ward service.
type guardrailResponse struct {
	Allowed bool   `json:"allowed"`
	Reason  string `json:"reason,omitempty"`
}

// CheckGuardrails validates input against the Ward guardrail service.
// Returns true if the input is allowed, false otherwise.
func (a *Activities) CheckGuardrails(ctx context.Context, input string) (bool, error) {
	payload := guardrailRequest{Input: input}

	var resp guardrailResponse
	if err := a.postJSON(ctx, a.Endpoints.WardURL+"/api/v1/evaluate", payload, &resp); err != nil {
		return false, fmt.Errorf("ward service call failed: %w", err)
	}

	return resp.Allowed, nil
}

// ---------------------------------------------------------------------------
// Activity: ExecuteTask
// ---------------------------------------------------------------------------

// executeTaskRequest is the payload sent to the engine/execution service.
type executeTaskRequest struct {
	AgentID  string `json:"agent_id"`
	TaskType string `json:"task_type"`
	Input    string `json:"input"`
}

// executeTaskResponse is the response from the task execution service.
type executeTaskResponse struct {
	Result string `json:"result"`
	Error  string `json:"error,omitempty"`
}

// ExecuteTask dispatches a task to the appropriate service based on task type.
func (a *Activities) ExecuteTask(ctx context.Context, agentID, taskType, input string) (string, error) {
	payload := executeTaskRequest{
		AgentID:  agentID,
		TaskType: taskType,
		Input:    input,
	}

	var resp executeTaskResponse
	if err := a.postJSON(ctx, a.Endpoints.EngineURL+"/api/v1/execute", payload, &resp); err != nil {
		return "", fmt.Errorf("task execution service call failed: %w", err)
	}

	if resp.Error != "" {
		return "", fmt.Errorf("task execution returned error: %s", resp.Error)
	}

	return resp.Result, nil
}

// ---------------------------------------------------------------------------
// Activity: StoreMemory
// ---------------------------------------------------------------------------

// storeMemoryRequest is the payload sent to the Memory service.
type storeMemoryRequest struct {
	AgentID string `json:"agent_id"`
	Content string `json:"content"`
}

// StoreMemory persists a result in the Memory (Oracle) service.
func (a *Activities) StoreMemory(ctx context.Context, agentID, result string) error {
	payload := storeMemoryRequest{
		AgentID: agentID,
		Content: result,
	}

	if err := a.postJSON(ctx, a.Endpoints.MemoryURL+"/api/v1/memories", payload, nil); err != nil {
		return fmt.Errorf("memory service call failed: %w", err)
	}

	return nil
}

// ---------------------------------------------------------------------------
// Activity: GetSkill
// ---------------------------------------------------------------------------

// GetSkill fetches a skill definition from the Skills (Registry) service.
func (a *Activities) GetSkill(ctx context.Context, skillName string) (*SkillDefinition, error) {
	url := fmt.Sprintf("%s/api/v1/skills/%s", a.Endpoints.SkillsURL, skillName)

	var skill SkillDefinition
	if err := a.getJSON(ctx, url, &skill); err != nil {
		return nil, fmt.Errorf("skills service call failed: %w", err)
	}

	return &skill, nil
}

// ---------------------------------------------------------------------------
// Activity: RunEvaluation
// ---------------------------------------------------------------------------

// evalRequest is the payload sent to the Probe evaluation service.
type evalRequest struct {
	AgentID string          `json:"agent_id"`
	Skill   SkillDefinition `json:"skill"`
}

// RunEvaluation runs an evaluation via the Probe (Audit) service.
func (a *Activities) RunEvaluation(ctx context.Context, agentID string, skill SkillDefinition) (*EvaluationResult, error) {
	payload := evalRequest{
		AgentID: agentID,
		Skill:   skill,
	}

	var result EvaluationResult
	if err := a.postJSON(ctx, a.Endpoints.ProbeURL+"/api/v1/evaluate", payload, &result); err != nil {
		return nil, fmt.Errorf("probe service call failed: %w", err)
	}

	return &result, nil
}

// ---------------------------------------------------------------------------
// Activity: CheckBudget
// ---------------------------------------------------------------------------

// CheckBudget verifies FinOps budget availability for a given agent.
func (a *Activities) CheckBudget(ctx context.Context, agentID string) (*BudgetCheckResult, error) {
	url := fmt.Sprintf("%s/api/v1/budget/check?agent_id=%s", a.Endpoints.FinOpsURL, agentID)

	var result BudgetCheckResult
	if err := a.getJSON(ctx, url, &result); err != nil {
		return nil, fmt.Errorf("finops service call failed: %w", err)
	}

	return &result, nil
}

// ---------------------------------------------------------------------------
// Activity: UpdateStatus
// ---------------------------------------------------------------------------

// statusPatch is the payload sent to the operator to update an agent CRD.
type statusPatch struct {
	AgentID string `json:"agent_id"`
	Status  string `json:"status"`
}

// UpdateStatus patches the Agent CRD status via the Operator service.
func (a *Activities) UpdateStatus(ctx context.Context, agentID, status string) error {
	payload := statusPatch{
		AgentID: agentID,
		Status:  status,
	}

	if err := a.postJSON(ctx, a.Endpoints.OperatorURL+"/api/v1/agents/status", payload, nil); err != nil {
		return fmt.Errorf("operator service call failed: %w", err)
	}

	return nil
}

// ---------------------------------------------------------------------------
// HTTP helpers
// ---------------------------------------------------------------------------

// postJSON encodes payload as JSON, POSTs it to url, and decodes the response
// into out (if out is non-nil). The request inherits the context's deadline.
func (a *Activities) postJSON(ctx context.Context, url string, payload interface{}, out interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("http post %s: %w", url, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("http post %s returned status %d: %s", url, resp.StatusCode, string(respBody))
	}

	if out != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("decode response from %s: %w", url, err)
		}
	}

	return nil
}

// getJSON sends a GET request to url and decodes the response into out.
// The request inherits the context's deadline.
func (a *Activities) getJSON(ctx context.Context, url string, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := a.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("http get %s: %w", url, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("http get %s returned status %d: %s", url, resp.StatusCode, string(respBody))
	}

	if out != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("decode response from %s: %w", url, err)
		}
	}

	return nil
}
