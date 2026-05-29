package mcp

// Tool describes an MCP tool exposed by a server.
type Tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

// Resource describes an MCP resource exposed by a server.
type Resource struct {
	URI      string `json:"uri"`
	Name     string `json:"name"`
	MimeType string `json:"mimeType"`
}

// ContentPart is a single content element in a tool result.
type ContentPart struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// ToolResult is the outcome of an MCP tool invocation.
type ToolResult struct {
	Content []ContentPart `json:"content"`
	IsError bool          `json:"isError"`
}
