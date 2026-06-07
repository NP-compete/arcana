package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/NP-compete/arcana/pkg/server"
)

var engineHost string
var httpClient = &http.Client{Timeout: 10 * time.Second}

func main() {
	engineHost = os.Getenv("ENGINE_HOST")
	if engineHost == "" {
		engineHost = "arcana-engine.arcana.svc.cluster.local"
	}

	httpSrv := server.New(server.Config{
		ServiceName: "agui",
		Port:        "8084",
	})

	hub := NewEventHub()

	httpSrv.HandleFunc("/api/v1/agui/events", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		handleSSE(w, r, hub)
	}))

	httpSrv.HandleFunc("/api/v1/agui/subscribe", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		handleSubscribe(w, r, hub)
	}))

	httpSrv.HandleFunc("/api/v1/agui/publish", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		handlePublish(w, r, hub)
	}))

	go pollEngineStatus(hub)

	httpSrv.ListenAndServe()
}

type AGUIEvent struct {
	Type      string                 `json:"type"`
	AgentName string                 `json:"agent_name,omitempty"`
	TaskID    string                 `json:"task_id,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Timestamp string                 `json:"timestamp"`
}

type EventHub struct {
	mu       sync.RWMutex
	clients  map[chan AGUIEvent]string
	history  []AGUIEvent
	maxHist  int
}

func NewEventHub() *EventHub {
	return &EventHub{
		clients: make(map[chan AGUIEvent]string),
		history: make([]AGUIEvent, 0),
		maxHist: 100,
	}
}

func (h *EventHub) Subscribe(agentFilter string) chan AGUIEvent {
	ch := make(chan AGUIEvent, 50)
	h.mu.Lock()
	h.clients[ch] = agentFilter
	h.mu.Unlock()
	return ch
}

func (h *EventHub) Unsubscribe(ch chan AGUIEvent) {
	h.mu.Lock()
	delete(h.clients, ch)
	h.mu.Unlock()
	close(ch)
}

func (h *EventHub) Publish(event AGUIEvent) {
	if event.Timestamp == "" {
		event.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}
	h.mu.Lock()
	h.history = append(h.history, event)
	if len(h.history) > h.maxHist {
		h.history = h.history[len(h.history)-h.maxHist:]
	}
	h.mu.Unlock()

	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch, filter := range h.clients {
		if filter != "" && filter != event.AgentName {
			continue
		}
		select {
		case ch <- event:
		default:
		}
	}
}

func handleSSE(w http.ResponseWriter, r *http.Request, hub *EventHub) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	agentFilter := r.URL.Query().Get("agent")

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", corsOrigin())

	ch := hub.Subscribe(agentFilter)
	defer hub.Unsubscribe(ch)

	hub.Publish(AGUIEvent{
		Type: "RunStarted",
		Data: map[string]interface{}{"message": "SSE connection established"},
	})
	flusher.Flush()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-ch:
			if !ok {
				return
			}
			data, _ := json.Marshal(event)
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, string(data))
			flusher.Flush()
		}
	}
}

func handleSubscribe(w http.ResponseWriter, r *http.Request, hub *EventHub) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET required")
		return
	}
	hub.mu.RLock()
	count := len(hub.clients)
	histLen := len(hub.history)
	hub.mu.RUnlock()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"active_subscribers": count,
		"event_history":      histLen,
	})
}

func handlePublish(w http.ResponseWriter, r *http.Request, hub *EventHub) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST required")
		return
	}
	var event AGUIEvent
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if event.Type == "" {
		writeError(w, http.StatusBadRequest, "type is required")
		return
	}
	hub.Publish(event)
	writeJSON(w, http.StatusOK, map[string]string{"status": "published"})
}

func pollEngineStatus(hub *EventHub) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	var prevTaskCount int
	for range ticker.C {
		resp, err := httpClient.Get(fmt.Sprintf("http://%s:8081/api/v1/tasks?limit=5", engineHost))
		if err != nil {
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		var result struct {
			Tasks []struct {
				ID     string `json:"id"`
				Agent  string `json:"agent"`
				Status string `json:"status"`
			} `json:"tasks"`
			Total int `json:"total"`
		}
		if json.Unmarshal(body, &result) != nil {
			continue
		}

		if result.Total != prevTaskCount {
			for _, t := range result.Tasks {
				eventType := "TaskUpdate"
				switch t.Status {
				case "completed":
					eventType = "RunCompleted"
				case "failed":
					eventType = "RunFailed"
				case "running":
					eventType = "StepStarted"
				}
				hub.Publish(AGUIEvent{
					Type:      eventType,
					AgentName: t.Agent,
					TaskID:    t.ID,
					Data:      map[string]interface{}{"status": t.Status},
				})
			}
			prevTaskCount = result.Total
		}
	}
}

func corsOrigin() string {
	if origin := os.Getenv("CORS_ORIGIN"); origin != "" {
		return origin
	}
	return "*"
}

func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", corsOrigin())
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Cache-Control")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// Suppress unused import warnings
var _ = strings.TrimSpace
