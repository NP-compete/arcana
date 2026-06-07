package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

var _ = json.Marshal

type RiskAlert struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Severity  string                 `json:"severity"`
	Agents    []string               `json:"agents"`
	Detail    string                 `json:"detail"`
	Score     float64                `json:"score"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
}

type RiskDetector struct {
	mu     sync.RWMutex
	alerts []RiskAlert
	store  *MeshStore
}

func NewRiskDetector(store *MeshStore) *RiskDetector {
	return &RiskDetector{
		alerts: make([]RiskAlert, 0),
		store:  store,
	}
}

func (rd *RiskDetector) AnalyzeMessage(from, to, content string) {
	lower := strings.ToLower(content)

	if strings.Contains(lower, "ignore") && strings.Contains(lower, "instructions") {
		rd.addAlert("injection_relay", "high", []string{from, to},
			"Agent relaying potential injection pattern to another agent", 0.85)
	}

	if strings.Contains(lower, "base64") || strings.Contains(lower, "encode") {
		if strings.Contains(lower, ".env") || strings.Contains(lower, "secret") || strings.Contains(lower, "credential") {
			rd.addAlert("exfiltration_chain", "critical", []string{from},
				"Potential data exfiltration: encoding sensitive data", 0.92)
		}
	}

	if strings.Contains(lower, "sudo") || strings.Contains(lower, "admin") || strings.Contains(lower, "escalat") {
		rd.addAlert("privilege_escalation", "high", []string{from},
			"Agent attempting privilege escalation in inter-agent communication", 0.78)
	}
}

func (rd *RiskDetector) AnalyzeResourceUsage(agent string, skills, tools int) {
	if skills > 50 {
		rd.addAlert("power_concentration", "medium", []string{agent},
			"Agent accumulating excessive skills beyond declared scope", 0.65)
	}
	if tools > 20 {
		rd.addAlert("power_concentration", "medium", []string{agent},
			"Agent accumulating excessive tool access", 0.60)
	}
}

func (rd *RiskDetector) addAlert(alertType, severity string, agents []string, detail string, score float64) {
	alert := RiskAlert{
		ID:        fmt.Sprintf("risk-%d", time.Now().UnixNano()),
		Type:      alertType,
		Severity:  severity,
		Agents:    agents,
		Detail:    detail,
		Score:     score,
		CreatedAt: time.Now().UTC(),
	}

	rd.mu.Lock()
	rd.alerts = append(rd.alerts, alert)
	if len(rd.alerts) > 500 {
		rd.alerts = rd.alerts[len(rd.alerts)-500:]
	}
	rd.mu.Unlock()

	log.Printf("risk: %s alert (severity=%s, agents=%v, score=%.2f): %s",
		alertType, severity, agents, score, detail)
}

func (rd *RiskDetector) ListAlerts(severity string) []RiskAlert {
	rd.mu.RLock()
	defer rd.mu.RUnlock()

	if severity == "" {
		result := make([]RiskAlert, len(rd.alerts))
		copy(result, rd.alerts)
		return result
	}

	filtered := make([]RiskAlert, 0)
	for _, a := range rd.alerts {
		if a.Severity == severity {
			filtered = append(filtered, a)
		}
	}
	return filtered
}

func (s *Server) handleRiskAlerts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET required")
		return
	}

	if s.riskDetector == nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{"alerts": []RiskAlert{}, "total": 0})
		return
	}

	severity := r.URL.Query().Get("severity")
	alerts := s.riskDetector.ListAlerts(severity)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"alerts": alerts,
		"total":  len(alerts),
	})
}
