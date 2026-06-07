package main

import (
	"log"
	"time"
)

type HealthMonitor struct {
	store        *MeshStore
	k8s          *K8sClient
	interval     time.Duration
	prevRestarts map[string]int
	prevHealthy  map[string]bool
}

func NewHealthMonitor(store *MeshStore, k8s *K8sClient) *HealthMonitor {
	return &HealthMonitor{
		store:        store,
		k8s:          k8s,
		interval:     30 * time.Second,
		prevRestarts: make(map[string]int),
		prevHealthy:  make(map[string]bool),
	}
}

func (m *HealthMonitor) Start() {
	if m.k8s == nil {
		log.Printf("health monitor: k8s client not available, skipping")
		return
	}
	go m.loop()
}

func (m *HealthMonitor) loop() {
	log.Printf("health monitor: starting (interval=%s)", m.interval)
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	m.check()
	for range ticker.C {
		m.check()
	}
}

func (m *HealthMonitor) check() {
	tenant := "default"
	agents := m.store.ListAgents(tenant)
	for _, agent := range agents {
		if agent.AgentType != AgentTypeDeep {
			continue
		}
		if agent.Status == AgentStatusSuspended {
			continue
		}
		m.checkAgent(tenant, &agent)
	}
}

func (m *HealthMonitor) checkAgent(tenant string, agent *MeshAgent) {
	ns := AgentNamespace(agent.Name)
	pods, err := m.k8s.ListPods(ns, "app=deep-agent")
	if err != nil {
		log.Printf("health monitor: list pods for %s: %v", agent.Name, err)
		return
	}

	health := ExtractPodHealth(pods)
	key := tenant + "/" + agent.Name
	isHealthy := health.Phase == "Running" && health.Ready && health.FailureReason == ""

	prevRestart, exists := m.prevRestarts[key]
	wasHealthy := m.prevHealthy[key]

	// Get deployment for replica info
	var desiredReplicas, readyReplicas int
	dep, _ := m.k8s.GetDeploymentStatus(ns, "deep-agent")
	if dep != nil {
		if spec, ok := dep["spec"].(map[string]interface{}); ok {
			if r, ok := spec["replicas"].(float64); ok {
				desiredReplicas = int(r)
			}
		}
		if status, ok := dep["status"].(map[string]interface{}); ok {
			if rr, ok := status["readyReplicas"].(float64); ok {
				readyReplicas = int(rr)
			}
		}
	}

	// Record restart event separately so it's never silently dropped
	restarted := exists && health.RestartCount > prevRestart
	if restarted {
		m.store.RecordHealthEvent(tenant, agent.Name, "restart",
			health.RestartCount, readyReplicas, desiredReplicas,
			health.FailureReason, health.Phase)
	}

	// Determine primary event type
	eventType := "healthy"
	if !isHealthy {
		if health.FailureReason != "" {
			eventType = "failure"
		} else {
			eventType = "unhealthy"
		}
	} else if exists && !wasHealthy {
		eventType = "recovered"
	}

	// Record non-restart events on state changes only
	if eventType != "healthy" || !exists {
		if !(restarted && eventType == "restart") {
			m.store.RecordHealthEvent(tenant, agent.Name, eventType,
				health.RestartCount, readyReplicas, desiredReplicas,
				health.FailureReason, health.Phase)
		}
	}

	m.store.UpdateAgentHealth(tenant, agent.Name,
		health.RestartCount, health.Phase, health.FailureReason, isHealthy)

	if !isHealthy && agent.Status == AgentStatusActive {
		m.store.UpdateAgentStatus(tenant, agent.Name, AgentStatusOffline)
		log.Printf("health monitor: agent %s marked offline (phase=%s, reason=%s)",
			agent.Name, health.Phase, health.FailureReason)
	}
	if isHealthy && agent.Status == AgentStatusOffline {
		m.store.UpdateAgentStatus(tenant, agent.Name, AgentStatusActive)
		log.Printf("health monitor: agent %s recovered to active", agent.Name)
	}

	m.prevRestarts[key] = health.RestartCount
	m.prevHealthy[key] = isHealthy
}
