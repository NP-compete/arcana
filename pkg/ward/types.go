package ward

// Violation describes a single guardrail rule violation.
type Violation struct {
	Rule     string `json:"rule"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

// PolicyVerdict is the result of a guardrail evaluation.
type PolicyVerdict struct {
	Allowed    bool        `json:"allowed"`
	Reason     string      `json:"reason,omitempty"`
	Violations []Violation `json:"violations,omitempty"`
}

// GuardrailPipeline evaluates input against guardrail policies.
type GuardrailPipeline interface {
	Evaluate(input string) (PolicyVerdict, error)
}
