package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

type LLMClient struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

type LLMMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type LLMRequest struct {
	Model       string       `json:"model"`
	Messages    []LLMMessage `json:"messages"`
	MaxTokens   int          `json:"max_tokens"`
	Temperature float64      `json:"temperature,omitempty"`
}

type LLMResponse struct {
	ID    string `json:"id"`
	Model string `json:"model"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

func NewLLMClient() *LLMClient {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	baseURL := os.Getenv("LLM_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}
	model := os.Getenv("LLM_MODEL")
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}
	return &LLMClient{
		apiKey:  apiKey,
		baseURL: baseURL,
		model:   model,
		client:  &http.Client{Timeout: 120 * time.Second},
	}
}

func (c *LLMClient) Available() bool {
	return c.apiKey != ""
}

func (c *LLMClient) Complete(systemPrompt string, userMessage string, maxTokens int) (string, int, error) {
	if !c.Available() {
		return fmt.Sprintf("[LLM unavailable] Would process: %s", userMessage), 0, nil
	}

	reqBody := LLMRequest{
		Model: c.model,
		Messages: []LLMMessage{
			{Role: "user", Content: userMessage},
		},
		MaxTokens: maxTokens,
	}

	payload, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("POST", c.baseURL+"/v1/messages", bytes.NewReader(payload))
	if err != nil {
		return "", 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	if systemPrompt != "" {
		var withSystem struct {
			Model    string       `json:"model"`
			System   string       `json:"system"`
			Messages []LLMMessage `json:"messages"`
			MaxTokens int         `json:"max_tokens"`
		}
		withSystem.Model = c.model
		withSystem.System = systemPrompt
		withSystem.Messages = reqBody.Messages
		withSystem.MaxTokens = maxTokens
		payload, _ = json.Marshal(withSystem)
		req.Body = io.NopCloser(bytes.NewReader(payload))
		req.ContentLength = int64(len(payload))
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("llm request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		log.Printf("llm: HTTP %d: %s", resp.StatusCode, string(body))
		return "", 0, fmt.Errorf("llm returned HTTP %d", resp.StatusCode)
	}

	var llmResp LLMResponse
	if err := json.Unmarshal(body, &llmResp); err != nil {
		return "", 0, fmt.Errorf("llm response parse: %w", err)
	}

	text := ""
	for _, c := range llmResp.Content {
		if c.Type == "text" {
			text += c.Text
		}
	}

	totalTokens := llmResp.Usage.InputTokens + llmResp.Usage.OutputTokens
	return text, totalTokens, nil
}
