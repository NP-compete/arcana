package temporal

import "context"

// AgentWorkflows defines Temporal workflow function signatures for agent orchestration.
type AgentWorkflows interface {
	RunAgent(ctx context.Context, agentID string, input string) (string, error)
	EvaluateSkill(ctx context.Context, agentID string, skill string) (bool, error)
	PromoteAgent(ctx context.Context, agentID string) error
}

// AgentActivities defines Temporal activity function signatures for agent execution.
type AgentActivities interface {
	ExecuteSkill(ctx context.Context, agentID string, skill string, input string) (string, error)
	CallTool(ctx context.Context, toolName string, args map[string]any) (string, error)
	CheckGuardrails(ctx context.Context, input string) (bool, error)
	StoreResult(ctx context.Context, agentID string, result string) error
}
