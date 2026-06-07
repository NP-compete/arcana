package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type OpenAIChatRequest struct {
	Model       string              `json:"model"`
	Messages    []OpenAIChatMessage `json:"messages"`
	MaxTokens   int                 `json:"max_tokens,omitempty"`
	Temperature float64             `json:"temperature,omitempty"`
	Stream      bool                `json:"stream,omitempty"`
}

type OpenAIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAIChatResponse struct {
	ID      string             `json:"id"`
	Object  string             `json:"object"`
	Created int64              `json:"created"`
	Model   string             `json:"model"`
	Choices []OpenAIChatChoice `json:"choices"`
	Usage   OpenAIUsage        `json:"usage"`
}

type OpenAIChatChoice struct {
	Index        int               `json:"index"`
	Message      OpenAIChatMessage `json:"message"`
	FinishReason string            `json:"finish_reason"`
}

type OpenAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type OpenAIStreamDelta struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index int `json:"index"`
		Delta struct {
			Role    string `json:"role,omitempty"`
			Content string `json:"content,omitempty"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
}

type OpenAIModel struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

func handleOpenAIChatCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		writeOpenAIError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req OpenAIChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeOpenAIError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Model == "" {
		writeOpenAIError(w, http.StatusBadRequest, "model is required")
		return
	}

	agentName := req.Model
	lastMessage := ""
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" {
			lastMessage = req.Messages[i].Content
			break
		}
	}

	engineHost := os.Getenv("ENGINE_HOST")
	if engineHost == "" {
		engineHost = "arcana-engine.arcana.svc.cluster.local"
	}

	taskPayload, _ := json.Marshal(map[string]interface{}{
		"agent": agentName,
		"input": lastMessage,
	})

	resp, err := http.Post(
		fmt.Sprintf("http://%s:8081/api/v1/tasks", engineHost),
		"application/json",
		strings.NewReader(string(taskPayload)),
	)

	var output string
	var tokens int
	if err != nil {
		output = fmt.Sprintf("Agent %s is not available: %v", agentName, err)
		tokens = 20
	} else {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		var taskResp map[string]interface{}
		json.Unmarshal(body, &taskResp)

		taskID, _ := taskResp["id"].(string)
		output = fmt.Sprintf("Task %s submitted to agent %s. Use task ID to check status.", taskID, agentName)
		tokens = 30

		if result, ok := taskResp["result"].(map[string]interface{}); ok {
			if out, ok := result["output"].(string); ok {
				output = out
			}
		}
	}

	now := time.Now()

	if req.Stream {
		handleStreamResponse(w, agentName, output, now)
		return
	}

	response := OpenAIChatResponse{
		ID:      fmt.Sprintf("chatcmpl-%d", now.UnixNano()),
		Object:  "chat.completion",
		Created: now.Unix(),
		Model:   agentName,
		Choices: []OpenAIChatChoice{
			{
				Index:        0,
				Message:      OpenAIChatMessage{Role: "assistant", Content: output},
				FinishReason: "stop",
			},
		},
		Usage: OpenAIUsage{
			PromptTokens:     len(lastMessage) / 4,
			CompletionTokens: tokens,
			TotalTokens:      len(lastMessage)/4 + tokens,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(response)
}

func handleStreamResponse(w http.ResponseWriter, model, content string, created time.Time) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeOpenAIError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	id := fmt.Sprintf("chatcmpl-%d", created.UnixNano())

	roleDelta := OpenAIStreamDelta{
		ID:      id,
		Object:  "chat.completion.chunk",
		Created: created.Unix(),
		Model:   model,
		Choices: []struct {
			Index int `json:"index"`
			Delta struct {
				Role    string `json:"role,omitempty"`
				Content string `json:"content,omitempty"`
			} `json:"delta"`
			FinishReason *string `json:"finish_reason"`
		}{
			{Index: 0, Delta: struct {
				Role    string `json:"role,omitempty"`
				Content string `json:"content,omitempty"`
			}{Role: "assistant"}, FinishReason: nil},
		},
	}
	data, _ := json.Marshal(roleDelta)
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()

	words := strings.Fields(content)
	for i, word := range words {
		chunk := word
		if i < len(words)-1 {
			chunk += " "
		}
		delta := OpenAIStreamDelta{
			ID:      id,
			Object:  "chat.completion.chunk",
			Created: created.Unix(),
			Model:   model,
			Choices: []struct {
				Index int `json:"index"`
				Delta struct {
					Role    string `json:"role,omitempty"`
					Content string `json:"content,omitempty"`
				} `json:"delta"`
				FinishReason *string `json:"finish_reason"`
			}{
				{Index: 0, Delta: struct {
					Role    string `json:"role,omitempty"`
					Content string `json:"content,omitempty"`
				}{Content: chunk}, FinishReason: nil},
			},
		}
		data, _ := json.Marshal(delta)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
		time.Sleep(20 * time.Millisecond)
	}

	stop := "stop"
	finalDelta := OpenAIStreamDelta{
		ID:      id,
		Object:  "chat.completion.chunk",
		Created: created.Unix(),
		Model:   model,
		Choices: []struct {
			Index int `json:"index"`
			Delta struct {
				Role    string `json:"role,omitempty"`
				Content string `json:"content,omitempty"`
			} `json:"delta"`
			FinishReason *string `json:"finish_reason"`
		}{
			{Index: 0, Delta: struct {
				Role    string `json:"role,omitempty"`
				Content string `json:"content,omitempty"`
			}{}, FinishReason: &stop},
		},
	}
	data, _ = json.Marshal(finalDelta)
	fmt.Fprintf(w, "data: %s\n\n", data)
	fmt.Fprint(w, "data: [DONE]\n\n")
	flusher.Flush()
}

func handleOpenAIModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeOpenAIError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	meshHost := os.Getenv("MESH_HOST")
	if meshHost == "" {
		meshHost = "arcana-mesh.arcana.svc.cluster.local"
	}

	resp, err := http.Get(fmt.Sprintf("http://%s:8083/api/v1/agents", meshHost))
	models := []OpenAIModel{}

	if err == nil {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		var agentList struct {
			Agents []struct {
				Name string `json:"name"`
			} `json:"agents"`
		}
		if json.Unmarshal(body, &agentList) == nil {
			for _, a := range agentList.Agents {
				models = append(models, OpenAIModel{
					ID:      a.Name,
					Object:  "model",
					Created: time.Now().Unix(),
					OwnedBy: "arcana",
				})
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"object": "list",
		"data":   models,
	})
}

func writeOpenAIError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]interface{}{
			"message": msg,
			"type":    "invalid_request_error",
			"code":    status,
		},
	})
}
