package main

import (
	"context"
	"fmt"
	"log"
	"strings"
)

type Activities struct {
	store    *TaskStore
	react    *ReActEngine
	llm      *LLMClient
	services *ServiceClients
}

func (a *Activities) PlanActivity(_ context.Context, task TaskRequest, step int) (*PlanResult, error) {
	wardResult, _ := a.services.CheckWard(task.Agent, task.Goal, "inbound")
	if wardResult != nil && wardResult.Blocked {
		return &PlanResult{
			Action: fmt.Sprintf("BLOCKED by guardrails: %s", wardResult.Reason),
			Tool:   "none",
			Args:   nil,
		}, nil
	}

	prompt := fmt.Sprintf(
		"You are agent '%s'. Goal: %s\nStep %d of the ReAct loop.\n"+
			"Decide the next action. Respond with:\n"+
			"ACTION: <description of what to do>\n"+
			"TOOL: <tool_name or 'none'>\n"+
			"Keep it concise.",
		task.Agent, task.Goal, step+1,
	)

	response, tokens, err := a.llm.Complete(
		"You are an AI agent executing tasks step by step using the ReAct framework.",
		prompt,
		500,
	)
	if err != nil {
		log.Printf("plan: LLM call failed, using fallback: %v", err)
		return &PlanResult{
			Action: fmt.Sprintf("Step %d: process input for agent %s", step+1, task.Agent),
			Tool:   "none",
			Args:   task.Input,
		}, nil
	}

	action, tool := parsePlanResponse(response)

	a.store.Update(task.ID, func(t *AgentTask) {
		t.TokensUsed += tokens
		t.Cost += float64(tokens) * 0.00002
		t.CurrentStep = step + 1
	})

	return &PlanResult{
		Action: action,
		Tool:   tool,
		Args:   task.Input,
	}, nil
}

func (a *Activities) ActActivity(_ context.Context, plan PlanResult) (*ActionResult, error) {
	if plan.Tool == "none" || plan.Tool == "" {
		return &ActionResult{
			Output: plan.Action,
		}, nil
	}

	output, err := a.services.InvokeSkill(plan.Tool, plan.Args)
	if err != nil {
		log.Printf("act: skill invocation failed: %v", err)
		return &ActionResult{
			Output: plan.Action,
			Error:  err.Error(),
		}, nil
	}

	return &ActionResult{
		Output: output,
	}, nil
}

func (a *Activities) ObserveActivity(_ context.Context, action ActionResult) (*ObservationResult, error) {
	if action.Error != "" {
		return &ObservationResult{
			Summary: fmt.Sprintf("Action failed: %s", action.Error),
			Facts:   []string{"action_failed"},
		}, nil
	}

	response, tokens, err := a.llm.Complete(
		"Extract key facts from this tool output. List each fact on a new line starting with '- '.",
		fmt.Sprintf("Tool output:\n%s\n\nExtract the key facts.", action.Output),
		300,
	)
	if err != nil {
		return &ObservationResult{
			Summary: action.Output,
			Facts:   []string{"output_received"},
		}, nil
	}

	facts := []string{}
	for _, line := range strings.Split(response, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "- ") {
			facts = append(facts, strings.TrimPrefix(line, "- "))
		}
	}
	if len(facts) == 0 {
		facts = []string{response}
	}

	_ = tokens
	return &ObservationResult{
		Summary: response,
		Facts:   facts,
	}, nil
}

func (a *Activities) EvaluateActivity(_ context.Context, observation ObservationResult, task TaskRequest) (*EvalResult, error) {
	prompt := fmt.Sprintf(
		"Agent goal: %s\nObservations so far:\n%s\n\n"+
			"Is the goal complete? Respond with exactly 'COMPLETE' or 'CONTINUE' on the first line, then a brief reason.",
		task.Goal, observation.Summary,
	)

	response, tokens, err := a.llm.Complete(
		"You evaluate whether an agent has completed its goal.",
		prompt,
		200,
	)

	complete := false
	reason := "needs_more_data"
	if err == nil {
		firstLine := strings.Split(strings.TrimSpace(response), "\n")[0]
		if strings.Contains(strings.ToUpper(firstLine), "COMPLETE") {
			complete = true
			reason = "goal_satisfied"
		}
	}

	a.store.Update(task.ID, func(t *AgentTask) {
		t.TokensUsed += tokens
		t.Cost += float64(tokens) * 0.00002
	})

	wardResult, _ := a.services.CheckWard(task.Agent, observation.Summary, "outbound")
	if wardResult != nil && wardResult.Blocked {
		return &EvalResult{
			Complete: true,
			Result: TaskResult{
				Output:     fmt.Sprintf("Output blocked by guardrails: %s", wardResult.Reason),
				TokensUsed: tokens,
				StepsUsed:  1,
			},
			Reason: "guardrail_blocked",
		}, nil
	}

	a.services.AppendAudit(task.Agent, "evaluate", task.Goal, observation.Summary, reason)

	if complete {
		return &EvalResult{
			Complete: true,
			Result: TaskResult{
				Output:     observation.Summary,
				TokensUsed: tokens,
				StepsUsed:  1,
			},
			Reason: reason,
		}, nil
	}

	return &EvalResult{
		Complete: false,
		Reason:   reason,
	}, nil
}

func parsePlanResponse(response string) (action, tool string) {
	action = response
	tool = "none"
	for _, line := range strings.Split(response, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToUpper(line), "ACTION:") {
			action = strings.TrimSpace(strings.TrimPrefix(line, "ACTION:"))
			action = strings.TrimSpace(strings.TrimPrefix(action, "action:"))
		}
		if strings.HasPrefix(strings.ToUpper(line), "TOOL:") {
			tool = strings.TrimSpace(strings.TrimPrefix(line, "TOOL:"))
			tool = strings.TrimSpace(strings.TrimPrefix(tool, "tool:"))
		}
	}
	return action, tool
}
