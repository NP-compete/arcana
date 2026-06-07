package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/NP-compete/arcana/pkg/server"
)

var meshHost string
var engineHost string

func main() {
	meshHost = os.Getenv("MESH_HOST")
	if meshHost == "" {
		meshHost = "arcana-mesh.arcana.svc.cluster.local"
	}
	engineHost = os.Getenv("ENGINE_HOST")
	if engineHost == "" {
		engineHost = "arcana-engine.arcana.svc.cluster.local"
	}

	httpSrv := server.New(server.Config{
		ServiceName: "channels",
		Port:        "8108",
	})

	router := NewChannelRouter()

	httpSrv.HandleFunc("/api/v1/channels", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"channels": router.ListChannels(),
			})
			return
		}
		if r.Method == http.MethodPost {
			var config ChannelConfig
			if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
				writeError(w, http.StatusBadRequest, "invalid JSON")
				return
			}
			if err := router.RegisterChannel(config); err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			writeJSON(w, http.StatusCreated, map[string]string{
				"name":   config.Name,
				"status": "registered",
			})
			return
		}
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}))

	httpSrv.HandleFunc("/api/v1/channels/send", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "POST required")
			return
		}
		var req struct {
			Channel  string          `json:"channel"`
			Target   Target          `json:"target"`
			Message  OutboundMessage `json:"message"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		if err := router.Send(r.Context(), req.Channel, req.Target, req.Message); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "sent"})
	}))

	httpSrv.ListenAndServe()
}

type ChannelRouter struct {
	adapters map[string]ChannelAdapter
	configs  map[string]ChannelConfig
}

func NewChannelRouter() *ChannelRouter {
	return &ChannelRouter{
		adapters: make(map[string]ChannelAdapter),
		configs:  make(map[string]ChannelConfig),
	}
}

func (cr *ChannelRouter) RegisterChannel(config ChannelConfig) error {
	if config.Name == "" || config.AdapterType == "" || config.AgentName == "" {
		return fmt.Errorf("name, adapter_type, and agent_name required")
	}

	var adapter ChannelAdapter
	switch config.AdapterType {
	case "webhook":
		adapter = NewWebhookAdapter(config)
	default:
		return fmt.Errorf("unknown adapter type: %s", config.AdapterType)
	}

	adapter.OnMessage(func(ctx context.Context, msg InboundMessage) error {
		return cr.routeToAgent(ctx, config.AgentName, msg)
	})

	cr.adapters[config.Name] = adapter
	cr.configs[config.Name] = config

	go func() {
		if err := adapter.Start(context.Background()); err != nil {
			log.Printf("channel %s: adapter stopped: %v", config.Name, err)
		}
	}()

	log.Printf("channel router: registered %s (type=%s, agent=%s)", config.Name, config.AdapterType, config.AgentName)
	return nil
}

func (cr *ChannelRouter) ListChannels() []map[string]interface{} {
	channels := make([]map[string]interface{}, 0)
	for name, config := range cr.configs {
		adapter := cr.adapters[name]
		info := adapter.Info()
		channels = append(channels, map[string]interface{}{
			"name":         name,
			"adapter_type": config.AdapterType,
			"agent_name":   config.AgentName,
			"adapter_info": info,
		})
	}
	return channels
}

func (cr *ChannelRouter) Send(ctx context.Context, channelName string, target Target, msg OutboundMessage) error {
	adapter, ok := cr.adapters[channelName]
	if !ok {
		return fmt.Errorf("channel %s not found", channelName)
	}
	return adapter.SendMessage(ctx, target, msg)
}

func (cr *ChannelRouter) routeToAgent(_ context.Context, agentName string, msg InboundMessage) error {
	payload, _ := json.Marshal(map[string]interface{}{
		"agent": agentName,
		"input": msg.Content,
	})

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Post(
		fmt.Sprintf("http://%s:8081/api/v1/tasks", engineHost),
		"application/json",
		bytes.NewReader(payload),
	)
	if err != nil {
		return fmt.Errorf("route to agent %s: %w", agentName, err)
	}
	defer resp.Body.Close()

	log.Printf("channel router: routed message from %s to agent %s (status=%d)", msg.SenderID, agentName, resp.StatusCode)
	return nil
}

func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
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
