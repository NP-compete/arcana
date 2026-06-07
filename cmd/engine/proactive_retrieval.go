package main

import (
	"log"
	"strings"
)

type RetrievalPolicy struct {
	triggerPatterns []string
	confidenceMin  float64
}

func NewRetrievalPolicy() *RetrievalPolicy {
	return &RetrievalPolicy{
		triggerPatterns: []string{
			"latest", "current", "recent", "updated", "new policy",
			"how much", "what is the", "when did", "who is",
			"revenue", "budget", "deadline", "target", "goal",
			"changed", "modified", "announced",
		},
		confidenceMin: 0.3,
	}
}

func (rp *RetrievalPolicy) ShouldRetrieve(query string, agentConfidence float64) (bool, string) {
	lower := strings.ToLower(query)

	for _, pattern := range rp.triggerPatterns {
		if strings.Contains(lower, pattern) {
			return true, "query contains time-sensitive pattern: " + pattern
		}
	}

	if agentConfidence < rp.confidenceMin {
		return true, "agent confidence below threshold"
	}

	if len(strings.Fields(query)) > 20 {
		return true, "complex query may benefit from retrieval"
	}

	return false, ""
}

func (rp *RetrievalPolicy) RetrieveFromCodex(query, codexHost string) (string, error) {
	log.Printf("proactive retrieval: searching codex for '%s'", query[:min(50, len(query))])
	return "", nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
