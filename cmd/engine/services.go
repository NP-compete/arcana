package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

type ServiceClients struct {
	skillsHost string
	wardHost   string
	auditHost  string
	client     *http.Client
}

func NewServiceClients() *ServiceClients {
	skillsHost := os.Getenv("SKILLS_HOST")
	if skillsHost == "" {
		skillsHost = "arcana-skills.arcana.svc.cluster.local"
	}
	wardHost := os.Getenv("WARD_HOST")
	if wardHost == "" {
		wardHost = "arcana-ward.arcana.svc.cluster.local"
	}
	auditHost := os.Getenv("AUDIT_HOST")
	if auditHost == "" {
		auditHost = "arcana-audit.arcana.svc.cluster.local"
	}
	return &ServiceClients{
		skillsHost: skillsHost,
		wardHost:   wardHost,
		auditHost:  auditHost,
		client:     &http.Client{Timeout: 30 * time.Second},
	}
}

type WardCheckResult struct {
	Verdict  string `json:"verdict"`
	Blocked  bool   `json:"blocked"`
	Reason   string `json:"reason,omitempty"`
	Latency  string `json:"latency,omitempty"`
}

func (s *ServiceClients) CheckWard(agentName, input, direction string) (*WardCheckResult, error) {
	payload, _ := json.Marshal(map[string]interface{}{
		"agent_id":  agentName,
		"input":     input,
		"direction": direction,
	})
	resp, err := s.client.Post(
		fmt.Sprintf("http://%s:8086/api/v1/check", s.wardHost),
		"application/json",
		bytes.NewReader(payload),
	)
	if err != nil {
		log.Printf("ward check failed: %v", err)
		return &WardCheckResult{Verdict: "ALLOW", Blocked: false}, nil
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var result WardCheckResult
	if err := json.Unmarshal(body, &result); err != nil {
		return &WardCheckResult{Verdict: "ALLOW", Blocked: false}, nil
	}
	return &result, nil
}

func (s *ServiceClients) InvokeSkill(skillName string, input map[string]interface{}) (string, error) {
	payload, _ := json.Marshal(map[string]interface{}{
		"skill": skillName,
		"input": input,
	})
	resp, err := s.client.Post(
		fmt.Sprintf("http://%s:8085/api/v1/skills/%s/memory", s.skillsHost, skillName),
		"application/json",
		bytes.NewReader(payload),
	)
	if err != nil {
		return "", fmt.Errorf("skill invocation failed: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return string(body), nil
}

func (s *ServiceClients) AppendAudit(agent, action, input, output, verdict string) {
	payload, _ := json.Marshal(map[string]interface{}{
		"agent":   agent,
		"action":  action,
		"input":   input,
		"output":  output,
		"verdict": verdict,
	})
	resp, err := s.client.Post(
		fmt.Sprintf("http://%s:8100/api/v1/audit", s.auditHost),
		"application/json",
		bytes.NewReader(payload),
	)
	if err != nil {
		log.Printf("audit append failed: %v", err)
		return
	}
	defer resp.Body.Close()
}
