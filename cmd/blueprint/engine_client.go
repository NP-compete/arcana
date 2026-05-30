package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type EngineClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewEngineClient() *EngineClient {
	baseURL := os.Getenv("ENGINE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8081"
	}
	return &EngineClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type engineTaskRequest struct {
	Agent string                 `json:"agent"`
	Input map[string]interface{} `json:"input"`
	Model map[string]interface{} `json:"model,omitempty"`
	Steps int                    `json:"steps,omitempty"`
}

type engineTaskResponse struct {
	ID string `json:"id"`
}

func (c *EngineClient) SubmitTask(agent string, input map[string]interface{}, model string) (string, error) {
	reqBody := engineTaskRequest{
		Agent: agent,
		Input: input,
		Steps: 3,
	}
	if model != "" {
		reqBody.Model = map[string]interface{}{"model": model}
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	resp, err := c.httpClient.Post(c.baseURL+"/api/v1/tasks", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("engine unreachable: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("engine returned %d: %s", resp.StatusCode, string(respBody))
	}

	var task engineTaskResponse
	if err := json.Unmarshal(respBody, &task); err != nil {
		return "", fmt.Errorf("invalid engine response: %w", err)
	}
	return task.ID, nil
}
