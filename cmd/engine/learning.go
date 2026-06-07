package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

var _ = json.Marshal

type LearningChannel string

const (
	ChannelInContext      LearningChannel = "in_context"
	ChannelCrystallize   LearningChannel = "crystallization"
	ChannelFailureImmune LearningChannel = "failure_immunization"
	ChannelPeriodicTrain LearningChannel = "periodic_training"
)

type LearningEvent struct {
	ID        string                 `json:"id"`
	Channel   LearningChannel        `json:"channel"`
	AgentName string                 `json:"agent_name"`
	SkillName string                 `json:"skill_name,omitempty"`
	Detail    string                 `json:"detail"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
}

type LearningTracker struct {
	mu     sync.RWMutex
	events []LearningEvent
	counts map[LearningChannel]int
}

func NewLearningTracker() *LearningTracker {
	return &LearningTracker{
		events: make([]LearningEvent, 0),
		counts: map[LearningChannel]int{
			ChannelInContext:      0,
			ChannelCrystallize:   0,
			ChannelFailureImmune: 0,
			ChannelPeriodicTrain: 0,
		},
	}
}

func (lt *LearningTracker) Record(channel LearningChannel, agent, skill, detail string) {
	lt.mu.Lock()
	defer lt.mu.Unlock()

	event := LearningEvent{
		ID:        fmt.Sprintf("learn-%d", time.Now().UnixNano()),
		Channel:   channel,
		AgentName: agent,
		SkillName: skill,
		Detail:    detail,
		CreatedAt: time.Now().UTC(),
	}
	lt.events = append(lt.events, event)
	lt.counts[channel]++

	if len(lt.events) > 1000 {
		lt.events = lt.events[len(lt.events)-1000:]
	}

	log.Printf("learning: [%s] agent=%s skill=%s: %s", channel, agent, skill, detail)
}

func (lt *LearningTracker) Stats() map[string]interface{} {
	lt.mu.RLock()
	defer lt.mu.RUnlock()
	return map[string]interface{}{
		"total_events":         len(lt.events),
		"by_channel":           lt.counts,
		"in_context_count":     lt.counts[ChannelInContext],
		"crystallization_count": lt.counts[ChannelCrystallize],
		"failure_immune_count": lt.counts[ChannelFailureImmune],
		"periodic_train_count": lt.counts[ChannelPeriodicTrain],
	}
}

func (lt *LearningTracker) RecentEvents(limit int) []LearningEvent {
	lt.mu.RLock()
	defer lt.mu.RUnlock()
	if limit <= 0 || limit > len(lt.events) {
		limit = len(lt.events)
	}
	start := len(lt.events) - limit
	result := make([]LearningEvent, limit)
	copy(result, lt.events[start:])
	return result
}

func handleLearningStats(tracker *LearningTracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "GET required")
			return
		}
		stats := tracker.Stats()
		events := tracker.RecentEvents(20)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"stats":  stats,
			"recent": events,
		})
	}
}
