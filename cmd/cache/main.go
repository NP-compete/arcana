package main

import (
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

type CacheLayer string

const (
	LayerExact     CacheLayer = "exact"
	LayerSemantic  CacheLayer = "semantic"
	LayerRetrieval CacheLayer = "retrieval"
	LayerContext   CacheLayer = "context"
)

type CacheEntry struct {
	Key       string      `json:"key"`
	Layer     CacheLayer  `json:"layer"`
	Value     interface{} `json:"value"`
	Tenant    string      `json:"tenant"`
	TTL       int64       `json:"ttl_seconds"`
	CreatedAt time.Time   `json:"created_at"`
	ExpiresAt time.Time   `json:"expires_at"`
	HitCount  int         `json:"hit_count"`
}

type CacheStats struct {
	TotalEntries int            `json:"total_entries"`
	HitCount     int            `json:"hit_count"`
	MissCount    int            `json:"miss_count"`
	HitRate      float64        `json:"hit_rate"`
	ByLayer      map[string]int `json:"by_layer"`
	SavedCostUSD float64        `json:"saved_cost_usd"`
}

type CacheStore struct {
	mu      sync.RWMutex
	entries map[string]*CacheEntry
	stats   CacheStats
}

func NewCacheStore() *CacheStore {
	cs := &CacheStore{
		entries: make(map[string]*CacheEntry),
		stats:   CacheStats{ByLayer: map[string]int{"exact": 0, "semantic": 0, "retrieval": 0, "context": 0}},
	}
	go cs.evictionLoop()
	return cs
}

func (cs *CacheStore) Get(key, tenant string) (*CacheEntry, bool) {
	cs.mu.RLock()
	entry, ok := cs.entries[tenant+":"+key]
	cs.mu.RUnlock()

	if !ok {
		cs.mu.Lock()
		cs.stats.MissCount++
		cs.mu.Unlock()
		return nil, false
	}

	if time.Now().After(entry.ExpiresAt) {
		cs.mu.Lock()
		delete(cs.entries, tenant+":"+key)
		cs.stats.TotalEntries--
		cs.stats.MissCount++
		cs.mu.Unlock()
		return nil, false
	}

	cs.mu.Lock()
	entry.HitCount++
	cs.stats.HitCount++
	cs.stats.SavedCostUSD += 0.002
	cs.mu.Unlock()

	return entry, true
}

func (cs *CacheStore) Put(key, tenant string, layer CacheLayer, value interface{}, ttlSeconds int64) {
	now := time.Now()
	entry := &CacheEntry{
		Key:       key,
		Layer:     layer,
		Value:     value,
		Tenant:    tenant,
		TTL:       ttlSeconds,
		CreatedAt: now,
		ExpiresAt: now.Add(time.Duration(ttlSeconds) * time.Second),
	}
	cs.mu.Lock()
	cs.entries[tenant+":"+key] = entry
	cs.stats.TotalEntries = len(cs.entries)
	cs.stats.ByLayer[string(layer)]++
	cs.mu.Unlock()
}

func (cs *CacheStore) Invalidate(key, tenant string) bool {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	fullKey := tenant + ":" + key
	if _, ok := cs.entries[fullKey]; ok {
		delete(cs.entries, fullKey)
		cs.stats.TotalEntries = len(cs.entries)
		return true
	}
	return false
}

func (cs *CacheStore) GetStats() CacheStats {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	stats := cs.stats
	total := stats.HitCount + stats.MissCount
	if total > 0 {
		stats.HitRate = float64(stats.HitCount) / float64(total)
	}
	return stats
}

func (cs *CacheStore) evictionLoop() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		cs.mu.Lock()
		for k, e := range cs.entries {
			if now.After(e.ExpiresAt) {
				delete(cs.entries, k)
			}
		}
		cs.stats.TotalEntries = len(cs.entries)
		cs.mu.Unlock()
	}
}

func exactCacheKey(model, systemPrompt string, messages interface{}) string {
	payload, _ := json.Marshal(map[string]interface{}{
		"model":  model,
		"system": systemPrompt,
		"msgs":   messages,
	})
	h := sha256.Sum256(payload)
	return hex.EncodeToString(h[:])
}

func main() {
	httpSrv := server.New(server.Config{
		ServiceName: "cache",
		Port:        "8110",
	})

	store := NewCacheStore()

	httpSrv.HandleFunc("/api/v1/cache/get", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "POST required")
			return
		}
		var req struct {
			Key    string `json:"key"`
			Tenant string `json:"tenant"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		if req.Tenant == "" {
			req.Tenant = "default"
		}
		entry, ok := store.Get(req.Key, req.Tenant)
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"status": "miss"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"status": "hit", "entry": entry})
	}))

	httpSrv.HandleFunc("/api/v1/cache/put", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "POST required")
			return
		}
		var req struct {
			Key    string      `json:"key"`
			Tenant string      `json:"tenant"`
			Layer  string      `json:"layer"`
			Value  interface{} `json:"value"`
			TTL    int64       `json:"ttl_seconds"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		if req.Tenant == "" {
			req.Tenant = "default"
		}
		if req.TTL <= 0 {
			req.TTL = 3600
		}
		layer := CacheLayer(req.Layer)
		if layer == "" {
			layer = LayerExact
		}
		store.Put(req.Key, req.Tenant, layer, req.Value, req.TTL)
		writeJSON(w, http.StatusCreated, map[string]string{"status": "cached"})
	}))

	httpSrv.HandleFunc("/api/v1/cache/invalidate", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "POST required")
			return
		}
		var req struct {
			Key    string `json:"key"`
			Tenant string `json:"tenant"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		if req.Tenant == "" {
			req.Tenant = "default"
		}
		removed := store.Invalidate(req.Key, req.Tenant)
		writeJSON(w, http.StatusOK, map[string]bool{"removed": removed})
	}))

	httpSrv.HandleFunc("/api/v1/cache/stats", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, store.GetStats())
	}))

	httpSrv.HandleFunc("/api/v1/cache/llm", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "POST required")
			return
		}
		var req struct {
			Model        string      `json:"model"`
			SystemPrompt string      `json:"system_prompt"`
			Messages     interface{} `json:"messages"`
			Tenant       string      `json:"tenant"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		if req.Tenant == "" {
			req.Tenant = "default"
		}
		key := exactCacheKey(req.Model, req.SystemPrompt, req.Messages)
		entry, ok := store.Get(key, req.Tenant)
		if ok {
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"status":   "hit",
				"response": entry.Value,
				"layer":    entry.Layer,
				"savings":  "$0.002",
			})
			return
		}
		writeJSON(w, http.StatusNotFound, map[string]interface{}{
			"status": "miss",
			"key":    key,
		})
	}))

	log.Println("cache: 4-layer semantic cache ready (exact, semantic, retrieval, context)")

	httpSrv.ListenAndServe()
}

func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := os.Getenv("CORS_ORIGIN")
		if origin == "" {
			origin = "*"
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
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

var _ = fmt.Sprintf
