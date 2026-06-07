package security

import (
	"sync"
)

type TaintLabel string

const (
	TaintSecret     TaintLabel = "secret"
	TaintPII        TaintLabel = "pii"
	TaintCredential TaintLabel = "credential"
	TaintInternal   TaintLabel = "internal"
)

type TaintTracker struct {
	mu     sync.RWMutex
	labels map[string]map[TaintLabel]bool
}

func NewTaintTracker() *TaintTracker {
	return &TaintTracker{
		labels: make(map[string]map[TaintLabel]bool),
	}
}

func (tt *TaintTracker) MarkTainted(dataID string, label TaintLabel) {
	tt.mu.Lock()
	defer tt.mu.Unlock()
	if tt.labels[dataID] == nil {
		tt.labels[dataID] = make(map[TaintLabel]bool)
	}
	tt.labels[dataID][label] = true
}

func (tt *TaintTracker) IsTainted(dataID string) bool {
	tt.mu.RLock()
	defer tt.mu.RUnlock()
	return len(tt.labels[dataID]) > 0
}

func (tt *TaintTracker) GetLabels(dataID string) []TaintLabel {
	tt.mu.RLock()
	defer tt.mu.RUnlock()
	labels := make([]TaintLabel, 0)
	for l := range tt.labels[dataID] {
		labels = append(labels, l)
	}
	return labels
}

func (tt *TaintTracker) Propagate(sourceID, targetID string) {
	tt.mu.Lock()
	defer tt.mu.Unlock()
	if tt.labels[sourceID] == nil {
		return
	}
	if tt.labels[targetID] == nil {
		tt.labels[targetID] = make(map[TaintLabel]bool)
	}
	for l := range tt.labels[sourceID] {
		tt.labels[targetID][l] = true
	}
}

func (tt *TaintTracker) CheckSink(dataID string, allowedLabels []TaintLabel) bool {
	tt.mu.RLock()
	defer tt.mu.RUnlock()
	labels := tt.labels[dataID]
	if labels == nil {
		return true
	}
	allowed := make(map[TaintLabel]bool)
	for _, l := range allowedLabels {
		allowed[l] = true
	}
	for l := range labels {
		if !allowed[l] {
			return false
		}
	}
	return true
}

func (tt *TaintTracker) Zeroize(dataID string) {
	tt.mu.Lock()
	defer tt.mu.Unlock()
	delete(tt.labels, dataID)
}

func (tt *TaintTracker) Stats() map[string]interface{} {
	tt.mu.RLock()
	defer tt.mu.RUnlock()
	labelCounts := map[TaintLabel]int{}
	for _, labels := range tt.labels {
		for l := range labels {
			labelCounts[l]++
		}
	}
	return map[string]interface{}{
		"tracked_items": len(tt.labels),
		"by_label":      labelCounts,
	}
}
