package acp

import "time"

// ACPAgent describes a registered ACP agent.
type ACPAgent struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Capabilities []string          `json:"capabilities"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// Step is a single action-observation pair in an agent trajectory.
type Step struct {
	Action      string    `json:"action"`
	Observation string    `json:"observation"`
	Timestamp   time.Time `json:"timestamp"`
}

// Trajectory captures an agent's execution path and outcome.
type Trajectory struct {
	AgentID string `json:"agent_id"`
	Steps   []Step `json:"steps"`
	Outcome string `json:"outcome"`
}
