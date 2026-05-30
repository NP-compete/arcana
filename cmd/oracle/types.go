package main

import "time"

type Prediction struct {
	ID              string                 `json:"id"`
	Tool            string                 `json:"tool"`
	Params          map[string]interface{} `json:"params"`
	Context         map[string]interface{} `json:"context,omitempty"`
	PredictedOutput map[string]interface{} `json:"predicted_output"`
	Confidence      float64                `json:"confidence"`
	ActualOutput    map[string]interface{} `json:"actual_output,omitempty"`
	WasCorrect      *bool                  `json:"was_correct,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	CalibratedAt    *time.Time             `json:"calibrated_at,omitempty"`
}

type PredictRequest struct {
	ToolName string                 `json:"tool_name"`
	Params   map[string]interface{} `json:"params"`
	Context  map[string]interface{} `json:"context,omitempty"`
}

type CalibrateRequest struct {
	PredictionID string                 `json:"prediction_id"`
	ActualOutput map[string]interface{} `json:"actual_output"`
}

type StatsResponse struct {
	TotalPredictions   int     `json:"total_predictions"`
	CalibratedCount    int     `json:"calibrated_count"`
	CorrectCount       int     `json:"correct_count"`
	Accuracy           float64 `json:"accuracy"`
	AverageConfidence  float64 `json:"average_confidence"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
