package a2a

// A2AClient communicates with remote A2A agents.
type A2AClient interface {
	Discover(url string) (AgentCard, error)
	SendTask(card AgentCard, task Task) (Task, error)
}
