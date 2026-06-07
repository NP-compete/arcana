package main

import (
	"context"
	"testing"
)

func TestParsePlanResponse(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantAction string
		wantTool   string
	}{
		{
			name:       "standard format",
			input:      "ACTION: search the web for recent papers\nTOOL: web_search",
			wantAction: "search the web for recent papers",
			wantTool:   "web_search",
		},
		{
			name:       "no tool",
			input:      "ACTION: think about the problem\nTOOL: none",
			wantAction: "think about the problem",
			wantTool:   "none",
		},
		{
			name:       "plain text fallback",
			input:      "I should search for more information",
			wantAction: "I should search for more information",
			wantTool:   "none",
		},
		{
			name:       "empty",
			input:      "",
			wantAction: "",
			wantTool:   "none",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action, tool := parsePlanResponse(tt.input)
			if action != tt.wantAction {
				t.Errorf("action = %q, want %q", action, tt.wantAction)
			}
			if tool != tt.wantTool {
				t.Errorf("tool = %q, want %q", tool, tt.wantTool)
			}
		})
	}
}

func TestPlanActivity_WardCheckFallback(t *testing.T) {
	svc := NewServiceClients()
	wardResult, _ := svc.CheckWard("test-agent", "test input", "inbound")
	if wardResult == nil {
		t.Fatal("expected non-nil ward result on fallback")
	}
	if wardResult.Blocked {
		t.Error("ward should default to allow when unreachable")
	}
	if wardResult.Verdict != "ALLOW" {
		t.Errorf("expected ALLOW verdict, got %s", wardResult.Verdict)
	}
}

func TestContextDAG_Basic(t *testing.T) {
	dag := NewContextDAG(1000)

	dag.AddNode("n1", "hello world this is a test", "")
	dag.AddNode("n2", "second node with more content", "n1")

	active := dag.ActiveContext()
	if len(active) != 2 {
		t.Errorf("expected 2 active nodes, got %d", len(active))
	}

	if dag.BudgetRemaining() >= 1000 {
		t.Error("budget should have decreased")
	}

	dag.Skip("n1")
	active = dag.ActiveContext()
	if len(active) != 1 {
		t.Errorf("expected 1 active node after skip, got %d", len(active))
	}

	stats := dag.Stats()
	if stats["skipped_nodes"].(int) != 1 {
		t.Errorf("expected 1 skipped node, got %v", stats["skipped_nodes"])
	}
}

func TestContextDAG_Rollback(t *testing.T) {
	dag := NewContextDAG(10000)

	dag.AddNode("n1", "first", "")
	dag.AddNode("n2", "second", "n1")
	dag.AddNode("n3", "third", "n2")

	dag.Rollback("n1")

	active := dag.ActiveContext()
	if len(active) != 1 {
		t.Errorf("expected 1 active node after rollback, got %d", len(active))
	}
}

func TestContextDAG_Fold(t *testing.T) {
	dag := NewContextDAG(10000)

	dag.AddNode("n1", "first long content", "")
	dag.AddNode("n2", "second long content", "n1")
	dag.AddNode("n3", "third long content", "n2")

	foldedID := dag.Fold([]string{"n1", "n2"}, "summary of n1 and n2")

	if foldedID == "" {
		t.Error("expected folded ID")
	}
}

func TestProactiveRetrieval(t *testing.T) {
	rp := NewRetrievalPolicy()

	tests := []struct {
		query  string
		expect bool
	}{
		{"What is the latest revenue target?", true},
		{"How do binary trees work?", false},
		{"Who is the current CEO?", true},
		{"Explain recursion", false},
		{"What changed in the policy recently?", true},
	}

	for _, tt := range tests {
		should, _ := rp.ShouldRetrieve(tt.query, 0.5)
		if should != tt.expect {
			t.Errorf("ShouldRetrieve(%q) = %v, want %v", tt.query, should, tt.expect)
		}
	}
}

func TestMCTSPlanner_SelectsBestAction(t *testing.T) {
	planner := NewMCTSPlanner()

	actions := []string{"search_web", "read_file", "ask_user", "generate_code"}
	result := planner.Plan("find information about AI agents", actions)

	if result == "" {
		t.Error("expected a selected action")
	}

	found := false
	for _, a := range actions {
		if a == result {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("result %q not in candidate actions", result)
	}
}

func TestMCTSPlanner_EmptyActions(t *testing.T) {
	planner := NewMCTSPlanner()
	result := planner.Plan("goal", []string{})
	if result != "" {
		t.Errorf("expected empty string for no actions, got %q", result)
	}
}

func TestMCTSPlanner_ShouldUseMCTS(t *testing.T) {
	planner := NewMCTSPlanner()
	if planner.ShouldUseMCTS(0.5) {
		t.Error("should not use MCTS for low complexity")
	}
	if !planner.ShouldUseMCTS(0.8) {
		t.Error("should use MCTS for high complexity")
	}
}
