package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

type AuditStore struct {
	mu       sync.RWMutex
	entries  []AuditEntry
	index    map[string]int
	lastHash string
}

func NewAuditStore() *AuditStore {
	return &AuditStore{
		entries:  make([]AuditEntry, 0),
		index:    make(map[string]int),
		lastHash: "genesis",
	}
}

func (s *AuditStore) computeHash(entry AuditEntry) string {
	data, _ := json.Marshal(struct {
		ID          string    `json:"id"`
		Timestamp   time.Time `json:"timestamp"`
		Tenant      string    `json:"tenant"`
		Agent       string    `json:"agent"`
		Action      string    `json:"action"`
		Tool        string    `json:"tool"`
		InputHash   string    `json:"input_hash"`
		OutputHash  string    `json:"output_hash"`
		WardVerdict string    `json:"ward_verdict"`
		OPAVerdict  string    `json:"opa_verdict"`
		CostUSD     float64   `json:"cost_usd"`
		Tokens      int64     `json:"tokens"`
		Model       string    `json:"model"`
		SessionID   string    `json:"session_id"`
		WorkflowID  string    `json:"workflow_id"`
		PrevHash    string    `json:"prev_hash"`
	}{entry.ID, entry.Timestamp, entry.Tenant, entry.Agent, entry.Action, entry.Tool,
		entry.InputHash, entry.OutputHash, entry.WardVerdict, entry.OPAVerdict,
		entry.CostUSD, entry.Tokens, entry.Model, entry.SessionID, entry.WorkflowID, entry.PrevHash})
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func (s *AuditStore) Append(req AppendAuditRequest) AuditEntry {
	s.mu.Lock()
	defer s.mu.Unlock()

	ts := req.Timestamp
	if ts.IsZero() {
		ts = time.Now().UTC()
	}

	entry := AuditEntry{
		ID:          uuid.New().String(),
		Timestamp:   ts,
		Tenant:      req.Tenant,
		Agent:       req.Agent,
		Action:      req.Action,
		Tool:        req.Tool,
		InputHash:   req.InputHash,
		OutputHash:  req.OutputHash,
		WardVerdict: req.WardVerdict,
		OPAVerdict:  req.OPAVerdict,
		CostUSD:     req.CostUSD,
		Tokens:      req.Tokens,
		Model:       req.Model,
		SessionID:   req.SessionID,
		WorkflowID:  req.WorkflowID,
		PrevHash:    s.lastHash,
	}
	entry.EntryHash = s.computeHash(entry)
	s.lastHash = entry.EntryHash

	s.index[entry.ID] = len(s.entries)
	s.entries = append(s.entries, entry)
	return entry
}

func (s *AuditStore) Get(id string) (*AuditEntry, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	idx, ok := s.index[id]
	if !ok {
		return nil, false
	}
	copy := s.entries[idx]
	return &copy, true
}

func (s *AuditStore) Query(q AuditQuery) ([]AuditEntry, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var since, until time.Time
	if q.Since != "" {
		since, _ = time.Parse(time.RFC3339, q.Since)
	}
	if q.Until != "" {
		until, _ = time.Parse(time.RFC3339, q.Until)
	}

	filtered := make([]AuditEntry, 0)
	for _, e := range s.entries {
		if q.Agent != "" && e.Agent != q.Agent {
			continue
		}
		if q.Action != "" && e.Action != q.Action {
			continue
		}
		if q.Verdict != "" && e.WardVerdict != q.Verdict && e.OPAVerdict != q.Verdict {
			continue
		}
		if !since.IsZero() && e.Timestamp.Before(since) {
			continue
		}
		if !until.IsZero() && e.Timestamp.After(until) {
			continue
		}
		filtered = append(filtered, e)
	}

	total := len(filtered)
	limit := q.Limit
	if limit <= 0 {
		limit = 100
	}
	offset := q.Offset
	if offset < 0 {
		offset = 0
	}
	if offset >= len(filtered) {
		return []AuditEntry{}, total
	}
	end := offset + limit
	if end > len(filtered) {
		end = len(filtered)
	}
	return filtered[offset:end], total
}

func (s *AuditStore) Stats() AuditStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := AuditStats{
		TotalEntries: int64(len(s.entries)),
		ByAgent:      make(map[string]int64),
		ByVerdict:    make(map[string]int64),
	}
	for _, e := range s.entries {
		stats.ByAgent[e.Agent]++
		if e.WardVerdict != "" {
			stats.ByVerdict[e.WardVerdict]++
		}
		stats.TotalCost += e.CostUSD
	}
	return stats
}

func (s *AuditStore) VerifyChain() (bool, string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	prev := "genesis"
	for i, e := range s.entries {
		if e.PrevHash != prev {
			return false, fmt.Sprintf("chain broken at entry %d: expected prev_hash %s, got %s", i, prev, e.PrevHash)
		}
		expected := s.computeHash(e)
		if e.EntryHash != expected {
			return false, fmt.Sprintf("hash mismatch at entry %d", i)
		}
		prev = e.EntryHash
	}
	return true, ""
}
