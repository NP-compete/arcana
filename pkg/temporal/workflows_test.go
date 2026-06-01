package temporal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/testsuite"
)

func setupEnv(t *testing.T) *testsuite.TestWorkflowEnvironment {
	t.Helper()
	suite := &testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()
	var acts *Activities
	env.RegisterActivity(acts)
	return env
}

// ---------------------------------------------------------------------------
// RunAgentWorkflow tests
// ---------------------------------------------------------------------------

func TestRunAgentWorkflow_Success(t *testing.T) {
	env := setupEnv(t)

	env.OnActivity("CheckGuardrails", mock.Anything, "summarize this text").Return(true, nil)
	env.OnActivity("ExecuteTask", mock.Anything, "agent-1", "summarize", "summarize this text").Return("summary result", nil)
	env.OnActivity("StoreMemory", mock.Anything, "agent-1", "summary result").Return(nil)

	env.ExecuteWorkflow(RunAgentWorkflow, RunAgentInput{
		AgentID:  "agent-1",
		TaskType: "summarize",
		Input:    "summarize this text",
	})

	assert.True(t, env.IsWorkflowCompleted())
	assert.NoError(t, env.GetWorkflowError())

	var result string
	assert.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, "summary result", result)
}

func TestRunAgentWorkflow_GuardrailRejected(t *testing.T) {
	env := setupEnv(t)

	env.OnActivity("CheckGuardrails", mock.Anything, "bad input").Return(false, nil)

	env.ExecuteWorkflow(RunAgentWorkflow, RunAgentInput{
		AgentID:  "agent-1",
		TaskType: "test",
		Input:    "bad input",
	})

	assert.True(t, env.IsWorkflowCompleted())
	assert.Error(t, env.GetWorkflowError())
	assert.Contains(t, env.GetWorkflowError().Error(), "input rejected by guardrails")
}

func TestRunAgentWorkflow_ExecuteTaskFails(t *testing.T) {
	env := setupEnv(t)

	env.OnActivity("CheckGuardrails", mock.Anything, "input").Return(true, nil)
	env.OnActivity("ExecuteTask", mock.Anything, "agent-1", "fail-type", "input").Return("", assert.AnError)

	env.ExecuteWorkflow(RunAgentWorkflow, RunAgentInput{
		AgentID:  "agent-1",
		TaskType: "fail-type",
		Input:    "input",
	})

	assert.True(t, env.IsWorkflowCompleted())
	assert.Error(t, env.GetWorkflowError())
	assert.Contains(t, env.GetWorkflowError().Error(), "task execution failed")
}

func TestRunAgentWorkflow_StoreMemoryFails(t *testing.T) {
	env := setupEnv(t)

	env.OnActivity("CheckGuardrails", mock.Anything, "input").Return(true, nil)
	env.OnActivity("ExecuteTask", mock.Anything, "agent-1", "task", "input").Return("result", nil)
	env.OnActivity("StoreMemory", mock.Anything, "agent-1", "result").Return(assert.AnError)

	env.ExecuteWorkflow(RunAgentWorkflow, RunAgentInput{
		AgentID:  "agent-1",
		TaskType: "task",
		Input:    "input",
	})

	assert.True(t, env.IsWorkflowCompleted())
	assert.Error(t, env.GetWorkflowError())
	assert.Contains(t, env.GetWorkflowError().Error(), "memory storage failed")
}

// ---------------------------------------------------------------------------
// EvaluateSkillWorkflow tests
// ---------------------------------------------------------------------------

func TestEvaluateSkillWorkflow_Pass(t *testing.T) {
	env := setupEnv(t)

	skill := &SkillDefinition{
		Name:        "code-review",
		Description: "Reviews code changes",
		Version:     "1.0",
	}

	env.OnActivity("GetSkill", mock.Anything, "code-review").Return(skill, nil)
	env.OnActivity("RunEvaluation", mock.Anything, "agent-1", *skill).Return(
		&EvaluationResult{Passed: true, Score: 0.95, Details: "all checks passed"},
		nil,
	)

	env.ExecuteWorkflow(EvaluateSkillWorkflow, EvaluateSkillInput{
		AgentID: "agent-1",
		Skill:   "code-review",
	})

	assert.True(t, env.IsWorkflowCompleted())
	assert.NoError(t, env.GetWorkflowError())

	var passed bool
	assert.NoError(t, env.GetWorkflowResult(&passed))
	assert.True(t, passed)
}

func TestEvaluateSkillWorkflow_Fail(t *testing.T) {
	env := setupEnv(t)

	skill := &SkillDefinition{
		Name:        "code-review",
		Description: "Reviews code changes",
		Version:     "1.0",
	}

	env.OnActivity("GetSkill", mock.Anything, "code-review").Return(skill, nil)
	env.OnActivity("RunEvaluation", mock.Anything, "agent-1", *skill).Return(
		&EvaluationResult{Passed: false, Score: 0.3, Details: "accuracy below threshold"},
		nil,
	)

	env.ExecuteWorkflow(EvaluateSkillWorkflow, EvaluateSkillInput{
		AgentID: "agent-1",
		Skill:   "code-review",
	})

	assert.True(t, env.IsWorkflowCompleted())
	assert.NoError(t, env.GetWorkflowError())

	var passed bool
	assert.NoError(t, env.GetWorkflowResult(&passed))
	assert.False(t, passed)
}

func TestEvaluateSkillWorkflow_GetSkillFails(t *testing.T) {
	env := setupEnv(t)

	env.OnActivity("GetSkill", mock.Anything, "nonexistent").Return(nil, assert.AnError)

	env.ExecuteWorkflow(EvaluateSkillWorkflow, EvaluateSkillInput{
		AgentID: "agent-1",
		Skill:   "nonexistent",
	})

	assert.True(t, env.IsWorkflowCompleted())
	assert.Error(t, env.GetWorkflowError())
	assert.Contains(t, env.GetWorkflowError().Error(), "get skill failed")
}

// ---------------------------------------------------------------------------
// PromoteAgentWorkflow tests
// ---------------------------------------------------------------------------

func TestPromoteAgentWorkflow_Success(t *testing.T) {
	env := setupEnv(t)

	env.OnActivity("RunEvaluation", mock.Anything, "agent-1", mock.Anything).Return(
		&EvaluationResult{Passed: true, Score: 0.9, Details: "passed"},
		nil,
	)
	env.OnActivity("CheckBudget", mock.Anything, "agent-1").Return(
		&BudgetCheckResult{Approved: true, RemainingBudget: 500.0},
		nil,
	)
	env.OnActivity("UpdateStatus", mock.Anything, "agent-1", "promoted").Return(nil)

	env.ExecuteWorkflow(PromoteAgentWorkflow, PromoteAgentInput{
		AgentID: "agent-1",
	})

	assert.True(t, env.IsWorkflowCompleted())
	assert.NoError(t, env.GetWorkflowError())
}

func TestPromoteAgentWorkflow_EvalFails(t *testing.T) {
	env := setupEnv(t)

	env.OnActivity("RunEvaluation", mock.Anything, "agent-1", mock.Anything).Return(
		&EvaluationResult{Passed: false, Score: 0.2, Details: "failed checks"},
		nil,
	)

	env.ExecuteWorkflow(PromoteAgentWorkflow, PromoteAgentInput{
		AgentID: "agent-1",
	})

	assert.True(t, env.IsWorkflowCompleted())
	assert.Error(t, env.GetWorkflowError())
	assert.Contains(t, env.GetWorkflowError().Error(), "did not pass evaluation")
}

func TestPromoteAgentWorkflow_BudgetDenied(t *testing.T) {
	env := setupEnv(t)

	env.OnActivity("RunEvaluation", mock.Anything, "agent-1", mock.Anything).Return(
		&EvaluationResult{Passed: true, Score: 0.9, Details: "passed"},
		nil,
	)
	env.OnActivity("CheckBudget", mock.Anything, "agent-1").Return(
		&BudgetCheckResult{Approved: false, RemainingBudget: 0, Reason: "budget exhausted"},
		nil,
	)

	env.ExecuteWorkflow(PromoteAgentWorkflow, PromoteAgentInput{
		AgentID: "agent-1",
	})

	assert.True(t, env.IsWorkflowCompleted())
	assert.Error(t, env.GetWorkflowError())
	assert.Contains(t, env.GetWorkflowError().Error(), "budget not approved")
}

func TestPromoteAgentWorkflow_UpdateStatusFails(t *testing.T) {
	env := setupEnv(t)

	env.OnActivity("RunEvaluation", mock.Anything, "agent-1", mock.Anything).Return(
		&EvaluationResult{Passed: true, Score: 0.9, Details: "passed"},
		nil,
	)
	env.OnActivity("CheckBudget", mock.Anything, "agent-1").Return(
		&BudgetCheckResult{Approved: true, RemainingBudget: 500.0},
		nil,
	)
	env.OnActivity("UpdateStatus", mock.Anything, "agent-1", "promoted").Return(assert.AnError)

	env.ExecuteWorkflow(PromoteAgentWorkflow, PromoteAgentInput{
		AgentID: "agent-1",
	})

	assert.True(t, env.IsWorkflowCompleted())
	assert.Error(t, env.GetWorkflowError())
	assert.Contains(t, env.GetWorkflowError().Error(), "status update failed")
}
