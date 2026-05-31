package main

// TaskRequest is the input to the AgentTaskWorkflow.
type TaskRequest struct {
	ID    string                 `json:"id"`
	Agent string                 `json:"agent"`
	Goal  string                 `json:"goal"`
	Input map[string]interface{} `json:"input"`
}

// PlanResult is returned by PlanActivity.
type PlanResult struct {
	Action string                 `json:"action"`
	Tool   string                 `json:"tool"`
	Args   map[string]interface{} `json:"args"`
}

// ActionResult is returned by ActActivity.
type ActionResult struct {
	Output string `json:"output"`
	Error  string `json:"error"`
}

// ObservationResult is returned by ObserveActivity.
type ObservationResult struct {
	Summary string   `json:"summary"`
	Facts   []string `json:"facts"`
}

// EvalResult is returned by EvaluateActivity.
type EvalResult struct {
	Complete bool       `json:"complete"`
	Result   TaskResult `json:"result"`
	Reason   string     `json:"reason"`
}

// TaskResult carries the final output of a workflow execution.
type TaskResult struct {
	Output     string `json:"output"`
	TokensUsed int    `json:"tokens_used"`
	StepsUsed  int    `json:"steps_used"`
}
