package main

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// AgentTaskWorkflow runs the ReAct loop as a durable Temporal workflow.
// Each step (plan, act, observe, evaluate) is an activity, giving crash
// recovery between steps.
func AgentTaskWorkflow(ctx workflow.Context, task TaskRequest) (*TaskResult, error) {
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts:    3,
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	var result TaskResult
	maxSteps := 10

	// Activities are registered as struct methods on *Activities.
	// Temporal resolves them by name: "PlanActivity", "ActActivity", etc.
	var acts *Activities

	for step := 0; step < maxSteps; step++ {
		// Plan
		var plan PlanResult
		err := workflow.ExecuteActivity(ctx, acts.PlanActivity, task, step).Get(ctx, &plan)
		if err != nil {
			return nil, err
		}

		// Act
		var action ActionResult
		err = workflow.ExecuteActivity(ctx, acts.ActActivity, plan).Get(ctx, &action)
		if err != nil {
			return nil, err
		}

		// Observe
		var observation ObservationResult
		err = workflow.ExecuteActivity(ctx, acts.ObserveActivity, action).Get(ctx, &observation)
		if err != nil {
			return nil, err
		}

		// Evaluate
		var eval EvalResult
		err = workflow.ExecuteActivity(ctx, acts.EvaluateActivity, observation, task).Get(ctx, &eval)
		if err != nil {
			return nil, err
		}

		if eval.Complete {
			result = eval.Result
			break
		}
	}

	return &result, nil
}
