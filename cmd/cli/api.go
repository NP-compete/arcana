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

func apiGet(path string) map[string]interface{} {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", apiBase+path, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating request: %v\n", err)
		os.Exit(1)
	}
	addAuth(req)
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading response: %v\n", err)
		os.Exit(1)
	}
	if resp.StatusCode >= 400 {
		fmt.Fprintf(os.Stderr, "API error (HTTP %d): %s\n", resp.StatusCode, string(body))
		os.Exit(1)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing response: %v\n", err)
		os.Exit(1)
	}
	return result
}

func apiPost(path string, data interface{}) map[string]interface{} {
	client := &http.Client{Timeout: 30 * time.Second}
	var body io.Reader
	if data != nil {
		b, err := json.Marshal(data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error marshaling request: %v\n", err)
			os.Exit(1)
		}
		body = strings.NewReader(string(b))
	}
	req, err := http.NewRequest("POST", apiBase+path, body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating request: %v\n", err)
		os.Exit(1)
	}
	req.Header.Set("Content-Type", "application/json")
	addAuth(req)
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading response: %v\n", err)
		os.Exit(1)
	}
	if resp.StatusCode >= 400 {
		fmt.Fprintf(os.Stderr, "API error (HTTP %d): %s\n", resp.StatusCode, string(b))
		os.Exit(1)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(b, &result); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing response: %v\n", err)
		os.Exit(1)
	}
	return result
}

func apiDelete(path string) map[string]interface{} {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("DELETE", apiBase+path, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating request: %v\n", err)
		os.Exit(1)
	}
	addAuth(req)
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading response: %v\n", err)
		os.Exit(1)
	}
	if resp.StatusCode >= 400 {
		fmt.Fprintf(os.Stderr, "API error (HTTP %d): %s\n", resp.StatusCode, string(b))
		os.Exit(1)
	}
	var result map[string]interface{}
	if len(b) > 0 {
		if err := json.Unmarshal(b, &result); err != nil {
			// DELETE may return empty body on success
			return map[string]interface{}{"status": "deleted"}
		}
	} else {
		return map[string]interface{}{"status": "deleted"}
	}
	return result
}

func addAuth(req *http.Request) {
	if key := os.Getenv("ARCANA_API_KEY"); key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
}

func printJSON(v interface{}) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(b))
}
