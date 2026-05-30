package main

import (
	"fmt"
	"log"
	"math/rand"
	"time"
)

type ReActEngine struct {
	store *TaskStore
}

func NewReActEngine(store *TaskStore) *ReActEngine {
	return &ReActEngine{store: store}
}

func (e *ReActEngine) Run(taskID string, steps int) {
	if steps <= 0 {
		steps = 3
	}
	if steps > 20 {
		steps = 20
	}

	e.store.Update(taskID, func(t *AgentTask) {
		t.Status = TaskStatusRunning
	})

	stepsLog := make([]map[string]interface{}, 0, steps)
	totalTokens := 0

	for i := 1; i <= steps; i++ {
		if cancelled := e.checkCancelled(taskID); cancelled {
			return
		}

		phase := reactPhase(i)
		tokens := 50 + rand.Intn(150)
		totalTokens += tokens

		stepResult := map[string]interface{}{
			"step":      i,
			"phase":     phase,
			"plan":      fmt.Sprintf("Step %d: analyze input and determine next action for agent %s", i, e.getAgent(taskID)),
			"action":    fmt.Sprintf("invoke_tool_%d", i%3),
			"observation": fmt.Sprintf("Tool returned structured data for step %d", i),
			"evaluation":  evaluateStep(i, steps),
			"tokens":    tokens,
		}
		stepsLog = append(stepsLog, stepResult)

		e.store.Update(taskID, func(t *AgentTask) {
			t.TokensUsed = totalTokens
			t.Cost = float64(totalTokens) * 0.00002
		})

		time.Sleep(100 * time.Millisecond)
	}

	e.store.Update(taskID, func(t *AgentTask) {
		t.Status = TaskStatusCompleted
		t.TokensUsed = totalTokens
		t.Cost = float64(totalTokens) * 0.00002
		t.Result = map[string]interface{}{
			"steps_executed": steps,
			"workflow":       "react",
			"trace":          stepsLog,
			"summary":        fmt.Sprintf("Agent %s completed %d ReAct cycles successfully", t.Agent, steps),
		}
	})

	log.Printf("task %s completed with %d steps, %d tokens", taskID, steps, totalTokens)
}

func (e *ReActEngine) checkCancelled(taskID string) bool {
	task, ok := e.store.Get(taskID)
	if !ok {
		return true
	}
	return task.Status == TaskStatusCancelled
}

func (e *ReActEngine) getAgent(taskID string) string {
	task, ok := e.store.Get(taskID)
	if !ok {
		return "unknown"
	}
	return task.Agent
}

func reactPhase(step int) string {
	phases := []string{"plan", "act", "observe", "evaluate"}
	return phases[(step-1)%len(phases)]
}

func evaluateStep(step, total int) string {
	if step == total {
		return "goal_satisfied"
	}
	if step%4 == 0 {
		return "continue"
	}
	return "proceed"
}
