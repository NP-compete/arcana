package a2a

// AgentCard describes a discoverable A2A agent.
type AgentCard struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	URL         string   `json:"url"`
	Skills      []string `json:"skills"`
	Protocols   []string `json:"protocols"`
}

// Part represents a content part within an A2A message.
type Part struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
	Data any    `json:"data,omitempty"`
}

// Message is a single turn in an A2A conversation.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Parts   []Part `json:"parts,omitempty"`
}

// Task represents an A2A task lifecycle.
type Task struct {
	ID      string    `json:"id"`
	Status  string    `json:"status"`
	Input   string    `json:"input"`
	Output  string    `json:"output,omitempty"`
	History []Message `json:"history,omitempty"`
}
