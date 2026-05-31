package main

import (
	"context"
	"fmt"
	"math/rand"
)

// Activities holds shared dependencies needed by Temporal activity functions.
// Methods on this struct are registered with the Temporal worker.
// The workflow references them by method name (e.g. "PlanActivity").
type Activities struct {
	store *TaskStore
	react *ReActEngine
}

// PlanActivity determines the next action for a given step.
func (a *Activities) PlanActivity(_ context.Context, task TaskRequest, step int) (*PlanResult, error) {
	return &PlanResult{
		Action: fmt.Sprintf("Step %d: analyze input and determine next action for agent %s", step, task.Agent),
		Tool:   fmt.Sprintf("invoke_tool_%d", step%3),
		Args:   task.Input,
	}, nil
}

// ActActivity executes the planned action (tool invocation).
func (a *Activities) ActActivity(_ context.Context, plan PlanResult) (*ActionResult, error) {
	return &ActionResult{
		Output: fmt.Sprintf("Tool %s returned structured data", plan.Tool),
	}, nil
}

// ObserveActivity synthesises tool output into observations.
func (a *Activities) ObserveActivity(_ context.Context, action ActionResult) (*ObservationResult, error) {
	return &ObservationResult{
		Summary: fmt.Sprintf("Observed: %s", action.Output),
		Facts:   []string{"fact_1", "fact_2"},
	}, nil
}

// EvaluateActivity decides whether the goal is satisfied or more steps are needed.
func (a *Activities) EvaluateActivity(_ context.Context, observation ObservationResult, task TaskRequest) (*EvalResult, error) {
	tokens := 50 + rand.Intn(150)
	if len(observation.Facts) >= 2 {
		return &EvalResult{
			Complete: true,
			Result: TaskResult{
				Output:     fmt.Sprintf("Agent %s completed goal: %s", task.Agent, task.Goal),
				TokensUsed: tokens,
				StepsUsed:  1,
			},
			Reason: "goal_satisfied",
		}, nil
	}

	return &EvalResult{
		Complete: false,
		Reason:   "needs_more_data",
	}, nil
}
