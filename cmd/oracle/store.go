package main

import (
	"encoding/json"
	"math"
	"sync"
	"time"

	"github.com/google/uuid"
)

type PredictionStore struct {
	mu          sync.RWMutex
	predictions map[string]*Prediction
	toolStats   map[string]*toolCalibration
}

type toolCalibration struct {
	total     int
	correct   int
	confidence float64
}

func NewPredictionStore() *PredictionStore {
	return &PredictionStore{
		predictions: make(map[string]*Prediction),
		toolStats:   make(map[string]*toolCalibration),
	}
}

func (s *PredictionStore) Create(tool string, params, context map[string]interface{}, predicted map[string]interface{}, confidence float64) *Prediction {
	now := time.Now().UTC()
	p := &Prediction{
		ID:              uuid.New().String(),
		Tool:            tool,
		Params:          params,
		Context:         context,
		PredictedOutput: predicted,
		Confidence:      confidence,
		CreatedAt:       now,
	}
	if p.Params == nil {
		p.Params = make(map[string]interface{})
	}
	if p.Context == nil {
		p.Context = make(map[string]interface{})
	}

	s.mu.Lock()
	s.predictions[p.ID] = p
	s.mu.Unlock()
	return p
}

func (s *PredictionStore) Get(id string) (*Prediction, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.predictions[id]
	if !ok {
		return nil, false
	}
	copy := *p
	return &copy, true
}

func (s *PredictionStore) Calibrate(id string, actual map[string]interface{}) (*Prediction, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	p, ok := s.predictions[id]
	if !ok {
		return nil, false
	}

	correct := outputsMatch(p.PredictedOutput, actual)
	now := time.Now().UTC()
	p.ActualOutput = actual
	p.WasCorrect = &correct
	p.CalibratedAt = &now

	stats, ok := s.toolStats[p.Tool]
	if !ok {
		stats = &toolCalibration{}
		s.toolStats[p.Tool] = stats
	}
	stats.total++
	stats.confidence += p.Confidence
	if correct {
		stats.correct++
	}

	copy := *p
	return &copy, true
}

func (s *PredictionStore) Stats() StatsResponse {
	s.mu.RLock()
	defer s.mu.RUnlock()

	total := len(s.predictions)
	calibrated := 0
	correct := 0
	confSum := 0.0

	for _, p := range s.predictions {
		confSum += p.Confidence
		if p.WasCorrect != nil {
			calibrated++
			if *p.WasCorrect {
				correct++
			}
		}
	}

	accuracy := 0.0
	if calibrated > 0 {
		accuracy = float64(correct) / float64(calibrated)
	}

	avgConf := 0.0
	if total > 0 {
		avgConf = confSum / float64(total)
	}

	return StatsResponse{
		TotalPredictions:  total,
		CalibratedCount:   calibrated,
		CorrectCount:      correct,
		Accuracy:          math.Round(accuracy*10000) / 10000,
		AverageConfidence: math.Round(avgConf*10000) / 10000,
	}
}

func outputsMatch(predicted, actual map[string]interface{}) bool {
	if len(predicted) == 0 && len(actual) == 0 {
		return true
	}
	pJSON, _ := json.Marshal(predicted)
	aJSON, _ := json.Marshal(actual)
	return string(pJSON) == string(aJSON)
}
