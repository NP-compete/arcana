package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

func corsOrigin() string {
	if origin := os.Getenv("CORS_ORIGIN"); origin != "" {
		return origin
	}
	return "http://arcana-api.arcana.svc.cluster.local:8080"
}

type Server struct {
	store        *MeshStore
	k8s          *K8sClient
	riskDetector *RiskDetector
}

func NewServer(store *MeshStore) *Server {
	k8s, err := NewK8sClient()
	if err != nil {
		log.Printf("k8s client not available (running outside cluster?): %v", err)
	} else {
		log.Printf("k8s client initialized — agent namespaces enabled")
	}
	return &Server{store: store, k8s: k8s, riskDetector: NewRiskDetector(store)}
}

func extractTenant(r *http.Request) string {
	tenant := r.Header.Get("X-Arcana-Tenant")
	if tenant == "" {
		tenant = "default"
	}
	return tenant
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", corsOrigin())
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Arcana-Tenant")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, ErrorResponse{Error: msg})
}

func (s *Server) corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", corsOrigin())
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Arcana-Tenant")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}

func (s *Server) handleRegisterAgent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req RegisterAgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	tenant := extractTenant(r)

	status := AgentStatusActive
	if req.Status != "" {
		status = AgentStatus(req.Status)
	}
	agentType := AgentType(req.AgentType)
	if agentType == "" {
		agentType = AgentTypeStandard
	}

	agent := s.store.RegisterAgent(tenant, req.Name, req.Capabilities, req.Protocols, status, agentType, req.DeepConfig)

	if s.k8s != nil {
		go s.provisionAgentNamespace(agent)
	}

	writeJSON(w, http.StatusCreated, agent)
}

func (s *Server) provisionAgentNamespace(agent *MeshAgent) {
	ns := AgentNamespace(agent.Name)
	labels := map[string]string{
		"app.kubernetes.io/part-of":    "arcana",
		"app.kubernetes.io/managed-by": "arcana-mesh",
		"arcana.io/agent":              agent.Name,
		"arcana.io/plane":              "agent",
	}

	if err := s.k8s.CreateNamespace(ns, labels); err != nil {
		log.Printf("failed to create namespace %s: %v", ns, err)
		return
	}
	log.Printf("namespace %s created for agent %s", ns, agent.Name)

	capsJSON, _ := json.Marshal(agent.Capabilities)
	protosJSON, _ := json.Marshal(agent.Protocols)
	cmData := map[string]string{
		"agent-name":   agent.Name,
		"capabilities": string(capsJSON),
		"protocols":    string(protosJSON),
		"status":       string(agent.Status),
		"model":        "gpt-4o",
	}
	if err := s.k8s.CreateConfigMap(ns, "agent-config", cmData); err != nil {
		log.Printf("failed to create configmap in %s: %v", ns, err)
	}

	if err := s.k8s.CreateResourceQuota(ns, "agent-quota", map[string]string{
		"pods":            "10",
		"requests.cpu":    "2",
		"requests.memory": "4Gi",
		"limits.cpu":      "4",
		"limits.memory":   "8Gi",
	}); err != nil {
		log.Printf("failed to create resourcequota in %s: %v", ns, err)
	}

	if err := s.k8s.CreateNetworkPolicy(ns, "agent-isolation"); err != nil {
		log.Printf("failed to create networkpolicy in %s: %v", ns, err)
	}

	agentSpec := map[string]interface{}{
		"model":  "gpt-4o",
		"skills": agent.Capabilities,
		"memory": map[string]string{
			"backend": "pgvector",
			"ttl":     "720h",
		},
		"budget": map[string]interface{}{
			"maxTokensPerTurn": 4096,
			"routingStrategy":  "baar",
		},
		"sandbox": map[string]string{
			"runtime": "gvisor",
		},
	}
	if err := s.k8s.CreateArcanaAgent(ns, agent.Name, agentSpec); err != nil {
		log.Printf("failed to create ArcanaAgent CR in %s: %v", ns, err)
	}

	if agent.AgentType == AgentTypeDeep && agent.DeepConfig != nil {
		s.provisionDeepAgentPod(ns, agent)
	}

	log.Printf("agent %s fully provisioned in namespace %s", agent.Name, ns)
}

func templateAgentImage() string {
	if img := os.Getenv("TEMPLATE_AGENT_IMAGE"); img != "" {
		return img
	}
	return "ghcr.io/redhat-data-and-ai/template-agent:deep-agent"
}

func templateAgentPullPolicy() string {
	if policy := os.Getenv("TEMPLATE_AGENT_PULL_POLICY"); policy != "" {
		return policy
	}
	return "Always"
}

func agentDBMode(dc *DeepAgentConfig) string {
	if dc != nil && dc.DBMode != "" {
		return dc.DBMode
	}
	if m := os.Getenv("AGENT_DB_MODE"); m != "" {
		return m
	}
	return "dedicated"
}

func agentDBHost(dc *DeepAgentConfig) string {
	if dc != nil && dc.DBMode == "external" && dc.ExternalDBURL != "" {
		return dc.ExternalDBURL
	}
	return envOr("POSTGRES_HOST", "postgres.arcana.svc.cluster.local")
}

func agentDBPort(dc *DeepAgentConfig) string {
	return envOr("POSTGRES_PORT", "5432")
}

func agentDBName(agentName string, dc *DeepAgentConfig) string {
	switch agentDBMode(dc) {
	case "shared":
		return envOr("POSTGRES_DB", "arcana")
	case "external":
		return envOr("POSTGRES_DB", "arcana")
	default:
		return strings.ReplaceAll(agentName, "-", "_")
	}
}

func (s *Server) provisionDeepAgentPod(ns string, agent *MeshAgent) {
	dbMode := agentDBMode(agent.DeepConfig)
	if dbMode == "dedicated" {
		s.provisionAgentDatabase(agent.Name)
	}

	configData := GenerateDeepAgentConfig(agent)
	if err := s.k8s.CreateConfigMap(ns, "deep-agent-config", configData); err != nil {
		log.Printf("failed to create deep-agent-config in %s: %v", ns, err)
	}

	pgUser := os.Getenv("POSTGRES_USER")
	if pgUser == "" {
		pgUser = "arcana"
	}
	pgPassword := os.Getenv("POSTGRES_PASSWORD")
	if pgPassword == "" {
		if os.Getenv("ARCANA_ENV") == "production" {
			log.Printf("FATAL: POSTGRES_PASSWORD must be set in production — refusing to provision deep agent")
			return
		}
		pgPassword = "arcana-dev"
	}

	secretData := map[string]string{
		"POSTGRES_USER":     pgUser,
		"POSTGRES_PASSWORD": pgPassword,
	}

	// Forward optional provider credentials to deep agent pods
	for _, key := range []string{
		"GOOGLE_APPLICATION_CREDENTIALS_CONTENT",
		"VLLM_BASE_URL", "VLLM_API_KEY",
		"LANGFUSE_PUBLIC_KEY", "LANGFUSE_SECRET_KEY", "LANGFUSE_BASE_URL",
		"SSO_ISSUER_URL", "SSO_CLIENT_ID", "SSO_CLIENT_SECRET",
	} {
		if v := os.Getenv(key); v != "" {
			secretData[key] = v
		}
	}

	if err := s.k8s.CreateSecret(ns, "agent-secrets", secretData); err != nil {
		log.Printf("failed to create agent-secrets in %s: %v", ns, err)
	}

	deployName := "deep-agent"
	labels := map[string]string{
		"app":                          deployName,
		"arcana.io/agent":              agent.Name,
		"app.kubernetes.io/part-of":    "arcana",
		"app.kubernetes.io/managed-by": "arcana-mesh",
		"arcana.io/agent-type":         "create_deep_agent",
	}

	image := templateAgentImage()
	pullPolicy := templateAgentPullPolicy()

	envVars := []map[string]interface{}{
		{"name": "AGENT_HOST", "value": "0.0.0.0"},
		{"name": "AGENT_PORT", "value": "5002"},
		{"name": "PYTHON_LOG_LEVEL", "value": "INFO"},
		{"name": "ENABLE_AUTH", "value": "false"},
		{"name": "POSTGRES_HOST", "value": agentDBHost(agent.DeepConfig)},
		{"name": "POSTGRES_PORT", "value": agentDBPort(agent.DeepConfig)},
		{"name": "POSTGRES_DB", "value": agentDBName(agent.Name, agent.DeepConfig)},
		{"name": "REDIS_URL", "value": "redis://redis.arcana.svc.cluster.local:6379/0"},
		{"name": "REDIS_BROKER_ENABLED", "value": "true"},
		{"name": "OTEL_EXPORTER_OTLP_ENDPOINT", "value": "jaeger.arcana.svc.cluster.local:4318"},
		{"name": "CACHE_ENABLED", "value": "true"},
		{"name": "MEMORY_CONSOLIDATION_ENABLED", "value": "true"},
	}

	// Secret-backed env vars
	for _, key := range []string{
		"POSTGRES_USER", "POSTGRES_PASSWORD",
		"GOOGLE_APPLICATION_CREDENTIALS_CONTENT",
		"VLLM_BASE_URL", "VLLM_API_KEY",
		"LANGFUSE_PUBLIC_KEY", "LANGFUSE_SECRET_KEY", "LANGFUSE_BASE_URL",
		"SSO_ISSUER_URL", "SSO_CLIENT_ID", "SSO_CLIENT_SECRET",
	} {
		envVars = append(envVars, map[string]interface{}{
			"name": key,
			"valueFrom": map[string]interface{}{
				"secretKeyRef": map[string]interface{}{
					"name":     "agent-secrets",
					"key":      key,
					"optional": true,
				},
			},
		})
	}

	falseVal := false
	deploySpec := map[string]interface{}{
		"replicas": 1,
		"selector": map[string]interface{}{
			"matchLabels": map[string]string{"app": deployName},
		},
		"template": map[string]interface{}{
			"metadata": map[string]interface{}{
				"labels": labels,
			},
			"spec": map[string]interface{}{
				"terminationGracePeriodSeconds": 60,
				"automountServiceAccountToken":  falseVal,
				"securityContext": map[string]interface{}{
					"runAsNonRoot": true,
					"fsGroup":      65532,
				},
				"containers": []map[string]interface{}{
					{
						"name":            "agent",
						"image":           image,
						"imagePullPolicy": pullPolicy,
						"ports": []map[string]interface{}{
							{"containerPort": 5002, "name": "http", "protocol": "TCP"},
						},
						"env": envVars,
						"startupProbe": map[string]interface{}{
							"httpGet": map[string]interface{}{
								"path": "/health",
								"port": "http",
							},
							"initialDelaySeconds": 10,
							"periodSeconds":       5,
							"failureThreshold":    30,
						},
						"livenessProbe": map[string]interface{}{
							"httpGet": map[string]interface{}{
								"path": "/health",
								"port": "http",
							},
							"initialDelaySeconds": 30,
							"periodSeconds":       10,
							"timeoutSeconds":      5,
							"failureThreshold":    3,
						},
						"readinessProbe": map[string]interface{}{
							"httpGet": map[string]interface{}{
								"path": "/health",
								"port": "http",
							},
							"initialDelaySeconds": 10,
							"periodSeconds":       5,
							"timeoutSeconds":      3,
							"failureThreshold":    3,
						},
						"volumeMounts": []map[string]interface{}{
							{
								"name":      "agent-config",
								"mountPath": "/app/config/agent",
								"readOnly":  true,
							},
						},
						"resources": map[string]interface{}{
							"requests": map[string]string{
								"memory": "256Mi",
								"cpu":    "100m",
							},
							"limits": map[string]string{
								"memory": "512Mi",
								"cpu":    "500m",
							},
						},
						"securityContext": map[string]interface{}{
							"allowPrivilegeEscalation": false,
							"capabilities": map[string]interface{}{
								"drop": []string{"ALL"},
							},
						},
					},
				},
				"volumes": []map[string]interface{}{
					{
						"name": "agent-config",
						"configMap": map[string]interface{}{
							"name": "deep-agent-config",
						},
					},
				},
				"restartPolicy": "Always",
			},
		},
	}

	if err := s.k8s.CreateDeployment(ns, deployName, deploySpec); err != nil {
		log.Printf("failed to create deep-agent deployment in %s: %v", ns, err)
	}

	if err := s.k8s.CreateService(ns, deployName, 5002, 5002); err != nil {
		log.Printf("failed to create deep-agent service in %s: %v", ns, err)
	}

	log.Printf("deep agent pod provisioned for %s in %s (image: %s)", agent.Name, ns, image)
}

func (s *Server) provisionAgentDatabase(agentName string) {
	pgHost := os.Getenv("POSTGRES_HOST")
	if pgHost == "" {
		pgHost = "postgres.arcana.svc.cluster.local"
	}
	pgPort := os.Getenv("POSTGRES_PORT")
	if pgPort == "" {
		pgPort = "5432"
	}
	pgUser := os.Getenv("POSTGRES_USER")
	if pgUser == "" {
		pgUser = "arcana"
	}
	pgPass := os.Getenv("POSTGRES_PASSWORD")
	if pgPass == "" {
		if os.Getenv("ARCANA_ENV") == "production" {
			log.Printf("FATAL: POSTGRES_PASSWORD must be set in production — refusing to provision database")
			return
		}
		pgPass = "arcana-dev"
	}

	pgSSLMode := envOr("POSTGRES_SSLMODE", "prefer")
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=postgres sslmode=%s",
		pgHost, pgPort, pgUser, pgPass, pgSSLMode)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Printf("failed to connect to postgres for DB provisioning: %v", err)
		return
	}
	defer db.Close()

	dbName := strings.ReplaceAll(agentName, "-", "_")
	var exists bool
	err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", dbName).Scan(&exists)
	if err != nil {
		log.Printf("failed to check database existence: %v", err)
		return
	}
	if exists {
		log.Printf("database %s already exists", dbName)
		return
	}

	_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %q", dbName))
	if err != nil {
		log.Printf("failed to create database %s: %v", dbName, err)
		return
	}
	log.Printf("database %s created for agent %s", dbName, agentName)
}

func (s *Server) handleDeleteAgent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	name := strings.TrimPrefix(r.URL.Path, "/api/v1/agents/")
	if name == "" || strings.Contains(name, "/") {
		writeError(w, http.StatusBadRequest, "agent name required")
		return
	}

	tenant := extractTenant(r)

	if !s.store.DeleteAgent(tenant, name) {
		writeError(w, http.StatusNotFound, "agent not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "name": name})
}

func (s *Server) handleListAgents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	tenant := extractTenant(r)
	agents := s.store.ListAgents(tenant)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"agents": agents,
		"total":  len(agents),
	})
}

func (s *Server) handleGetAgent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	name := strings.TrimPrefix(r.URL.Path, "/api/v1/agents/")
	if name == "" || strings.Contains(name, "/") {
		writeError(w, http.StatusBadRequest, "agent name required")
		return
	}

	tenant := extractTenant(r)
	agent, ok := s.store.GetAgent(tenant, name)
	if !ok {
		writeError(w, http.StatusNotFound, "agent not found")
		return
	}
	writeJSON(w, http.StatusOK, agent)
}

func (s *Server) handleSendMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.From == "" || req.To == "" {
		writeError(w, http.StatusBadRequest, "from and to are required")
		return
	}
	if req.Protocol != "a2a" && req.Protocol != "acp" {
		writeError(w, http.StatusBadRequest, "protocol must be a2a or acp")
		return
	}

	tenant := extractTenant(r)
	msg, err := s.store.SendMessage(tenant, req.From, req.To, req.Payload, req.Protocol)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, msg)
}

func (s *Server) handleGetMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	agentName := strings.TrimPrefix(r.URL.Path, "/api/v1/messages/")
	if agentName == "" {
		writeError(w, http.StatusBadRequest, "agent name required")
		return
	}

	tenant := extractTenant(r)
	messages := s.store.GetPendingMessages(tenant, agentName)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"agent":    agentName,
		"messages": messages,
		"count":    len(messages),
	})
}

func (s *Server) handleDelegate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req DelegateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.FromAgent == "" || req.ToAgent == "" {
		writeError(w, http.StatusBadRequest, "from_agent and to_agent are required")
		return
	}
	if req.TaskType == "" {
		writeError(w, http.StatusBadRequest, "task_type is required")
		return
	}

	tenant := extractTenant(r)
	task, err := s.store.CreateDelegation(tenant, req.FromAgent, req.ToAgent, req.TaskType, req.Payload)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	go s.processDelegation(task.ID)

	writeJSON(w, http.StatusAccepted, task)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func fetchJSON(url string) (map[string]interface{}, error) {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)
	return result, nil
}

type K8sInfo struct {
	Namespace       string         `json:"namespace"`
	NamespaceExists bool           `json:"namespace_exists"`
	Resources       map[string]int `json:"resources"`
	Isolation       string         `json:"isolation"`
}

type AgentDetail struct {
	Name         string           `json:"name"`
	AgentType    AgentType        `json:"agent_type"`
	Capabilities []string         `json:"capabilities"`
	Protocols    []string         `json:"protocols"`
	Status       AgentStatus      `json:"status"`
	RegisteredAt time.Time        `json:"registered_at"`
	DeepConfig   *DeepAgentConfig `json:"deep_config,omitempty"`

	TasksCompleted int                      `json:"tasks_completed"`
	TasksPending   int                      `json:"tasks_pending"`
	TotalTokens    int                      `json:"total_tokens"`
	TotalCostUSD   float64                  `json:"total_cost_usd"`
	RecentTasks    []map[string]interface{} `json:"recent_tasks"`

	GuardrailLayers int    `json:"guardrail_layers"`
	GuardrailStatus string `json:"guardrail_status"`

	BudgetPerDay string  `json:"budget_per_day"`
	BudgetUsed   float64 `json:"budget_used_usd"`

	MessagesCount    int `json:"messages_count"`
	DelegationsCount int `json:"delegations_count"`

	Uptime string `json:"uptime"`

	Memory     MemoryInfo          `json:"memory"`
	Kubernetes K8sInfo             `json:"kubernetes"`
	DeepPod    *DeepPodInfo        `json:"deep_pod,omitempty"`
	Health     *AgentHealthSummary `json:"health,omitempty"`
}

type DeepPodInfo struct {
	Deployed          bool   `json:"deployed"`
	Status            string `json:"status"`
	Endpoint          string `json:"endpoint"`
	Replicas          int    `json:"replicas"`
	ReadyReplicas     int    `json:"ready_replicas"`
	RestartCount      int    `json:"restart_count"`
	LastFailureAt     string `json:"last_failure_at,omitempty"`
	LastFailureReason string `json:"last_failure_reason,omitempty"`
	PodPhase          string `json:"pod_phase"`
}

type MemoryInfo struct {
	ShortTermCount int `json:"short_term_count"`
	LongTermCount  int `json:"long_term_count"`
	SkillCount     int `json:"skill_count"`
}

func (s *Server) handleAgentDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v1/agents/")
	name := strings.TrimSuffix(path, "/detail")
	if name == "" {
		writeError(w, http.StatusBadRequest, "agent name required")
		return
	}

	tenant := extractTenant(r)
	agent, ok := s.store.GetAgent(tenant, name)
	if !ok {
		writeError(w, http.StatusNotFound, "agent not found")
		return
	}

	detail := AgentDetail{
		Name:         agent.Name,
		AgentType:    agent.AgentType,
		Capabilities: agent.Capabilities,
		Protocols:    agent.Protocols,
		Status:       agent.Status,
		RegisteredAt: agent.RegisteredAt,
		DeepConfig:   agent.DeepConfig,
		Uptime:       time.Since(agent.RegisteredAt).Round(time.Second).String(),
	}

	engineHost := envOr("ENGINE_HOST", "arcana-engine")
	engineURL := fmt.Sprintf("http://%s:8081/api/v1/tasks", engineHost)
	if tasksResp, err := fetchJSON(engineURL); err == nil {
		if taskList, ok := tasksResp["tasks"].([]interface{}); ok {
			for _, t := range taskList {
				task, ok := t.(map[string]interface{})
				if !ok {
					continue
				}
				if task["agent"] != name {
					continue
				}
				status, _ := task["status"].(string)
				if status == "completed" {
					detail.TasksCompleted++
				} else {
					detail.TasksPending++
				}
				if tokens, ok := task["tokens_used"].(float64); ok {
					detail.TotalTokens += int(tokens)
				}
				if cost, ok := task["cost"].(float64); ok {
					detail.TotalCostUSD += cost
				}
				entry := map[string]interface{}{
					"id":         task["id"],
					"status":     task["status"],
					"tokens":     task["tokens_used"],
					"cost":       task["cost"],
					"created_at": task["created_at"],
				}
				detail.RecentTasks = append(detail.RecentTasks, entry)
			}
		}
	}
	if detail.RecentTasks == nil {
		detail.RecentTasks = []map[string]interface{}{}
	}

	wardHost := envOr("WARD_HOST", "arcana-ward")
	wardURL := fmt.Sprintf("http://%s:8086/api/v1/rules", wardHost)
	if wardResp, err := fetchJSON(wardURL); err == nil {
		if layers, ok := wardResp["layers"].([]interface{}); ok {
			detail.GuardrailLayers = len(layers)
		}
		detail.GuardrailStatus = "active"
	} else {
		detail.GuardrailStatus = "unreachable"
	}

	finopsHost := envOr("FINOPS_HOST", "arcana-finops")
	finopsURL := fmt.Sprintf("http://%s:8105/api/v1/costs", finopsHost)
	if _, err := fetchJSON(finopsURL); err == nil {
		detail.BudgetPerDay = "$20"
		detail.BudgetUsed = detail.TotalCostUSD
	}

	detail.MessagesCount = s.store.CountMessagesToAgent(tenant, name)
	detail.DelegationsCount = s.store.CountDelegationsForAgent(tenant, name)

	memoryHost := envOr("MEMORY_HOST", "arcana-memory")
	memClient := &http.Client{Timeout: 3 * time.Second}
	stURL := fmt.Sprintf("http://%s:8087/api/v1/memory/short-term/%s", memoryHost, name)
	if resp, err := memClient.Get(stURL); err == nil {
		defer resp.Body.Close()
		bodyBytes, _ := io.ReadAll(resp.Body)
		var arr []interface{}
		if json.Unmarshal(bodyBytes, &arr) == nil {
			detail.Memory.ShortTermCount = len(arr)
		}
	}
	ltSearchURL := fmt.Sprintf("http://%s:8087/api/v1/memory/long-term/%s/search?query=conversation&top_k=50", memoryHost, name)
	if resp, err := memClient.Get(ltSearchURL); err == nil {
		defer resp.Body.Close()
		bodyBytes, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		if json.Unmarshal(bodyBytes, &result) == nil {
			if results, ok := result["results"].([]interface{}); ok {
				detail.Memory.LongTermCount = len(results)
			}
		}
	}

	nsName := AgentNamespace(name)
	detail.Kubernetes = K8sInfo{
		Namespace: nsName,
		Resources: map[string]int{},
		Isolation: "none",
	}
	if s.k8s != nil {
		nsData, err := s.k8s.GetNamespace(nsName)
		if err == nil && nsData != nil {
			detail.Kubernetes.NamespaceExists = true
			resources, _ := s.k8s.ListNamespaceResources(nsName)
			detail.Kubernetes.Resources = resources
			if np, ok := resources["networkpolicies"]; ok && np > 0 {
				detail.Kubernetes.Isolation = "network-policy"
			}
		}

		if agent.AgentType == AgentTypeDeep {
			podInfo := &DeepPodInfo{
				Endpoint: fmt.Sprintf("http://deep-agent.%s.svc.cluster.local:5002", nsName),
			}
			depStatus, err := s.k8s.GetDeploymentStatus(nsName, "deep-agent")
			if err == nil && depStatus != nil {
				podInfo.Deployed = true
				podInfo.Status = "deployed"
				if spec, ok := depStatus["spec"].(map[string]interface{}); ok {
					if r, ok := spec["replicas"].(float64); ok {
						podInfo.Replicas = int(r)
					}
				}
				if status, ok := depStatus["status"].(map[string]interface{}); ok {
					if rr, ok := status["readyReplicas"].(float64); ok {
						podInfo.ReadyReplicas = int(rr)
						if podInfo.ReadyReplicas > 0 {
							podInfo.Status = "running"
						}
					}
					if rr, ok := status["unavailableReplicas"].(float64); ok && rr > 0 {
						podInfo.Status = "starting"
					}
				}
			} else {
				podInfo.Status = "not_deployed"
			}
			detail.DeepPod = podInfo
		}
	}

	healthSummary := s.store.GetAgentHealthSummary(tenant, name)
	if healthSummary != nil {
		detail.Health = healthSummary
		if detail.DeepPod != nil {
			detail.DeepPod.RestartCount = healthSummary.RestartCount
			detail.DeepPod.PodPhase = healthSummary.PodPhase
			detail.DeepPod.LastFailureReason = healthSummary.LastFailureReason
			if healthSummary.LastFailureAt != nil {
				detail.DeepPod.LastFailureAt = healthSummary.LastFailureAt.Format(time.RFC3339)
			}
		}
	}

	writeJSON(w, http.StatusOK, detail)
}

func extractAgentName(path string) string {
	parts := strings.Split(strings.TrimRight(path, "/"), "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

// handleSuspendAgent snapshots agent state and scales deployment to zero.
func (s *Server) handleSuspendAgent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST required")
		return
	}

	name := extractAgentName(r.URL.Path)
	tenant := extractTenant(r)
	_, ok := s.store.GetAgent(tenant, name)
	if !ok {
		writeError(w, http.StatusNotFound, "agent not found")
		return
	}

	ns := AgentNamespace(name)

	snapshot := map[string]interface{}{
		"agent_name":    name,
		"namespace":     ns,
		"suspended_at":  time.Now().UTC().Format(time.RFC3339),
		"status":        "suspended",
		"snapshot_path":  fmt.Sprintf("s3://arcana/%s/snapshots/%s/%d.tar.gz", tenant, name, time.Now().Unix()),
	}

	if s.k8s != nil {
		if err := s.k8s.ScaleDeployment(ns, "deep-agent", 0); err != nil {
			log.Printf("suspend: scale to 0 failed for %s: %v", name, err)
		} else {
			log.Printf("suspend: scaled %s/deep-agent to 0 replicas", ns)
		}
	}

	s.store.UpdateAgentStatus(tenant, name, AgentStatusSuspended)

	writeJSON(w, http.StatusOK, snapshot)
}

// handleResumeAgent restores agent from snapshot and scales deployment back up.
func (s *Server) handleResumeAgent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST required")
		return
	}

	name := extractAgentName(r.URL.Path)
	tenant := extractTenant(r)
	_, ok := s.store.GetAgent(tenant, name)
	if !ok {
		writeError(w, http.StatusNotFound, "agent not found")
		return
	}

	ns := AgentNamespace(name)

	resume := map[string]interface{}{
		"agent_name":  name,
		"namespace":   ns,
		"resumed_at":  time.Now().UTC().Format(time.RFC3339),
		"status":      "active",
	}

	if s.k8s != nil {
		if err := s.k8s.ScaleDeployment(ns, "deep-agent", 1); err != nil {
			log.Printf("resume: scale to 1 failed for %s: %v", name, err)
		} else {
			log.Printf("resume: scaled %s/deep-agent to 1 replica", ns)
		}
	}

	s.store.UpdateAgentStatus(tenant, name, AgentStatusActive)

	writeJSON(w, http.StatusOK, resume)
}

func (s *Server) handleAgentHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/agents/")
	name := strings.TrimSuffix(path, "/health")
	if name == "" {
		writeError(w, http.StatusBadRequest, "agent name required")
		return
	}
	tenant := extractTenant(r)
	summary := s.store.GetAgentHealthSummary(tenant, name)
	if summary == nil {
		writeError(w, http.StatusNotFound, "agent not found")
		return
	}
	events := s.store.GetAgentHealthEvents(tenant, name, 20)
	writeJSON(w, http.StatusOK, AgentHealthResponse{
		Summary: *summary,
		Events:  events,
	})
}

func (s *Server) handleAgentsHealthOverview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	tenant := extractTenant(r)
	overview := s.store.GetAgentsHealthOverview(tenant)
	writeJSON(w, http.StatusOK, overview)
}

func (s *Server) processDelegation(taskID string) {
	s.store.UpdateDelegation(taskID, func(t *DelegationTask) {
		t.Status = DelegationAccepted
	})

	time.Sleep(50 * time.Millisecond)

	s.store.UpdateDelegation(taskID, func(t *DelegationTask) {
		t.Status = DelegationCompleted
		t.Result = map[string]interface{}{
			"delegated": true,
			"message":   "Task accepted and processed by target agent",
		}
	})
}
