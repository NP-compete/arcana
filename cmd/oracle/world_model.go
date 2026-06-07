package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"time"
)

type PredictionCache struct {
	mu    sync.RWMutex
	cache map[string]*Prediction
}

type Prediction struct {
	ToolName   string                 `json:"tool_name"`
	Args       map[string]interface{} `json:"args"`
	Result     interface{}            `json:"predicted_result"`
	Confidence float64                `json:"confidence"`
	Latency    float64                `json:"predicted_latency_ms"`
	Cost       float64                `json:"predicted_cost_usd"`
	CachedAt   time.Time              `json:"cached_at"`
	HitCount   int                    `json:"hit_count"`
}

func NewPredictionCache() *PredictionCache {
	return &PredictionCache{
		cache: make(map[string]*Prediction),
	}
}

func (pc *PredictionCache) Predict(toolName string, args map[string]interface{}) (*Prediction, bool) {
	key := predictionKey(toolName, args)
	pc.mu.RLock()
	p, ok := pc.cache[key]
	pc.mu.RUnlock()

	if !ok {
		return nil, false
	}

	age := time.Since(p.CachedAt)
	decayFactor := math.Exp(-age.Hours() / 24.0)
	adjustedConfidence := p.Confidence * decayFactor

	if adjustedConfidence < 0.5 {
		return nil, false
	}

	pc.mu.Lock()
	p.HitCount++
	pc.mu.Unlock()

	result := *p
	result.Confidence = adjustedConfidence
	return &result, true
}

func (pc *PredictionCache) Learn(toolName string, args map[string]interface{}, result interface{}, latencyMs, costUSD float64) {
	key := predictionKey(toolName, args)
	pc.mu.Lock()
	defer pc.mu.Unlock()

	existing, ok := pc.cache[key]
	if ok {
		existing.Result = result
		existing.Latency = latencyMs
		existing.Cost = costUSD
		existing.Confidence = math.Min(existing.Confidence+0.05, 0.99)
		existing.CachedAt = time.Now()
	} else {
		pc.cache[key] = &Prediction{
			ToolName:   toolName,
			Args:       args,
			Result:     result,
			Confidence: 0.6,
			Latency:    latencyMs,
			Cost:       costUSD,
			CachedAt:   time.Now(),
		}
	}
}

func (pc *PredictionCache) Stats() map[string]interface{} {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	totalHits := 0
	avgConfidence := 0.0
	for _, p := range pc.cache {
		totalHits += p.HitCount
		avgConfidence += p.Confidence
	}
	if len(pc.cache) > 0 {
		avgConfidence /= float64(len(pc.cache))
	}

	return map[string]interface{}{
		"cached_predictions": len(pc.cache),
		"total_hits":         totalHits,
		"avg_confidence":     fmt.Sprintf("%.2f", avgConfidence),
	}
}

func predictionKey(toolName string, args map[string]interface{}) string {
	payload, _ := json.Marshal(map[string]interface{}{
		"tool": toolName,
		"args": args,
	})
	h := sha256.Sum256(payload)
	return hex.EncodeToString(h[:8])
}
