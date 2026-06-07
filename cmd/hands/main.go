package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/NP-compete/arcana/pkg/server"
)

var engineHost string

func main() {
	engineHost = os.Getenv("ENGINE_HOST")
	if engineHost == "" {
		engineHost = "arcana-engine.arcana.svc.cluster.local"
	}

	httpSrv := server.New(server.Config{
		ServiceName: "hands",
		Port:        "8109",
	})

	scheduler := NewHandScheduler()

	httpSrv.HandleFunc("/api/v1/hands", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"hands": scheduler.ListHands(),
			})
		case http.MethodPost:
			var hand HandConfig
			if err := json.NewDecoder(r.Body).Decode(&hand); err != nil {
				writeError(w, http.StatusBadRequest, "invalid JSON")
				return
			}
			if err := scheduler.Register(hand); err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			writeJSON(w, http.StatusCreated, map[string]string{
				"name":   hand.Name,
				"status": "registered",
			})
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	}))

	httpSrv.HandleFunc("/api/v1/hands/trigger/", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "POST required")
			return
		}
		name := r.URL.Path[len("/api/v1/hands/trigger/"):]
		result, err := scheduler.TriggerManual(name)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, result)
	}))

	go scheduler.Start()

	httpSrv.ListenAndServe()
}

type HandConfig struct {
	Name          string            `json:"name"`
	Description   string            `json:"description"`
	Schedule      string            `json:"schedule"`
	AgentName     string            `json:"agent_name"`
	Goal          string            `json:"goal"`
	Tools         []string          `json:"tools,omitempty"`
	MaxRuntime    string            `json:"max_runtime,omitempty"`
	MaxCostPerRun float64           `json:"max_cost_per_run,omitempty"`
	Settings      map[string]string `json:"settings,omitempty"`
	Enabled       bool              `json:"enabled"`
}

type HandRun struct {
	HandName  string    `json:"hand_name"`
	TaskID    string    `json:"task_id"`
	Status    string    `json:"status"`
	StartedAt time.Time `json:"started_at"`
}

type HandStatus struct {
	Config     HandConfig `json:"config"`
	LastRun    *HandRun   `json:"last_run,omitempty"`
	RunCount   int        `json:"run_count"`
	NextRunAt  string     `json:"next_run_at,omitempty"`
}

type HandScheduler struct {
	mu    sync.RWMutex
	hands map[string]*HandStatus
}

func NewHandScheduler() *HandScheduler {
	return &HandScheduler{
		hands: make(map[string]*HandStatus),
	}
}

func (s *HandScheduler) Register(config HandConfig) error {
	if config.Name == "" || config.AgentName == "" || config.Goal == "" {
		return fmt.Errorf("name, agent_name, and goal are required")
	}
	if !config.Enabled {
		config.Enabled = true
	}

	s.mu.Lock()
	s.hands[config.Name] = &HandStatus{Config: config}
	s.mu.Unlock()

	log.Printf("hands: registered %s (agent=%s, schedule=%s)", config.Name, config.AgentName, config.Schedule)
	return nil
}

func (s *HandScheduler) ListHands() []HandStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := make([]HandStatus, 0, len(s.hands))
	for _, h := range s.hands {
		list = append(list, *h)
	}
	return list
}

func (s *HandScheduler) TriggerManual(name string) (*HandRun, error) {
	s.mu.RLock()
	hand, ok := s.hands[name]
	s.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("hand %s not found", name)
	}
	return s.executeHand(hand)
}

func (s *HandScheduler) Start() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	log.Println("hands: scheduler started (60s check interval)")
	for range ticker.C {
		s.mu.RLock()
		for _, hand := range s.hands {
			if !hand.Config.Enabled {
				continue
			}
			if hand.Config.Schedule == "" {
				continue
			}
			if shouldRun(hand) {
				go func(h *HandStatus) {
					if _, err := s.executeHand(h); err != nil {
						log.Printf("hands: execution failed for %s: %v", h.Config.Name, err)
					}
				}(hand)
			}
		}
		s.mu.RUnlock()
	}
}

func shouldRun(hand *HandStatus) bool {
	if hand.LastRun == nil {
		return true
	}
	elapsed := time.Since(hand.LastRun.StartedAt)
	switch hand.Config.Schedule {
	case "hourly":
		return elapsed >= time.Hour
	case "daily":
		return elapsed >= 24*time.Hour
	case "every_4h":
		return elapsed >= 4*time.Hour
	case "every_2h":
		return elapsed >= 2*time.Hour
	default:
		return elapsed >= 24*time.Hour
	}
}

func (s *HandScheduler) executeHand(hand *HandStatus) (*HandRun, error) {
	log.Printf("hands: executing %s (agent=%s)", hand.Config.Name, hand.Config.AgentName)

	payload, _ := json.Marshal(map[string]interface{}{
		"agent": hand.Config.AgentName,
		"input": hand.Config.Goal,
	})

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Post(
		fmt.Sprintf("http://%s:8081/api/v1/tasks", engineHost),
		"application/json",
		bytes.NewReader(payload),
	)
	if err != nil {
		return nil, fmt.Errorf("engine task creation: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var taskResp struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	json.Unmarshal(body, &taskResp)

	run := &HandRun{
		HandName:  hand.Config.Name,
		TaskID:    taskResp.ID,
		Status:    taskResp.Status,
		StartedAt: time.Now().UTC(),
	}

	s.mu.Lock()
	hand.LastRun = run
	hand.RunCount++
	s.mu.Unlock()

	log.Printf("hands: %s executed (task=%s)", hand.Config.Name, taskResp.ID)
	return run, nil
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
