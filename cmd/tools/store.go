package main

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

type ToolsStore struct {
	mu      sync.RWMutex
	servers map[string]*ToolServer
	calls   map[string]*toolCallRecord
}

func NewToolsStore() *ToolsStore {
	s := &ToolsStore{
		servers: make(map[string]*ToolServer),
		calls:   make(map[string]*toolCallRecord),
	}
	s.seedDefaultServers()
	return s
}

func (s *ToolsStore) seedDefaultServers() {
	defaultTools := map[string][]ToolSchema{
		"filesystem": {
			{Name: "read_file", Server: "filesystem", Description: "Read contents of a file", InputSchema: map[string]interface{}{
				"type": "object", "properties": map[string]interface{}{"path": map[string]interface{}{"type": "string"}},
				"required": []string{"path"},
			}},
			{Name: "write_file", Server: "filesystem", Description: "Write contents to a file", InputSchema: map[string]interface{}{
				"type": "object", "properties": map[string]interface{}{
					"path":    map[string]interface{}{"type": "string"},
					"content": map[string]interface{}{"type": "string"},
				},
				"required": []string{"path", "content"},
			}},
		},
		"web": {
			{Name: "fetch_url", Server: "web", Description: "Fetch content from a URL", InputSchema: map[string]interface{}{
				"type": "object", "properties": map[string]interface{}{"url": map[string]interface{}{"type": "string"}},
				"required": []string{"url"},
			}},
			{Name: "search", Server: "web", Description: "Search the web", InputSchema: map[string]interface{}{
				"type": "object", "properties": map[string]interface{}{"query": map[string]interface{}{"type": "string"}},
				"required": []string{"query"},
			}},
		},
		"database": {
			{Name: "query", Server: "database", Description: "Execute a read-only SQL query", InputSchema: map[string]interface{}{
				"type": "object", "properties": map[string]interface{}{"sql": map[string]interface{}{"type": "string"}},
				"required": []string{"sql"},
			}},
		},
	}

	for name, tools := range defaultTools {
		s.servers[name] = &ToolServer{
			Name:            name,
			Tools:           tools,
			ProtocolVersion: "2024-11-05",
		}
	}
}

func (s *ToolsStore) ListServers() []ToolServer {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]ToolServer, 0, len(s.servers))
	for _, srv := range s.servers {
		result = append(result, *srv)
	}
	return result
}

func (s *ToolsStore) GetToolSchema(server, tool string) (*ToolSchema, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	srv, ok := s.servers[server]
	if !ok {
		return nil, false
	}
	for _, t := range srv.Tools {
		if t.Name == tool {
			copy := t
			return &copy, true
		}
	}
	return nil, false
}

func (s *ToolsStore) CallTool(toolName, server string, params map[string]interface{}, agentID string) (*ToolCall, error) {
	s.mu.RLock()
	srv, ok := s.servers[server]
	if !ok {
		s.mu.RUnlock()
		return nil, errServerNotFound
	}
	var schema *ToolSchema
	for i := range srv.Tools {
		if srv.Tools[i].Name == toolName {
			schema = &srv.Tools[i]
			break
		}
	}
	s.mu.RUnlock()

	if schema == nil {
		return nil, errToolNotFound
	}

	if params == nil {
		params = make(map[string]interface{})
	}

	start := time.Now()
	id := uuid.New().String()

	result := s.simulateMCPCall(server, toolName, params)
	latency := time.Since(start).Milliseconds()

	call := ToolCall{
		ID:        id,
		Tool:      toolName,
		Server:    server,
		Params:    params,
		AgentID:   agentID,
		Result:    result,
		Status:    ToolCallCompleted,
		LatencyMs: latency,
	}

	s.mu.Lock()
	s.calls[id] = &toolCallRecord{call: call, createdAt: time.Now().UTC()}
	s.mu.Unlock()

	return &call, nil
}

func (s *ToolsStore) simulateMCPCall(server, tool string, params map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"server":  server,
		"tool":    tool,
		"success": true,
		"output":  map[string]interface{}{"params_received": params, "message": "MCP tool executed successfully"},
	}
}

func (s *ToolsStore) TranslateSchema(sourceVersion, targetVersion string, schema map[string]interface{}) map[string]interface{} {
	translated := make(map[string]interface{})
	for k, v := range schema {
		translated[k] = v
	}
	translated["_toolglot"] = map[string]interface{}{
		"source_version": sourceVersion,
		"target_version": targetVersion,
		"translated":     true,
	}
	if targetVersion == "2024-11-05" && sourceVersion == "2024-06-01" {
		if props, ok := translated["inputSchema"].(map[string]interface{}); ok {
			translated["inputSchema"] = props
		} else if props, ok := translated["input_schema"].(map[string]interface{}); ok {
			translated["inputSchema"] = props
			delete(translated, "input_schema")
		}
	}
	return translated
}

type serverNotFoundError struct{}

func (e *serverNotFoundError) Error() string { return "MCP server not found" }

type toolNotFoundError struct{}

func (e *toolNotFoundError) Error() string { return "tool not found on server" }

var errServerNotFound = &serverNotFoundError{}
var errToolNotFound = &toolNotFoundError{}
