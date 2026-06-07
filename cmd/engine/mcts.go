package main

import (
	"log"
	"math"
	"math/rand"
	"time"
)

type MCTSNode struct {
	Action     string
	Parent     *MCTSNode
	Children   []*MCTSNode
	Visits     int
	TotalValue float64
	Depth      int
}

type MCTSPlanner struct {
	maxDepth      int
	maxIterations int
	explorationC  float64
}

func NewMCTSPlanner() *MCTSPlanner {
	return &MCTSPlanner{
		maxDepth:      5,
		maxIterations: 100,
		explorationC:  1.41,
	}
}

func (p *MCTSPlanner) ShouldUseMCTS(complexity float64) bool {
	return complexity > 0.7
}

func (p *MCTSPlanner) Plan(goal string, candidateActions []string) string {
	if len(candidateActions) == 0 {
		return ""
	}

	root := &MCTSNode{Action: "root", Depth: 0}
	for _, action := range candidateActions {
		root.Children = append(root.Children, &MCTSNode{
			Action: action,
			Parent: root,
			Depth:  1,
		})
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 0; i < p.maxIterations; i++ {
		node := p.selectNode(root)
		if node.Depth < p.maxDepth && node.Visits > 0 {
			node = p.expand(node, rng)
		}
		value := p.simulate(node, rng)
		p.backpropagate(node, value)
	}

	best := p.bestChild(root)
	if best == nil {
		return candidateActions[0]
	}

	log.Printf("mcts: selected '%s' (visits=%d, avg_value=%.3f) from %d candidates",
		best.Action, best.Visits, best.TotalValue/float64(max2(best.Visits, 1)), len(candidateActions))
	return best.Action
}

func (p *MCTSPlanner) selectNode(node *MCTSNode) *MCTSNode {
	for len(node.Children) > 0 {
		node = p.uctSelect(node)
	}
	return node
}

func (p *MCTSPlanner) uctSelect(parent *MCTSNode) *MCTSNode {
	var best *MCTSNode
	bestUCT := -math.MaxFloat64

	for _, child := range parent.Children {
		if child.Visits == 0 {
			return child
		}
		exploitation := child.TotalValue / float64(child.Visits)
		exploration := p.explorationC * math.Sqrt(math.Log(float64(parent.Visits))/float64(child.Visits))
		uct := exploitation + exploration
		if uct > bestUCT {
			bestUCT = uct
			best = child
		}
	}
	return best
}

func (p *MCTSPlanner) expand(node *MCTSNode, rng *rand.Rand) *MCTSNode {
	child := &MCTSNode{
		Action: node.Action + "_sub",
		Parent: node,
		Depth:  node.Depth + 1,
	}
	node.Children = append(node.Children, child)
	return child
}

func (p *MCTSPlanner) simulate(_ *MCTSNode, rng *rand.Rand) float64 {
	return rng.Float64()
}

func (p *MCTSPlanner) backpropagate(node *MCTSNode, value float64) {
	for node != nil {
		node.Visits++
		node.TotalValue += value
		node = node.Parent
	}
}

func (p *MCTSPlanner) bestChild(node *MCTSNode) *MCTSNode {
	var best *MCTSNode
	bestVisits := 0
	for _, child := range node.Children {
		if child.Visits > bestVisits {
			bestVisits = child.Visits
			best = child
		}
	}
	return best
}

func max2(a, b int) int {
	if a > b {
		return a
	}
	return b
}
