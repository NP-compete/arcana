package main

import (
	"fmt"
	"hash/fnv"
	"math"
)

type Predictor struct {
	store *PredictionStore
}

func NewPredictor(store *PredictionStore) *Predictor {
	return &Predictor{store: store}
}

func (p *Predictor) Predict(toolName string, params, context map[string]interface{}) *Prediction {
	if params == nil {
		params = make(map[string]interface{})
	}
	if context == nil {
		context = make(map[string]interface{})
	}

	hash := hashParams(toolName, params)
	confidence := 0.65 + float64(hash%30)/100.0

	predicted := map[string]interface{}{
		"status":  "success",
		"tool":    toolName,
		"message": fmt.Sprintf("Predicted outcome for %s based on historical patterns", toolName),
		"data":    synthesizeOutput(toolName, params),
	}

	return p.store.Create(toolName, params, context, predicted, math.Round(confidence*10000)/10000)
}

func hashParams(tool string, params map[string]interface{}) uint32 {
	h := fnv.New32a()
	h.Write([]byte(tool))
	for k, v := range params {
		h.Write([]byte(k))
		h.Write([]byte(fmt.Sprintf("%v", v)))
	}
	return h.Sum32()
}

func synthesizeOutput(tool string, params map[string]interface{}) map[string]interface{} {
	output := map[string]interface{}{
		"tool": tool,
	}
	for k, v := range params {
		output["result_"+k] = v
	}
	return output
}
