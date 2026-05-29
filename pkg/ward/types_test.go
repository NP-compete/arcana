package ward

import "testing"

func TestPolicyVerdictAllowed(t *testing.T) {
	verdict := PolicyVerdict{
		Allowed: true,
		Reason:  "input passed all guardrails",
	}

	if !verdict.Allowed {
		t.Error("expected Allowed to be true")
	}
	if verdict.Reason != "input passed all guardrails" {
		t.Errorf("reason: got %q, want %q", verdict.Reason, "input passed all guardrails")
	}
}
