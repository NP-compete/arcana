package main

import (
	"encoding/json"
	"net/http"
)

type MCPServerInfo struct {
	Name         string                 `json:"name"`
	Version      string                 `json:"version"`
	Description  string                 `json:"description"`
	Capabilities map[string]interface{} `json:"capabilities"`
	Tools        []MCPToolInfo          `json:"tools"`
}

type MCPToolInfo struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

type MCPInvokeRequest struct {
	Method string                 `json:"method"`
	Params map[string]interface{} `json:"params"`
}

type MCPInvokeResponse struct {
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
}

type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (s *Server) handleMCPDiscovery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET required")
		return
	}

	servers := s.store.ListServers()
	tools := make([]MCPToolInfo, 0)
	for _, srv := range servers {
		for _, t := range srv.Tools {
			tools = append(tools, MCPToolInfo{
				Name:        srv.Name + "." + t.Name,
				Description: t.Description,
				InputSchema: t.InputSchema,
			})
		}
	}

	info := MCPServerInfo{
		Name:        "arcana-tools",
		Version:     "0.1.0",
		Description: "Arcana MCP tool gateway — unified access to all registered tool servers",
		Capabilities: map[string]interface{}{
			"tools":     true,
			"translate": true,
		},
		Tools: tools,
	}
	writeJSON(w, http.StatusOK, info)
}

func (s *Server) handleMCPInvoke(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST required")
		return
	}

	var req MCPInvokeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(MCPInvokeResponse{
			Error: &MCPError{Code: -32700, Message: "parse error"},
		})
		return
	}

	switch req.Method {
	case "tools/list":
		servers := s.store.ListServers()
		tools := make([]map[string]interface{}, 0)
		for _, srv := range servers {
			for _, t := range srv.Tools {
				tools = append(tools, map[string]interface{}{
					"name":        srv.Name + "." + t.Name,
					"description": t.Description,
					"inputSchema": t.InputSchema,
				})
			}
		}
		writeJSON(w, http.StatusOK, MCPInvokeResponse{Result: tools})

	case "tools/call":
		toolName, _ := req.Params["name"].(string)
		args, _ := req.Params["arguments"].(map[string]interface{})
		if toolName == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(MCPInvokeResponse{
				Error: &MCPError{Code: -32602, Message: "tool name required"},
			})
			return
		}

		serverName := ""
		localTool := toolName
		for _, srv := range s.store.ListServers() {
			for _, t := range srv.Tools {
				if srv.Name+"."+t.Name == toolName || t.Name == toolName {
					serverName = srv.Name
					localTool = t.Name
					break
				}
			}
			if serverName != "" {
				break
			}
		}

		if serverName == "" {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(MCPInvokeResponse{
				Error: &MCPError{Code: -32601, Message: "tool not found"},
			})
			return
		}

		result, err := s.store.CallTool(localTool, serverName, args, "")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(MCPInvokeResponse{
				Error: &MCPError{Code: -32000, Message: err.Error()},
			})
			return
		}
		writeJSON(w, http.StatusOK, MCPInvokeResponse{Result: result})

	default:
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(MCPInvokeResponse{
			Error: &MCPError{Code: -32601, Message: "method not found"},
		})
	}
}
