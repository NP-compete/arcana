package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/NP-compete/arcana/pkg/server"
)

type WebhookConfig struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Endpoint string   `json:"endpoint"`
	Secret   string   `json:"secret,omitempty"`
	Format   string   `json:"format"`
	Events   []string `json:"events"`
	Enabled  bool     `json:"enabled"`
}

type WebhookEvent struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Timestamp string                 `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

type DeliveryRecord struct {
	WebhookID  string `json:"webhook_id"`
	EventID    string `json:"event_id"`
	StatusCode int    `json:"status_code"`
	Success    bool   `json:"success"`
	Attempts   int    `json:"attempts"`
	Error      string `json:"error,omitempty"`
	Timestamp  string `json:"timestamp"`
}

type NotifyStore struct {
	mu         sync.RWMutex
	webhooks   map[string]*WebhookConfig
	deliveries []DeliveryRecord
	client     *http.Client
}

func NewNotifyStore() *NotifyStore {
	return &NotifyStore{
		webhooks:   make(map[string]*WebhookConfig),
		deliveries: make([]DeliveryRecord, 0),
		client:     &http.Client{Timeout: 10 * time.Second},
	}
}

func (s *NotifyStore) Register(wh WebhookConfig) {
	s.mu.Lock()
	s.webhooks[wh.ID] = &wh
	s.mu.Unlock()
}

func (s *NotifyStore) List() []WebhookConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := make([]WebhookConfig, 0, len(s.webhooks))
	for _, wh := range s.webhooks {
		list = append(list, *wh)
	}
	return list
}

func (s *NotifyStore) Dispatch(event WebhookEvent) {
	s.mu.RLock()
	webhooks := make([]*WebhookConfig, 0)
	for _, wh := range s.webhooks {
		if !wh.Enabled {
			continue
		}
		for _, ev := range wh.Events {
			if ev == event.Type || ev == "*" {
				webhooks = append(webhooks, wh)
				break
			}
		}
	}
	s.mu.RUnlock()

	for _, wh := range webhooks {
		go s.deliver(wh, event)
	}
}

func (s *NotifyStore) deliver(wh *WebhookConfig, event WebhookEvent) {
	payload := formatPayload(wh.Format, event)
	body, _ := json.Marshal(payload)

	backoff := []time.Duration{0, 1 * time.Second, 5 * time.Second, 30 * time.Second, 2 * time.Minute}
	var lastErr string
	var statusCode int

	for attempt, delay := range backoff {
		if delay > 0 {
			time.Sleep(delay)
		}

		req, _ := http.NewRequest("POST", wh.Endpoint, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Arcana-Event", event.Type)
		req.Header.Set("X-Arcana-Delivery", event.ID)

		if wh.Secret != "" {
			mac := hmac.New(sha256.New, []byte(wh.Secret))
			mac.Write(body)
			sig := hex.EncodeToString(mac.Sum(nil))
			req.Header.Set("X-Arcana-Signature", sig)
		}

		resp, err := s.client.Do(req)
		if err != nil {
			lastErr = err.Error()
			log.Printf("notify: delivery failed for %s (attempt %d): %v", wh.Name, attempt+1, err)
			continue
		}
		statusCode = resp.StatusCode
		resp.Body.Close()

		if statusCode < 300 {
			s.recordDelivery(wh.ID, event.ID, statusCode, true, attempt+1, "")
			return
		}
		lastErr = fmt.Sprintf("HTTP %d", statusCode)
	}

	s.recordDelivery(wh.ID, event.ID, statusCode, false, len(backoff), lastErr)
	log.Printf("notify: delivery permanently failed for %s event %s: %s", wh.Name, event.ID, lastErr)
}

func (s *NotifyStore) recordDelivery(webhookID, eventID string, statusCode int, success bool, attempts int, errMsg string) {
	s.mu.Lock()
	s.deliveries = append(s.deliveries, DeliveryRecord{
		WebhookID:  webhookID,
		EventID:    eventID,
		StatusCode: statusCode,
		Success:    success,
		Attempts:   attempts,
		Error:      errMsg,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
	})
	if len(s.deliveries) > 1000 {
		s.deliveries = s.deliveries[len(s.deliveries)-1000:]
	}
	s.mu.Unlock()
}

func formatPayload(format string, event WebhookEvent) interface{} {
	switch format {
	case "slack":
		text := fmt.Sprintf("*[%s]* %s", event.Type, event.Data["message"])
		return map[string]string{"text": text}
	case "pagerduty":
		return map[string]interface{}{
			"routing_key":  event.Data["routing_key"],
			"event_action": "trigger",
			"payload": map[string]interface{}{
				"summary":  fmt.Sprintf("Arcana: %s", event.Type),
				"severity": "warning",
				"source":   "arcana",
			},
		}
	default:
		return event
	}
}

func main() {
	httpSrv := server.New(server.Config{
		ServiceName: "notify",
		Port:        "8111",
	})

	store := NewNotifyStore()

	httpSrv.HandleFunc("/api/v1/webhooks", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, http.StatusOK, map[string]interface{}{"webhooks": store.List()})
		case http.MethodPost:
			var wh WebhookConfig
			json.NewDecoder(r.Body).Decode(&wh)
			if wh.ID == "" {
				wh.ID = fmt.Sprintf("wh-%d", time.Now().UnixNano())
			}
			if !wh.Enabled {
				wh.Enabled = true
			}
			store.Register(wh)
			writeJSON(w, http.StatusCreated, wh)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	}))

	httpSrv.HandleFunc("/api/v1/webhooks/test", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "POST required")
			return
		}
		store.Dispatch(WebhookEvent{
			ID:        fmt.Sprintf("test-%d", time.Now().UnixNano()),
			Type:      "test.ping",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Data:      map[string]interface{}{"message": "Arcana webhook test"},
		})
		writeJSON(w, http.StatusOK, map[string]string{"status": "dispatched"})
	}))

	httpSrv.HandleFunc("/api/v1/webhooks/publish", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "POST required")
			return
		}
		var event WebhookEvent
		json.NewDecoder(r.Body).Decode(&event)
		if event.ID == "" {
			event.ID = fmt.Sprintf("evt-%d", time.Now().UnixNano())
		}
		if event.Timestamp == "" {
			event.Timestamp = time.Now().UTC().Format(time.RFC3339)
		}
		store.Dispatch(event)
		writeJSON(w, http.StatusOK, map[string]string{"status": "dispatched", "event_id": event.ID})
	}))

	httpSrv.HandleFunc("/api/v1/webhooks/deliveries", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		store.mu.RLock()
		deliveries := make([]DeliveryRecord, len(store.deliveries))
		copy(deliveries, store.deliveries)
		store.mu.RUnlock()
		writeJSON(w, http.StatusOK, map[string]interface{}{"deliveries": deliveries, "total": len(deliveries)})
	}))

	httpSrv.ListenAndServe()
}

func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", os.Getenv("CORS_ORIGIN"))
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
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
