package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

type WebhookAdapter struct {
	config  ChannelConfig
	handler func(ctx context.Context, msg InboundMessage) error
	mu      sync.Mutex
	server  *http.Server
}

func NewWebhookAdapter(config ChannelConfig) *WebhookAdapter {
	return &WebhookAdapter{config: config}
}

func (w *WebhookAdapter) Info() AdapterInfo {
	return AdapterInfo{
		Name:         "webhook",
		Version:      "1.0.0",
		Capabilities: []string{"inbound", "outbound", "json", "text"},
	}
}

func (w *WebhookAdapter) OnMessage(handler func(ctx context.Context, msg InboundMessage) error) {
	w.mu.Lock()
	w.handler = handler
	w.mu.Unlock()
}

func (w *WebhookAdapter) Start(ctx context.Context) error {
	port := w.config.Credentials["port"]
	if port == "" {
		port = "9100"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/webhook", func(rw http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(rw, "POST required", http.StatusMethodNotAllowed)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(rw, "bad body", http.StatusBadRequest)
			return
		}

		var payload struct {
			Message  string `json:"message"`
			SenderID string `json:"sender_id"`
			ThreadID string `json:"thread_id"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			http.Error(rw, "invalid JSON", http.StatusBadRequest)
			return
		}

		msg := InboundMessage{
			ChannelType: "webhook",
			ChannelID:   w.config.Name,
			SenderID:    payload.SenderID,
			SenderName:  payload.SenderID,
			Content:     payload.Message,
			ThreadID:    payload.ThreadID,
			Timestamp:   time.Now().UTC().Format(time.RFC3339),
		}

		w.mu.Lock()
		h := w.handler
		w.mu.Unlock()

		if h != nil {
			if err := h(r.Context(), msg); err != nil {
				log.Printf("webhook: handler error: %v", err)
			}
		}

		rw.Header().Set("Content-Type", "application/json")
		json.NewEncoder(rw).Encode(map[string]string{"status": "received"})
	})

	mux.HandleFunc("/health", func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Content-Type", "application/json")
		json.NewEncoder(rw).Encode(map[string]string{"status": "ok"})
	})

	w.server = &http.Server{Addr: ":" + port, Handler: mux}
	log.Printf("webhook adapter: listening on :%s for channel %s", port, w.config.Name)

	go func() {
		<-ctx.Done()
		w.Stop()
	}()

	return w.server.ListenAndServe()
}

func (w *WebhookAdapter) Stop() error {
	if w.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return w.server.Shutdown(ctx)
	}
	return nil
}

func (w *WebhookAdapter) SendMessage(_ context.Context, target Target, msg OutboundMessage) error {
	callbackURL := w.config.Credentials["callback_url"]
	if callbackURL == "" {
		return fmt.Errorf("webhook: no callback_url configured")
	}

	payload, _ := json.Marshal(map[string]interface{}{
		"channel_id": target.ChannelID,
		"user_id":    target.UserID,
		"content":    msg.Content,
		"thread_id":  msg.ThreadID,
	})

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(callbackURL, "application/json",
		io.NopCloser(io.Reader(bytes.NewReader(payload))))
	if err != nil {
		return fmt.Errorf("webhook callback failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("webhook callback returned %d", resp.StatusCode)
	}
	return nil
}
