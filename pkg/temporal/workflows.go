package temporal

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	// TaskQueue is the Temporal task queue name used by all Arcana workflows.
	TaskQueue = "arcana-agent-tasks"
)

// ---------------------------------------------------------------------------
// Workflow input/output types
// ---------------------------------------------------------------------------

// RunAgentInput is the input to RunAgentWorkflow.
type RunAgentInput struct {
	AgentID  string `json:"agent_id"`
	TaskType string `json:"task_type"`
	Input    string `json:"input"`
}

// EvaluateSkillInput is the input to EvaluateSkillWorkflow.
type EvaluateSkillInput struct {
	AgentID string `json:"agent_id"`
	Skill   string `json:"skill"`
}

// SkillDefinition holds skill metadata retrieved from the Skills service.
type SkillDefinition struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
}

// EvaluationResult holds the outcome of a skill evaluation.
type EvaluationResult struct {
	Passed  bool   `json:"passed"`
	Score   float64 `json:"score"`
	Details string `json:"details"`
}

// PromoteAgentInput is the input to PromoteAgentWorkflow.
type PromoteAgentInput struct {
	AgentID string `json:"agent_id"`
}

// BudgetCheckResult holds the FinOps budget check outcome.
type BudgetCheckResult struct {
	Approved       bool    `json:"approved"`
	RemainingBudget float64 `json:"remaining_budget"`
	Reason         string  `json:"reason"`
}

// ---------------------------------------------------------------------------
// Workflow: RunAgentWorkflow
// ---------------------------------------------------------------------------

// RunAgentWorkflow orchestrates an agent execution:
//  1. CheckGuardrails -- validate the input via the Ward service.
//  2. ExecuteTask     -- invoke the appropriate service for the task type.
//  3. StoreMemory     -- persist the result in the Memory service.
//
// Returns the final result string.
func RunAgentWorkflow(ctx workflow.Context, input RunAgentInput) (string, error) {
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts:    3,
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	var acts *Activities

	// Step 1: Check guardrails.
	var allowed bool
	if err := workflow.ExecuteActivity(ctx, acts.CheckGuardrails, input.Input).Get(ctx, &allowed); err != nil {
		return "", fmt.Errorf("guardrail check failed: %w", err)
	}
	if !allowed {
		return "", fmt.Errorf("input rejected by guardrails")
	}

	// Step 2: Execute the task.
	var result string
	if err := workflow.ExecuteActivity(ctx, acts.ExecuteTask, input.AgentID, input.TaskType, input.Input).Get(ctx, &result); err != nil {
		return "", fmt.Errorf("task execution failed: %w", err)
	}

	// Step 3: Store the result in memory.
	if err := workflow.ExecuteActivity(ctx, acts.StoreMemory, input.AgentID, result).Get(ctx, nil); err != nil {
		return "", fmt.Errorf("memory storage failed: %w", err)
	}

	return result, nil
}

// ---------------------------------------------------------------------------
// Workflow: EvaluateSkillWorkflow
// ---------------------------------------------------------------------------

// EvaluateSkillWorkflow evaluates a specific skill for an agent:
//  1. GetSkill       -- fetch the skill definition from the Skills service.
//  2. RunEvaluation  -- run an evaluation via the Probe service.
//
// Returns whether the evaluation passed.
func EvaluateSkillWorkflow(ctx workflow.Context, input EvaluateSkillInput) (bool, error) {
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 60 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts:    3,
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	var acts *Activities

	// Step 1: Get the skill definition.
	var skill SkillDefinition
	if err := workflow.ExecuteActivity(ctx, acts.GetSkill, input.Skill).Get(ctx, &skill); err != nil {
		return false, fmt.Errorf("get skill failed: %w", err)
	}

	// Step 2: Run evaluation against the Probe service.
	var evalResult EvaluationResult
	if err := workflow.ExecuteActivity(ctx, acts.RunEvaluation, input.AgentID, skill).Get(ctx, &evalResult); err != nil {
		return false, fmt.Errorf("evaluation failed: %w", err)
	}

	return evalResult.Passed, nil
}

// ---------------------------------------------------------------------------
// Workflow: PromoteAgentWorkflow
// ---------------------------------------------------------------------------

// PromoteAgentWorkflow promotes an agent to a higher environment:
//  1. RunEvaluation -- run the full evaluation suite.
//  2. CheckBudget   -- verify FinOps budget is available.
//  3. UpdateStatus   -- patch the Agent CRD status to promoted.
//
// Returns an error if any step fails or preconditions are not met.
func PromoteAgentWorkflow(ctx workflow.Context, input PromoteAgentInput) error {
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 2 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts:    3,
			InitialInterval:    2 * time.Second,
			BackoffCoefficient: 2.0,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	var acts *Activities

	// Step 1: Run the evaluation suite.
	var evalResult EvaluationResult
	evalInput := EvaluateSkillInput{AgentID: input.AgentID, Skill: "promotion-suite"}
	if err := workflow.ExecuteActivity(ctx, acts.RunEvaluation, evalInput.AgentID, SkillDefinition{
		Name: evalInput.Skill,
	}).Get(ctx, &evalResult); err != nil {
		return fmt.Errorf("promotion evaluation failed: %w", err)
	}
	if !evalResult.Passed {
		return fmt.Errorf("agent %s did not pass evaluation: %s", input.AgentID, evalResult.Details)
	}

	// Step 2: Check FinOps budget.
	var budget BudgetCheckResult
	if err := workflow.ExecuteActivity(ctx, acts.CheckBudget, input.AgentID).Get(ctx, &budget); err != nil {
		return fmt.Errorf("budget check failed: %w", err)
	}
	if !budget.Approved {
		return fmt.Errorf("budget not approved for agent %s: %s", input.AgentID, budget.Reason)
	}

	// Step 3: Update agent CRD status to promoted.
	if err := workflow.ExecuteActivity(ctx, acts.UpdateStatus, input.AgentID, "promoted").Get(ctx, nil); err != nil {
		return fmt.Errorf("status update failed: %w", err)
	}

	return nil
}
