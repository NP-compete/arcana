package main

import (
	"log"
	"strings"
	"time"
)

type ContextNode struct {
	ID          string
	Content     string
	Summary     string
	ParentID    string
	Depth       int
	TokenCount  int
	Compressed  bool
	Folded      bool
	Skipped     bool
	CreatedAt   time.Time
}

type ContextDAG struct {
	nodes          map[string]*ContextNode
	activeChain    []string
	budgetTotal    int
	budgetUsed     int
	compressThresh int
	nodeCompThresh int
}

func NewContextDAG(budgetTokens int) *ContextDAG {
	return &ContextDAG{
		nodes:          make(map[string]*ContextNode),
		activeChain:    make([]string, 0),
		budgetTotal:    budgetTokens,
		compressThresh: 180000,
		nodeCompThresh: 15000,
	}
}

func (dag *ContextDAG) AddNode(id, content, parentID string) {
	tokenCount := len(strings.Fields(content))
	node := &ContextNode{
		ID:         id,
		Content:    content,
		ParentID:   parentID,
		TokenCount: tokenCount,
		CreatedAt:  time.Now(),
	}
	dag.nodes[id] = node
	dag.activeChain = append(dag.activeChain, id)
	dag.budgetUsed += tokenCount
}

func (dag *ContextDAG) Compress(nodeID string) bool {
	node, ok := dag.nodes[nodeID]
	if !ok || node.Compressed {
		return false
	}

	originalLen := node.TokenCount
	words := strings.Fields(node.Content)
	if len(words) > 50 {
		node.Summary = strings.Join(words[:50], " ") + "..."
		node.Content = node.Summary
		node.TokenCount = 50
		node.Compressed = true
		dag.budgetUsed -= (originalLen - 50)
		log.Printf("context: compressed node %s (%d→50 tokens)", nodeID, originalLen)
		return true
	}
	return false
}

func (dag *ContextDAG) Fold(nodeIDs []string, summaryContent string) string {
	foldedID := "fold-" + nodeIDs[0]
	tokensSaved := 0

	for _, id := range nodeIDs {
		if node, ok := dag.nodes[id]; ok {
			node.Folded = true
			tokensSaved += node.TokenCount
		}
	}

	dag.AddNode(foldedID, summaryContent, "")
	log.Printf("context: folded %d nodes into %s (saved %d tokens)", len(nodeIDs), foldedID, tokensSaved)
	return foldedID
}

func (dag *ContextDAG) Skip(nodeID string) {
	if node, ok := dag.nodes[nodeID]; ok {
		node.Skipped = true
		dag.budgetUsed -= node.TokenCount
	}
}

func (dag *ContextDAG) Rollback(checkpointID string) {
	idx := -1
	for i, id := range dag.activeChain {
		if id == checkpointID {
			idx = i
			break
		}
	}
	if idx < 0 {
		return
	}

	for _, id := range dag.activeChain[idx+1:] {
		if node, ok := dag.nodes[id]; ok {
			dag.budgetUsed -= node.TokenCount
		}
	}
	dag.activeChain = dag.activeChain[:idx+1]
	log.Printf("context: rolled back to %s", checkpointID)
}

func (dag *ContextDAG) BudgetRemaining() int {
	return dag.budgetTotal - dag.budgetUsed
}

func (dag *ContextDAG) ShouldCompress() string {
	if dag.budgetUsed > dag.compressThresh {
		return "full"
	}
	for _, id := range dag.activeChain {
		node := dag.nodes[id]
		if node != nil && !node.Compressed && node.TokenCount > dag.nodeCompThresh {
			return "selective"
		}
	}
	if dag.BudgetRemaining() < dag.budgetTotal/5 {
		return "selective"
	}
	return "none"
}

func (dag *ContextDAG) ActiveContext() []ContextNode {
	result := make([]ContextNode, 0)
	for _, id := range dag.activeChain {
		node := dag.nodes[id]
		if node != nil && !node.Skipped && !node.Folded {
			result = append(result, *node)
		}
	}
	return result
}

func (dag *ContextDAG) Stats() map[string]interface{} {
	active := 0
	compressed := 0
	folded := 0
	skipped := 0
	for _, n := range dag.nodes {
		if n.Skipped {
			skipped++
		} else if n.Folded {
			folded++
		} else if n.Compressed {
			compressed++
			active++
		} else {
			active++
		}
	}
	return map[string]interface{}{
		"total_nodes":      len(dag.nodes),
		"active_nodes":     active,
		"compressed_nodes": compressed,
		"folded_nodes":     folded,
		"skipped_nodes":    skipped,
		"budget_total":     dag.budgetTotal,
		"budget_used":      dag.budgetUsed,
		"budget_remaining": dag.BudgetRemaining(),
		"compression":      dag.ShouldCompress(),
	}
}
