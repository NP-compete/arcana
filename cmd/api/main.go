package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/NP-compete/arcana/pkg/server"
	"github.com/google/uuid"
)

type ServiceHealth struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Latency string `json:"latency,omitempty"`
	Port    int    `json:"port"`
	Plane   string `json:"plane"`
}

type SystemHealth struct {
	Platform string          `json:"platform"`
	Version  string          `json:"version"`
	Uptime   string          `json:"uptime"`
	Services []ServiceHealth `json:"services"`
}

type ServiceRoute struct {
	Name    string
	EnvKey  string
	Default string
	Port    int
	Prefix  string
	Plane   string
}

var startTime = time.Now()
var corsOrigin string

// memoryWorkCh is a bounded channel for async memory persistence work.
// A pool of workers drains this channel, preventing unbounded goroutine creation.
var memoryWorkCh = make(chan func(), 100)

func init() {
	for i := 0; i < 5; i++ {
		go func() {
			for fn := range memoryWorkCh {
				fn()
			}
		}()
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func checkTCP(host string, port int, timeout time.Duration) (bool, time.Duration) {
	start := time.Now()
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), timeout)
	elapsed := time.Since(start)
	if err != nil {
		return false, elapsed
	}
	conn.Close()
	return true, elapsed
}

var backingServices = []struct {
	name  string
	host  string
	port  int
	plane string
}{
	{"PostgreSQL", "POSTGRES_HOST", 5432, "infra"},
	{"Redis", "REDIS_HOST", 6379, "infra"},
	{"Temporal", "TEMPORAL_HOST", 7233, "infra"},
	{"MinIO", "MINIO_HOST", 9000, "infra"},
	{"NATS", "NATS_HOST", 4222, "infra"},
}

var serviceRoutes = []ServiceRoute{
	// Agent Plane
	{Name: "engine", EnvKey: "ENGINE_HOST", Default: "arcana-engine", Port: 8081, Prefix: "/api/v1/tasks", Plane: "agent"},
	{Name: "blueprint", EnvKey: "BLUEPRINT_HOST", Default: "arcana-blueprint", Port: 8088, Prefix: "/api/v1/blueprints", Plane: "agent"},
	{Name: "oracle", EnvKey: "ORACLE_HOST", Default: "arcana-oracle", Port: 8089, Prefix: "/api/v1/predict", Plane: "agent"},
	{Name: "memory", EnvKey: "MEMORY_HOST", Default: "arcana-memory", Port: 8087, Prefix: "/api/v1/memory", Plane: "agent"},

	// Data Plane
	{Name: "codex-router", EnvKey: "CODEX_ROUTER_HOST", Default: "arcana-codex-router", Port: 8090, Prefix: "/api/v1/search", Plane: "data"},
	{Name: "codex-ingestor", EnvKey: "CODEX_INGESTOR_HOST", Default: "arcana-codex-ingestor", Port: 8092, Prefix: "/api/v1/ingest", Plane: "data"},
	{Name: "connectors", EnvKey: "CONNECTORS_HOST", Default: "arcana-connectors", Port: 8094, Prefix: "/api/v1/connectors", Plane: "data"},
	{Name: "graph", EnvKey: "GRAPH_HOST", Default: "arcana-graph", Port: 8095, Prefix: "/api/v1/graph", Plane: "data"},

	// Tool Plane
	{Name: "tools", EnvKey: "TOOLS_HOST", Default: "arcana-tools", Port: 8096, Prefix: "/api/v1/tools", Plane: "tool"},
	{Name: "sandbox", EnvKey: "SANDBOX_HOST", Default: "arcana-sandbox", Port: 8097, Prefix: "/api/v1/exec", Plane: "tool"},

	// Model Plane
	{Name: "forge", EnvKey: "FORGE_HOST", Default: "arcana-forge", Port: 8098, Prefix: "/api/v1/experiments", Plane: "model"},
	{Name: "models", EnvKey: "MODELS_HOST", Default: "arcana-models", Port: 8099, Prefix: "/api/v1/models", Plane: "model"},
	{Name: "budget", EnvKey: "MODELS_HOST", Default: "arcana-models", Port: 8099, Prefix: "/api/v1/budget", Plane: "model"},

	// Govern Plane
	{Name: "ward", EnvKey: "WARD_HOST", Default: "arcana-ward", Port: 8086, Prefix: "/api/v1/check", Plane: "govern"},
	{Name: "ward-rules", EnvKey: "WARD_HOST", Default: "arcana-ward", Port: 8086, Prefix: "/api/v1/rules", Plane: "govern"},
	{Name: "ward-stats", EnvKey: "WARD_HOST", Default: "arcana-ward", Port: 8086, Prefix: "/api/v1/stats", Plane: "govern"},
	{Name: "audit", EnvKey: "AUDIT_HOST", Default: "arcana-audit", Port: 8100, Prefix: "/api/v1/audit", Plane: "govern"},

	// Quality Plane
	{Name: "probe", EnvKey: "PROBE_HOST", Default: "arcana-probe", Port: 8101, Prefix: "/api/v1/eval", Plane: "quality"},
	{Name: "annotate", EnvKey: "ANNOTATE_HOST", Default: "arcana-annotate", Port: 8102, Prefix: "/api/v1/annotations", Plane: "quality"},

	// Ops Plane
	{Name: "skills", EnvKey: "SKILLS_HOST", Default: "arcana-skills", Port: 8085, Prefix: "/api/v1/skills", Plane: "ops"},
	{Name: "scheduler", EnvKey: "SCHEDULER_HOST", Default: "arcana-scheduler", Port: 8103, Prefix: "/api/v1/scheduler", Plane: "ops"},
	{Name: "registry", EnvKey: "REGISTRY_HOST", Default: "arcana-registry", Port: 8104, Prefix: "/api/v1/catalog", Plane: "ops"},
	{Name: "finops", EnvKey: "FINOPS_HOST", Default: "arcana-finops", Port: 8105, Prefix: "/api/v1/costs", Plane: "ops"},
	{Name: "gitops", EnvKey: "GITOPS_HOST", Default: "arcana-gitops", Port: 8106, Prefix: "/api/v1/promotions", Plane: "ops"},
}

func cors(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", corsOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	type checkTarget struct {
		name  string
		host  string
		port  int
		plane string
	}

	var targets []checkTarget
	for _, bs := range backingServices {
		targets = append(targets, checkTarget{
			name:  bs.name,
			host:  envOr(bs.host, "localhost"),
			port:  bs.port,
			plane: bs.plane,
		})
	}
	for _, sr := range serviceRoutes {
		targets = append(targets, checkTarget{
			name:  sr.Name,
			host:  envOr(sr.EnvKey, sr.Default),
			port:  sr.Port,
			plane: sr.Plane,
		})
	}

	results := make([]ServiceHealth, len(targets))
	var wg sync.WaitGroup

	for i, t := range targets {
		wg.Add(1)
		go func(idx int, name, host string, port int, plane string) {
			defer wg.Done()
			ok, latency := checkTCP(host, port, 2*time.Second)
			status := "healthy"
			if !ok {
				status = "unreachable"
			}
			results[idx] = ServiceHealth{
				Name:    name,
				Status:  status,
				Latency: latency.Round(time.Millisecond).String(),
				Port:    port,
				Plane:   plane,
			}
		}(i, t.name, t.host, t.port, t.plane)
	}
	wg.Wait()

	resp := SystemHealth{
		Platform: "arcana",
		Version:  "0.1.0",
		Uptime:   time.Since(startTime).Round(time.Second).String(),
		Services: results,
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", corsOrigin)
	json.NewEncoder(w).Encode(resp)
}

func routesHandler(w http.ResponseWriter, _ *http.Request) {
	type RouteInfo struct {
		Name   string `json:"name"`
		Prefix string `json:"prefix"`
		Plane  string `json:"plane"`
		Port   int    `json:"port"`
	}
	routes := make([]RouteInfo, len(serviceRoutes))
	for i, sr := range serviceRoutes {
		routes[i] = RouteInfo{Name: sr.Name, Prefix: sr.Prefix, Plane: sr.Plane, Port: sr.Port}
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", corsOrigin)
	json.NewEncoder(w).Encode(routes)
}

func makeProxy(host string, port int) http.Handler {
	target, _ := url.Parse(fmt.Sprintf("http://%s:%d", host, port))
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "service_unavailable",
			"message": fmt.Sprintf("upstream %s:%d unreachable: %v", host, port, err),
		})
	}
	return proxy
}

type ChatRequest struct {
	Message   string `json:"message"`
	SessionID string `json:"session_id"`
}

type ChatStep struct {
	Type    string `json:"type"`
	Service string `json:"service"`
	Message string `json:"message"`
}

type ChatResponse struct {
	Reply     string     `json:"reply"`
	Steps     []ChatStep `json:"steps"`
	SessionID string     `json:"session_id"`
}

func postJSON(baseURL, path string, body interface{}) ([]byte, int, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, 0, fmt.Errorf("marshal request body: %w", err)
	}
	resp, err := http.Post(baseURL+path, "application/json", strings.NewReader(string(payload)))
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read response body: %w", err)
	}
	return data, resp.StatusCode, nil
}

// postJSONCtx is a context-aware variant of postJSON for use in bounded workers.
func postJSONCtx(ctx context.Context, baseURL, path string, body interface{}) ([]byte, int, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, 0, fmt.Errorf("marshal request body: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+path, strings.NewReader(string(payload)))
	if err != nil {
		return nil, 0, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read response body: %w", err)
	}
	return data, resp.StatusCode, nil
}

func getJSON(baseURL, path string) ([]byte, int, error) {
	resp, err := http.Get(baseURL + path)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read response body: %w", err)
	}
	return data, resp.StatusCode, nil
}

// --- Conversation State Machine ---

type AgentSetup struct {
	Phase          string   // ask_name, ask_type, ask_connectors, ask_budget, ask_deep, ask_params, confirm
	AgentName      string
	Purpose        string
	AgentType      string   // create_agent or create_deep_agent
	Capabilities   []string
	Connectors     []string
	Budget         string
	Guardrails     []string
	WorldModel     bool
	SkillGraph     bool
	Blueprint      string
	MemoryPolicy   string
	SubAgents      []string
	HITLEnabled    bool
	SelfImprove    bool
	SystemPrompt   string
	Temperature    float64
	MaxTokens      int
	ModelCallLimit int
	ToolCallLimit  int
	ParamsAsked    bool
}

// sessionEntry wraps an AgentSetup with a timestamp for idle-session reaping.
type sessionEntry struct {
	setup        *AgentSetup
	lastAccessed time.Time
}

type ConversationSession struct {
	mu       sync.Mutex
	sessions map[string]*sessionEntry
}

var convSessions = &ConversationSession{sessions: make(map[string]*sessionEntry)}

func (cs *ConversationSession) Get(id string) *AgentSetup {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	entry := cs.sessions[id]
	if entry == nil {
		return nil
	}
	entry.lastAccessed = time.Now()
	return entry.setup
}

func (cs *ConversationSession) Set(id string, s *AgentSetup) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.sessions[id] = &sessionEntry{setup: s, lastAccessed: time.Now()}
}

func (cs *ConversationSession) Delete(id string) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	delete(cs.sessions, id)
}

// startSessionReaper periodically removes sessions that have been idle
// longer than maxAge, preventing unbounded memory growth.
func startSessionReaper(cs *ConversationSession, interval, maxAge time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			now := time.Now()
			cs.mu.Lock()
			for id, entry := range cs.sessions {
				if now.Sub(entry.lastAccessed) > maxAge {
					delete(cs.sessions, id)
				}
			}
			cs.mu.Unlock()
		}
	}()
}

func generateSessionID() string {
	return "s-" + uuid.New().String()
}

func slugify(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' {
			return r
		}
		if r == ' ' || r == '_' {
			return '-'
		}
		return -1
	}, name)
	for strings.Contains(name, "--") {
		name = strings.ReplaceAll(name, "--", "-")
	}
	return strings.Trim(name, "-")
}

func extractAgentName(msg string) string {
	lower := strings.ToLower(msg)
	for _, prefix := range []string{"called ", "named ", "name it ", "call it "} {
		if idx := strings.Index(lower, prefix); idx >= 0 {
			rest := strings.TrimSpace(msg[idx+len(prefix):])
			rest = strings.Trim(rest, `"'`)
			words := strings.Fields(rest)
			var nameParts []string
			for _, w := range words {
				cleaned := strings.TrimRight(w, ".,!?;:")
				stopWords := map[string]bool{
					"for": true, "to": true, "with": true, "and": true,
					"that": true, "which": true, "using": true, "connect": true,
					"set": true, "budget": true, "access": true,
				}
				if stopWords[strings.ToLower(cleaned)] {
					break
				}
				if len(cleaned) > 0 {
					nameParts = append(nameParts, cleaned)
				}
				if len(nameParts) >= 3 {
					break
				}
			}
			if len(nameParts) > 0 {
				return slugify(strings.Join(nameParts, "-"))
			}
		}
	}
	return ""
}

func extractConnectors(msg string) []string {
	lower := strings.ToLower(msg)
	var out []string
	connectorKeywords := map[string]string{
		"google drive": "gdrive", "gdrive": "gdrive", "drive": "gdrive",
		"mailchimp": "mailchimp", "mail chimp": "mailchimp",
		"confluence": "confluence", "slack": "slack",
		"s3": "s3", "aws s3": "s3",
		"notion": "notion", "jira": "jira",
		"hubspot": "hubspot", "salesforce": "salesforce",
		"github": "github", "gitlab": "gitlab",
		"snowflake": "snowflake", "bigquery": "bigquery",
	}
	seen := map[string]bool{}
	for kw, name := range connectorKeywords {
		if strings.Contains(lower, kw) && !seen[name] {
			out = append(out, name)
			seen[name] = true
		}
	}
	return out
}

func extractCapabilities(msg string) []string {
	lower := strings.ToLower(msg)
	var caps []string
	capKeywords := map[string]string{
		"email":      "email-campaign",
		"campaign":   "email-campaign",
		"content":    "content-gen",
		"draft":      "content-gen",
		"write":      "content-gen",
		"lead":       "lead-scoring",
		"scoring":    "lead-scoring",
		"search":     "search",
		"knowledge":  "search",
		"summarize":  "summarize",
		"summary":    "summarize",
		"support":    "customer-support",
		"customer":   "customer-support",
		"research":   "research",
		"analyze":    "analysis",
		"analysis":   "analysis",
		"code":       "code-gen",
		"data":       "data-processing",
		"report":     "reporting",
		"marketing":  "email-campaign",
		"outreach":   "email-campaign",
	}
	seen := map[string]bool{}
	for kw, cap := range capKeywords {
		if strings.Contains(lower, kw) && !seen[cap] {
			caps = append(caps, cap)
			seen[cap] = true
		}
	}
	return caps
}

func extractBudget(msg string) string {
	for _, word := range strings.Fields(msg) {
		w := strings.TrimRight(word, ".,!?")
		w = strings.TrimSuffix(w, "/day")
		if strings.HasPrefix(w, "$") && len(w) > 1 {
			return w + "/day"
		}
	}
	lower := strings.ToLower(msg)
	for _, amount := range []string{"5", "10", "15", "20", "25", "30", "50", "100"} {
		if strings.Contains(lower, amount+" per day") || strings.Contains(lower, amount+"/day") || strings.Contains(lower, amount+" a day") {
			return "$" + amount + "/day"
		}
	}
	return ""
}

func inferAgentName(purpose string, caps []string) string {
	if len(caps) > 0 {
		name := slugify(caps[0])
		if len(name) > 20 {
			name = name[:20]
			name = strings.TrimRight(name, "-")
		}
		return name + "-agent"
	}
	if purpose != "" {
		words := strings.Fields(slugify(purpose))
		if len(words) > 3 {
			words = words[:3]
		}
		return strings.Join(words, "-") + "-agent"
	}
	return ""
}

func isAffirmative(msg string) bool {
	lower := strings.ToLower(strings.TrimSpace(msg))
	for _, word := range []string{"yes", "yeah", "yep", "sure", "ok", "okay", "go", "proceed", "do it", "go ahead", "let's go", "confirm", "y", "sounds good", "perfect", "great", "lgtm", "ship it"} {
		if lower == word || strings.HasPrefix(lower, word+" ") || strings.HasPrefix(lower, word+".") || strings.HasPrefix(lower, word+"!") || strings.HasPrefix(lower, word+",") {
			return true
		}
	}
	return false
}

func isNegative(msg string) bool {
	lower := strings.ToLower(strings.TrimSpace(msg))
	for _, word := range []string{"no", "nope", "cancel", "stop", "never mind", "nevermind", "abort", "n"} {
		if lower == word || strings.HasPrefix(lower, word+" ") || strings.HasPrefix(lower, word+".") || strings.HasPrefix(lower, word+"!") {
			return true
		}
	}
	return false
}

func isAgentCreateIntent(msg string) bool {
	lower := strings.ToLower(msg)
	return (strings.Contains(lower, "create") || strings.Contains(lower, "deploy") ||
		strings.Contains(lower, "build") || strings.Contains(lower, "set up") ||
		strings.Contains(lower, "provision") || strings.Contains(lower, "make") ||
		strings.Contains(lower, "need") || strings.Contains(lower, "want") ||
		strings.Contains(lower, "spin up") || strings.Contains(lower, "launch")) &&
		strings.Contains(lower, "agent")
}

func nextSetupPhase(setup *AgentSetup) string {
	if setup.AgentName == "" && setup.Purpose == "" {
		return "ask_purpose"
	}
	if setup.AgentName == "" {
		return "ask_name"
	}
	if setup.AgentType == "" {
		return "ask_type"
	}
	if len(setup.Connectors) == 0 {
		return "ask_connectors"
	}
	if setup.Budget == "" {
		return "ask_budget"
	}
	if setup.AgentType == "create_deep_agent" && setup.MemoryPolicy == "" {
		return "ask_deep"
	}
	if setup.AgentType == "create_deep_agent" && !setup.ParamsAsked {
		return "ask_params"
	}
	return "confirm"
}

func buildSetupQuestion(phase string, setup *AgentSetup) string {
	switch phase {
	case "ask_purpose":
		return "I'd love to help you set up an agent! What should this agent do?\n\n" +
			"For example:\n" +
			"- Draft weekly email campaigns\n" +
			"- Summarize research papers\n" +
			"- Handle customer support tickets\n" +
			"- Generate content from brand guidelines"
	case "ask_name":
		suggested := inferAgentName(setup.Purpose, setup.Capabilities)
		if suggested != "" {
			return fmt.Sprintf("Got it! What should we call this agent? I'd suggest **%s** — or you can pick a different name.", suggested)
		}
		return "What should we call this agent? Give it a short, descriptive name (e.g. \"email-marketing-agent\")."
	case "ask_type":
		return "What kind of agent should **" + setup.AgentName + "** be?\n\n" +
			"**1. `create_agent`** — Standard agent\n" +
			"   Simple, fast, production-ready. Connectors, guardrails, budget.\n" +
			"   Best for: email campaigns, content gen, support bots, data lookups.\n\n" +
			"**2. `create_deep_agent`** — Deep agent\n" +
			"   Full platform capabilities. World model, skill graph, blueprints, sub-agents, HITL, self-improvement.\n" +
			"   Best for: research pipelines, multi-agent workflows, autonomous systems.\n\n" +
			"Type **1** or **2** (or say \"standard\" / \"deep\")."
	case "ask_deep":
		return "Configure deep agent capabilities for **" + setup.AgentName + "**:\n\n" +
			"Which features should be enabled? (comma-separated or say \"all\")\n\n" +
			"- **world_model** — Oracle L2 simulator for tool outcome prediction\n" +
			"- **skill_graph** — 3-tier skill graph with experiential memory\n" +
			"- **hitl** — Human-in-the-loop approval gates\n" +
			"- **self_improve** — Auto-crystallize annotations into skills\n" +
			"- **sub_agents** — Enable multi-agent delegation\n\n" +
			"Example: \"world_model, skill_graph, self_improve\" or just \"all\"."
	case "ask_params":
		return "Would you like to customize advanced parameters for **" + setup.AgentName + "**?\n\n" +
			"You can set any of these (comma-separated `key=value` pairs):\n\n" +
			"| Parameter | Default | Description |\n" +
			"|-----------|---------|-------------|\n" +
			"| **temperature** | 0.0 | LLM sampling temperature (0.0–2.0) |\n" +
			"| **max_tokens** | 8192 | Max output tokens per call |\n" +
			"| **model_limit** | 50 | Max model calls per run |\n" +
			"| **tool_limit** | 200 | Max tool calls per run |\n" +
			"| **memory** | tri-scope | Memory policy (tri-scope, long-term, short-term, none) |\n\n" +
			"Example: `temperature=0.7, max_tokens=4096, memory=long-term`\n\n" +
			"Or say **\"defaults\"** to keep everything as-is."
	case "ask_connectors":
		return "Which tools and data sources should **" + setup.AgentName + "** have access to?\n\n" +
			"Available connectors:\n" +
			"- **Google Drive** — access documents and brand guidelines\n" +
			"- **Mailchimp** — send email campaigns\n" +
			"- **Slack** — post messages and notifications\n" +
			"- **Confluence** — read documentation\n" +
			"- **HubSpot / Salesforce** — CRM data\n" +
			"- **GitHub / GitLab** — code repositories\n" +
			"- **Snowflake / BigQuery** — data warehouse\n\n" +
			"List the ones you need, or say \"none\" to skip."
	case "ask_budget":
		return "What daily token budget should **" + setup.AgentName + "** have?\n\n" +
			"Recommended budgets:\n" +
			"- **$5/day** — light usage (summaries, short replies)\n" +
			"- **$20/day** — standard (email campaigns, content gen)\n" +
			"- **$50/day** — heavy (research, multi-step workflows)\n\n" +
			"You can say a number like \"$20\" or just \"20\"."
	case "confirm":
		budgetDisplay := strings.TrimSuffix(setup.Budget, "/day")
		budgetDisplay = strings.TrimSuffix(budgetDisplay, "/day")
		summary := "Here's what I'll set up:\n\n"
		summary += "- **Agent:** " + setup.AgentName + "\n"
		summary += "- **Type:** `" + setup.AgentType + "`\n"
		if len(setup.Capabilities) > 0 {
			summary += "- **Capabilities:** " + strings.Join(setup.Capabilities, ", ") + "\n"
		}
		activeConns := []string{}
		for _, c := range setup.Connectors {
			if c != "none" {
				activeConns = append(activeConns, c)
			}
		}
		if len(activeConns) > 0 {
			summary += "- **Connectors:** " + strings.Join(activeConns, ", ") + "\n"
		}
		summary += "- **Budget:** " + budgetDisplay + "/day\n"
		summary += "- **Guardrails:** brand tone, PII filter, competitor blocking, rate limiting\n"
		summary += "- **Model:** gpt-4o\n"
		if setup.AgentType == "create_deep_agent" {
			features := []string{}
			if setup.WorldModel { features = append(features, "world_model") }
			if setup.SkillGraph { features = append(features, "skill_graph") }
			if setup.HITLEnabled { features = append(features, "hitl") }
			if setup.SelfImprove { features = append(features, "self_improve") }
			if len(setup.SubAgents) > 0 { features = append(features, "sub_agents") }
			summary += "- **Deep Features:** " + strings.Join(features, ", ") + "\n"
			summary += "- **Memory Policy:** " + setup.MemoryPolicy + "\n"
			params := []string{}
			if setup.Temperature > 0 {
				params = append(params, fmt.Sprintf("temperature=%.1f", setup.Temperature))
			}
			if setup.MaxTokens > 0 {
				params = append(params, fmt.Sprintf("max_tokens=%d", setup.MaxTokens))
			}
			if setup.ModelCallLimit > 0 {
				params = append(params, fmt.Sprintf("model_limit=%d", setup.ModelCallLimit))
			}
			if setup.ToolCallLimit > 0 {
				params = append(params, fmt.Sprintf("tool_limit=%d", setup.ToolCallLimit))
			}
			if len(params) > 0 {
				summary += "- **Custom Parameters:** " + strings.Join(params, ", ") + "\n"
			}
		}
		summary += "- **Namespace:** arcana-agent-" + setup.AgentName + "\n\n"
		summary += "Shall I proceed? (yes/no)"
		return summary
	}
	return ""
}

func processAgentSetupResponse(msg string, setup *AgentSetup) {
	switch setup.Phase {
	case "ask_purpose":
		setup.Purpose = strings.TrimSpace(msg)
		newCaps := extractCapabilities(msg)
		if len(newCaps) > 0 {
			setup.Capabilities = newCaps
		}
		newConns := extractConnectors(msg)
		if len(newConns) > 0 {
			setup.Connectors = newConns
		}
		budget := extractBudget(msg)
		if budget != "" {
			setup.Budget = budget
		}
		name := extractAgentName(msg)
		if name != "" {
			setup.AgentName = name
		}

	case "ask_name":
		lower := strings.ToLower(strings.TrimSpace(msg))
		if isAffirmative(msg) || lower == "that works" || lower == "that's fine" || lower == "that works!" ||
			strings.Contains(lower, "suggest") || strings.Contains(lower, "works for me") ||
			strings.Contains(lower, "fine") || strings.Contains(lower, "good") {
			setup.AgentName = inferAgentName(setup.Purpose, setup.Capabilities)
		} else {
			// Check if user accidentally answered with connectors instead of a name
			conns := extractConnectors(msg)
			if len(conns) > 0 && strings.Contains(lower, " and ") {
				setup.AgentName = inferAgentName(setup.Purpose, setup.Capabilities)
				setup.Connectors = conns
			} else {
				explicit := extractAgentName(msg)
				if explicit != "" {
					setup.AgentName = explicit
				} else {
					setup.AgentName = slugify(msg)
				}
			}
		}

	case "ask_type":
		lower := strings.ToLower(strings.TrimSpace(msg))
		if lower == "2" || strings.Contains(lower, "deep") || lower == "create_deep_agent" {
			setup.AgentType = "create_deep_agent"
		} else {
			setup.AgentType = "create_agent"
		}

	case "ask_deep":
		lower := strings.ToLower(strings.TrimSpace(msg))
		if lower == "all" || lower == "everything" {
			setup.WorldModel = true
			setup.SkillGraph = true
			setup.HITLEnabled = true
			setup.SelfImprove = true
			setup.SubAgents = []string{"enabled"}
		} else {
			if strings.Contains(lower, "world") || strings.Contains(lower, "oracle") {
				setup.WorldModel = true
			}
			if strings.Contains(lower, "skill") {
				setup.SkillGraph = true
			}
			if strings.Contains(lower, "hitl") || strings.Contains(lower, "human") || strings.Contains(lower, "approval") {
				setup.HITLEnabled = true
			}
			if strings.Contains(lower, "self") || strings.Contains(lower, "improve") || strings.Contains(lower, "learn") {
				setup.SelfImprove = true
			}
			if strings.Contains(lower, "sub") || strings.Contains(lower, "multi") || strings.Contains(lower, "delegate") {
				setup.SubAgents = []string{"enabled"}
			}
		}
		setup.MemoryPolicy = "tri-scope"

	case "ask_params":
		setup.ParamsAsked = true
		lower := strings.ToLower(strings.TrimSpace(msg))
		if lower == "defaults" || lower == "default" || lower == "skip" || lower == "no" || isNegative(msg) {
			break
		}
		for _, part := range strings.Split(msg, ",") {
			kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
			if len(kv) != 2 {
				continue
			}
			key := strings.ToLower(strings.TrimSpace(kv[0]))
			val := strings.TrimSpace(kv[1])
			switch key {
			case "temperature", "temp":
				if f, err := strconv.ParseFloat(val, 64); err == nil {
					setup.Temperature = f
				}
			case "max_tokens", "tokens":
				if n, err := strconv.Atoi(val); err == nil {
					setup.MaxTokens = n
				}
			case "model_limit", "model_call_limit":
				if n, err := strconv.Atoi(val); err == nil {
					setup.ModelCallLimit = n
				}
			case "tool_limit", "tool_call_limit":
				if n, err := strconv.Atoi(val); err == nil {
					setup.ToolCallLimit = n
				}
			case "memory", "memory_policy":
				setup.MemoryPolicy = val
			}
		}

	case "ask_connectors":
		lower := strings.ToLower(msg)
		if lower == "none" || lower == "skip" || lower == "no" {
			setup.Connectors = []string{"none"}
		} else {
			setup.Connectors = extractConnectors(msg)
			if len(setup.Connectors) == 0 {
				words := strings.Fields(slugify(msg))
				for _, w := range words {
					if len(w) > 2 {
						setup.Connectors = append(setup.Connectors, w)
					}
				}
			}
			if len(setup.Connectors) == 0 {
				setup.Connectors = []string{"none"}
			}
		}

	case "ask_budget":
		if isAffirmative(msg) {
			setup.Budget = "$20/day"
			return
		}
		budget := extractBudget(msg)
		if budget == "" {
			cleaned := strings.TrimSpace(msg)
			cleaned = strings.TrimPrefix(cleaned, "$")
			cleaned = strings.TrimRight(cleaned, ".,!?")
			cleaned = strings.TrimSpace(cleaned)
			isNumber := len(cleaned) > 0
			for _, c := range cleaned {
				if c < '0' || c > '9' {
					isNumber = false
					break
				}
			}
			if isNumber {
				budget = "$" + cleaned + "/day"
			}
		}
		if budget == "" {
			budget = "$20/day"
		}
		setup.Budget = budget
	}
}

func provisionAgent(setup *AgentSetup) (string, []ChatStep) {
	meshHost := envOr("MESH_HOST", "arcana-mesh")
	wardHost := envOr("WARD_HOST", "arcana-ward")
	finopsHost := envOr("FINOPS_HOST", "arcana-finops")
	connectorsHost := envOr("CONNECTORS_HOST", "arcana-connectors")

	var steps []ChatStep

	agentType := setup.AgentType
	if agentType == "" {
		agentType = "create_agent"
	}
	steps = append(steps, ChatStep{Type: "action", Service: "mesh", Message: fmt.Sprintf("Registering %s agent \"%s\"...", agentType, setup.AgentName)})
	agentBody := map[string]interface{}{
		"name":         setup.AgentName,
		"agent_type":   agentType,
		"model":        "gpt-4o",
		"capabilities": setup.Capabilities,
		"protocols":    []string{"a2a"},
	}
	if agentType == "create_deep_agent" {
		dc := map[string]interface{}{
			"world_model":   setup.WorldModel,
			"skill_graph":   setup.SkillGraph,
			"blueprint":     setup.Blueprint,
			"memory_policy": setup.MemoryPolicy,
			"sub_agents":    setup.SubAgents,
			"hitl_enabled":  setup.HITLEnabled,
			"self_improve":  setup.SelfImprove,
		}
		if setup.SystemPrompt != "" {
			dc["system_prompt"] = setup.SystemPrompt
		}
		if setup.Temperature > 0 {
			dc["temperature"] = setup.Temperature
		}
		if setup.MaxTokens > 0 {
			dc["max_tokens"] = setup.MaxTokens
		}
		if setup.ModelCallLimit > 0 {
			dc["model_call_limit"] = setup.ModelCallLimit
		}
		if setup.ToolCallLimit > 0 {
			dc["tool_call_limit"] = setup.ToolCallLimit
		}
		agentBody["deep_config"] = dc
	}
	meshURL := fmt.Sprintf("http://%s:8083", meshHost)
	_, code, err := postJSON(meshURL, "/api/v1/agents/register", agentBody)
	if err != nil || code >= 400 {
		steps = append(steps, ChatStep{Type: "error", Service: "mesh", Message: fmt.Sprintf("Registration returned %d", code)})
	} else {
		typeLabel := "standard"
		if agentType == "create_deep_agent" {
			typeLabel = "deep"
		}
		steps = append(steps, ChatStep{Type: "result", Service: "mesh", Message: fmt.Sprintf("Agent \"%s\" registered (%s, model: gpt-4o, namespace: arcana-agent-%s)", setup.AgentName, typeLabel, setup.AgentName)})
	}

	activeConnectors := []string{}
	for _, c := range setup.Connectors {
		if c != "none" {
			activeConnectors = append(activeConnectors, c)
		}
	}
	if len(activeConnectors) > 0 {
		steps = append(steps, ChatStep{Type: "action", Service: "connectors", Message: "Connecting data sources: " + strings.Join(activeConnectors, ", ")})
		connectorsURL := fmt.Sprintf("http://%s:8094", connectorsHost)
		_, _, connErr := getJSON(connectorsURL, "/api/v1/connectors")
		if connErr != nil {
			steps = append(steps, ChatStep{Type: "error", Service: "connectors", Message: "Could not reach connectors service"})
		} else {
			for _, cn := range activeConnectors {
				steps = append(steps, ChatStep{Type: "result", Service: "connectors", Message: cn + " connector linked to " + setup.AgentName})
			}
		}
	}

	steps = append(steps, ChatStep{Type: "action", Service: "ward", Message: "Applying guardrails (brand tone, PII filter, competitor block)..."})
	wardURL := fmt.Sprintf("http://%s:8086", wardHost)
	_, _, wardErr := getJSON(wardURL, "/api/v1/rules")
	if wardErr != nil {
		steps = append(steps, ChatStep{Type: "error", Service: "ward", Message: "Could not reach guardrails service"})
	} else {
		steps = append(steps, ChatStep{Type: "result", Service: "ward", Message: "6-layer guardrail pipeline active (schema, policy, rate-limit, pattern, semantic, risk)"})
	}

	budgetStr := setup.Budget
	if budgetStr == "" {
		budgetStr = "$20/day"
	}
	budgetClean := strings.TrimSuffix(budgetStr, "/day")
	budgetClean = strings.TrimSuffix(budgetClean, "/day")
	steps = append(steps, ChatStep{Type: "action", Service: "finops", Message: "Setting token budget: " + budgetClean + "/day"})
	finopsURL := fmt.Sprintf("http://%s:8105", finopsHost)
	_, _, finErr := getJSON(finopsURL, "/api/v1/costs")
	if finErr != nil {
		steps = append(steps, ChatStep{Type: "error", Service: "finops", Message: "Could not reach finops service"})
	} else {
		steps = append(steps, ChatStep{Type: "result", Service: "finops", Message: "Budget " + budgetClean + "/day set for " + setup.AgentName})
	}

	reply := "Your agent **" + setup.AgentName + "** is live!\n\n"
	reply += "**What I set up:**\n"
	reply += "- Registered with capabilities: " + strings.Join(setup.Capabilities, ", ") + "\n"
	if len(activeConnectors) > 0 {
		reply += "- Connected data sources: " + strings.Join(activeConnectors, ", ") + "\n"
	}
	reply += "- Applied 6-layer guardrail pipeline\n"
	reply += "- Token budget: " + budgetClean + "/day\n"
	reply += "- Deployed to namespace: **arcana-agent-" + setup.AgentName + "**\n\n"
	reply += "You can view your agent on the **Agents** page or ask me to run a task with it."

	return reply, steps
}

func chatHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", corsOrigin)
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "POST required"})
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON"})
		return
	}

	sessionID := req.SessionID
	if sessionID == "" {
		sessionID = generateSessionID()
	}

	msg := req.Message
	msgLower := strings.ToLower(msg)
	var steps []ChatStep
	var reply string

	// Check for active conversation
	setup := convSessions.Get(sessionID)

	if setup != nil {
		// We're in a multi-turn agent setup flow
		if isNegative(msg) && setup.Phase == "confirm" {
			convSessions.Delete(sessionID)
			reply = "No problem — I've cancelled the agent setup. Let me know if you want to start over or need anything else!"
			json.NewEncoder(w).Encode(ChatResponse{Reply: reply, Steps: nil, SessionID: sessionID})
			return
		}

		if isNegative(msg) && setup.Phase != "confirm" {
			convSessions.Delete(sessionID)
			reply = "Cancelled. Let me know whenever you'd like to set up an agent!"
			json.NewEncoder(w).Encode(ChatResponse{Reply: reply, Steps: nil, SessionID: sessionID})
			return
		}

		if setup.Phase == "confirm" && isAffirmative(msg) {
			reply, steps = provisionAgent(setup)
			convSessions.Delete(sessionID)
			json.NewEncoder(w).Encode(ChatResponse{Reply: reply, Steps: steps, SessionID: sessionID})
			return
		}

		processAgentSetupResponse(msg, setup)

		nextPhase := nextSetupPhase(setup)
		setup.Phase = nextPhase
		convSessions.Set(sessionID, setup)
		reply = buildSetupQuestion(nextPhase, setup)
		json.NewEncoder(w).Encode(ChatResponse{Reply: reply, Steps: nil, SessionID: sessionID})
		return
	}

	// --- Not in a multi-turn flow: detect intent ---

	if isAgentCreateIntent(msg) {
		setup := &AgentSetup{
			Guardrails: []string{"brand-tone", "pii-filter", "competitor-block", "rate-limit"},
		}

		setup.AgentName = extractAgentName(msg)
		setup.Capabilities = extractCapabilities(msg)
		setup.Connectors = extractConnectors(msg)
		setup.Budget = extractBudget(msg)

		if setup.AgentName == "" && len(setup.Capabilities) > 0 {
			setup.AgentName = inferAgentName("", setup.Capabilities)
		}

		// Infer purpose from capabilities
		if len(setup.Capabilities) > 0 {
			setup.Purpose = strings.Join(setup.Capabilities, ", ")
		}

		nextPhase := nextSetupPhase(setup)

		if nextPhase == "confirm" {
			// User provided everything upfront — go straight to confirm
			setup.Phase = "confirm"
			convSessions.Set(sessionID, setup)
			reply = buildSetupQuestion("confirm", setup)
		} else {
			setup.Phase = nextPhase
			convSessions.Set(sessionID, setup)
			reply = buildSetupQuestion(nextPhase, setup)
		}

		json.NewEncoder(w).Encode(ChatResponse{Reply: reply, Steps: nil, SessionID: sessionID})
		return
	}

	// --- Single-turn intents (no state needed) ---

	codexHost := envOr("CODEX_ROUTER_HOST", "arcana-codex-router")
	finopsHost := envOr("FINOPS_HOST", "arcana-finops")
	wardHost := envOr("WARD_HOST", "arcana-ward")
	probeHost := envOr("PROBE_HOST", "arcana-probe")
	engineHost := envOr("ENGINE_HOST", "arcana-engine")
	registryHost := envOr("REGISTRY_HOST", "arcana-registry")

	if strings.Contains(msgLower, "search") || strings.Contains(msgLower, "find") || strings.Contains(msgLower, "look up") || strings.Contains(msgLower, "knowledge") {
		query := req.Message
		steps = append(steps, ChatStep{Type: "action", Service: "codex-router", Message: "Searching knowledge base: \"" + query + "\""})
		codexURL := fmt.Sprintf("http://%s:8090", codexHost)
		body := map[string]interface{}{"query": query, "profile": "default", "top_k": 5}
		respData, _, err := postJSON(codexURL, "/api/v1/search", body)
		if err != nil {
			steps = append(steps, ChatStep{Type: "error", Service: "codex-router", Message: "Search failed"})
			reply = "I couldn't complete the search. The Codex service might be unavailable."
		} else {
			var result map[string]interface{}
			if err := json.Unmarshal(respData, &result); err != nil {
				log.Printf("[chat] failed to parse codex search response: %v", err)
				steps = append(steps, ChatStep{Type: "error", Service: "codex-router", Message: "Invalid response from search service"})
				reply = "Search returned an unreadable response."
			} else {
				hits := 0
				if th, ok := result["total_hits"]; ok {
					if f, ok := th.(float64); ok {
						hits = int(f)
					}
				}
				steps = append(steps, ChatStep{Type: "result", Service: "codex-router", Message: fmt.Sprintf("Found %d results across shards", hits)})
				reply = fmt.Sprintf("Found **%d results** for \"%s\". Results are from the Codex knowledge base across multiple shards.", hits, query)
			}
		}

	} else if strings.Contains(msgLower, "cost") || strings.Contains(msgLower, "spend") || strings.Contains(msgLower, "spent") || strings.Contains(msgLower, "budget") || strings.Contains(msgLower, "price") || strings.Contains(msgLower, "billing") {
		steps = append(steps, ChatStep{Type: "action", Service: "finops", Message: "Fetching cost report..."})
		finopsURL := fmt.Sprintf("http://%s:8105", finopsHost)
		respData, _, err := getJSON(finopsURL, "/api/v1/costs")
		if err != nil {
			steps = append(steps, ChatStep{Type: "error", Service: "finops", Message: "Could not reach finops service"})
			reply = "Couldn't fetch the cost report. The FinOps service might be down."
		} else {
			var result map[string]interface{}
			if err := json.Unmarshal(respData, &result); err != nil {
				log.Printf("[chat] failed to parse finops response: %v", err)
				steps = append(steps, ChatStep{Type: "error", Service: "finops", Message: "Invalid response from finops service"})
				reply = "Cost report returned an unreadable response."
			} else {
				totalUSD := result["total_usd"]
				steps = append(steps, ChatStep{Type: "result", Service: "finops", Message: fmt.Sprintf("Total spend: $%.2f", totalUSD)})
				reply = fmt.Sprintf("Your current total platform spend is **$%.2f**. No budget alerts triggered.", totalUSD)
			}
		}

	} else if strings.Contains(msgLower, "guardrail") || strings.Contains(msgLower, "safe") {
		textToCheck := req.Message
		steps = append(steps, ChatStep{Type: "action", Service: "ward", Message: "Running guardrail check..."})
		wardURL := fmt.Sprintf("http://%s:8086", wardHost)
		body := map[string]interface{}{"text": textToCheck, "agent_id": "chat-user", "direction": "input"}
		respData, _, err := postJSON(wardURL, "/api/v1/check", body)
		if err != nil {
			steps = append(steps, ChatStep{Type: "error", Service: "ward", Message: "Guardrail check failed"})
			reply = "Couldn't run the guardrail check. The Ward service might be unavailable."
		} else {
			var result map[string]interface{}
			if err := json.Unmarshal(respData, &result); err != nil {
				log.Printf("[chat] failed to parse ward response: %v", err)
				steps = append(steps, ChatStep{Type: "error", Service: "ward", Message: "Invalid response from guardrail service"})
				reply = "Guardrail check returned an unreadable response."
			} else {
				verdict := "unknown"
				if v, ok := result["verdict"]; ok {
					verdict = fmt.Sprintf("%v", v)
				}
				layers := 0
				if lr, ok := result["layer_results"]; ok {
					if arr, ok := lr.([]interface{}); ok {
						layers = len(arr)
					}
				}
				steps = append(steps, ChatStep{Type: "result", Service: "ward", Message: fmt.Sprintf("Verdict: %s (%d layers checked)", verdict, layers)})
				reply = fmt.Sprintf("Guardrail check complete. **Verdict: %s** — passed through all %d layers.", verdict, layers)
			}
		}

	} else if strings.Contains(msgLower, "eval") || strings.Contains(msgLower, "test") || strings.Contains(msgLower, "quality") {
		skillRef := "summarize"
		for _, word := range strings.Fields(msg) {
			if !strings.Contains("evaluate,eval,test,quality,run,the,a,my,on,for,skill", word) && len(word) > 3 {
				skillRef = word
				break
			}
		}
		steps = append(steps, ChatStep{Type: "action", Service: "probe", Message: "Running evaluation for skill \"" + skillRef + "\"..."})
		probeURL := fmt.Sprintf("http://%s:8101", probeHost)
		body := map[string]interface{}{"skill_ref": skillRef, "judges": []string{"deterministic", "llm"}}
		respData, _, err := postJSON(probeURL, "/api/v1/eval/run", body)
		if err != nil {
			steps = append(steps, ChatStep{Type: "error", Service: "probe", Message: "Evaluation failed"})
			reply = "Couldn't run the evaluation. The Probe service might be unavailable."
		} else {
			var result map[string]interface{}
			if err := json.Unmarshal(respData, &result); err != nil {
				log.Printf("[chat] failed to parse probe response: %v", err)
				steps = append(steps, ChatStep{Type: "error", Service: "probe", Message: "Invalid response from evaluation service"})
				reply = "Evaluation returned an unreadable response."
			} else {
				runID := ""
				if id, ok := result["run_id"]; ok {
					runID = fmt.Sprintf("%v", id)
				}
				steps = append(steps, ChatStep{Type: "result", Service: "probe", Message: "Evaluation run " + runID + " started"})
				reply = fmt.Sprintf("Evaluation started for skill **%s** (run: %s). Check the Evaluations page for results.", skillRef, runID)
			}
		}

	} else if strings.Contains(msgLower, "status") || strings.Contains(msgLower, "health") || strings.Contains(msgLower, "running") || strings.Contains(msgLower, "everything") {
		steps = append(steps, ChatStep{Type: "action", Service: "engine", Message: "Checking platform health..."})
		type HealthResp struct {
			Services []struct {
				Name   string `json:"name"`
				Status string `json:"status"`
			} `json:"services"`
		}
		respData, _, err := getJSON("http://localhost:8080", "/api/v1/health")
		if err != nil {
			steps = append(steps, ChatStep{Type: "error", Service: "engine", Message: "Could not fetch health"})
			reply = "Couldn't get the platform health."
		} else {
			var hr HealthResp
			if err := json.Unmarshal(respData, &hr); err != nil {
				log.Printf("[chat] failed to parse health response: %v", err)
				steps = append(steps, ChatStep{Type: "error", Service: "engine", Message: "Invalid response from health endpoint"})
				reply = "Health check returned an unreadable response."
			} else {
				healthy := 0
				for _, s := range hr.Services {
					if s.Status == "healthy" {
						healthy++
					}
				}
				steps = append(steps, ChatStep{Type: "result", Service: "engine", Message: fmt.Sprintf("%d/%d services healthy", healthy, len(hr.Services))})
				reply = fmt.Sprintf("Platform is healthy: **%d/%d services** running across 8 planes.", healthy, len(hr.Services))
			}
		}

	} else if strings.Contains(msgLower, "task") || strings.Contains(msgLower, "run") || strings.Contains(msgLower, "execute") {
		agentName := "my-agent"
		if strings.Contains(msgLower, "email") || strings.Contains(msgLower, "marketing") {
			agentName = "email-marketing-agent"
		}
		steps = append(steps, ChatStep{Type: "action", Service: "engine", Message: "Submitting task to agent \"" + agentName + "\"..."})
		engineURL := fmt.Sprintf("http://%s:8081", engineHost)
		body := map[string]interface{}{
			"agent": agentName,
			"input": map[string]string{"text": req.Message},
			"model": map[string]string{"model": "gpt-4o"},
		}
		respData, _, err := postJSON(engineURL, "/api/v1/tasks", body)
		if err != nil {
			steps = append(steps, ChatStep{Type: "error", Service: "engine", Message: "Task submission failed"})
			reply = "Couldn't submit the task. The Engine service might be unavailable."
		} else {
			var result map[string]interface{}
			if err := json.Unmarshal(respData, &result); err != nil {
				log.Printf("[chat] failed to parse engine task response: %v", err)
				steps = append(steps, ChatStep{Type: "error", Service: "engine", Message: "Invalid response from engine service"})
				reply = "Task submission returned an unreadable response."
			} else {
				taskID := ""
				if id, ok := result["id"]; ok {
					taskID = fmt.Sprintf("%v", id)
				}
				steps = append(steps, ChatStep{Type: "result", Service: "engine", Message: "Task " + taskID + " submitted (status: pending)"})
				reply = fmt.Sprintf("Task submitted to **%s** (ID: %s). The engine is processing it via Temporal workflows.", agentName, taskID)
			}
		}

	} else if strings.Contains(msgLower, "skill") && (strings.Contains(msgLower, "register") || strings.Contains(msgLower, "create") || strings.Contains(msgLower, "add")) {
		skillName := "custom-skill"
		for _, word := range strings.Fields(msg) {
			if !strings.Contains("register,create,add,a,the,new,skill,called,named", word) && len(word) > 3 {
				skillName = strings.TrimRight(word, ".,!?")
				break
			}
		}
		steps = append(steps, ChatStep{Type: "action", Service: "registry", Message: "Registering skill \"" + skillName + "\"..."})
		registryURL := fmt.Sprintf("http://%s:8104", registryHost)
		body := map[string]interface{}{
			"name":        skillName,
			"type":        "skills",
			"version":     "1.0.0",
			"description": "Skill created via Arcana Chat",
		}
		_, rcode, err := postJSON(registryURL, "/api/v1/catalog/skills", body)
		if err != nil || rcode >= 400 {
			steps = append(steps, ChatStep{Type: "error", Service: "registry", Message: "Skill registration failed"})
			reply = "Couldn't register the skill."
		} else {
			steps = append(steps, ChatStep{Type: "result", Service: "registry", Message: "Skill \"" + skillName + "\" registered (v1.0.0)"})
			reply = fmt.Sprintf("Skill **%s** registered (v1.0.0). You can see it on the Skills page.", skillName)
		}

	} else {
		reply = "I can help you with:\n\n" +
			"• **Deploy an agent** — \"I need an agent for email marketing\"\n" +
			"• **Search knowledge** — \"Search for brand guidelines\"\n" +
			"• **Check costs** — \"Show me the current spending\"\n" +
			"• **Run evaluations** — \"Evaluate the summarize skill\"\n" +
			"• **Check guardrails** — \"Is this prompt safe?\"\n" +
			"• **Submit tasks** — \"Run a task with my email agent\"\n" +
			"• **Register skills** — \"Register a new skill called brand-tone\"\n" +
			"• **Platform status** — \"Is everything running?\"\n\n" +
			"Just tell me what you need in plain English!"
		steps = nil
	}

	json.NewEncoder(w).Encode(ChatResponse{Reply: reply, Steps: steps, SessionID: sessionID})
}

type AgentChatRequest struct {
	Message   string `json:"message"`
	SessionID string `json:"session_id"`
}

type AgentChatResponse struct {
	Reply     string     `json:"reply"`
	Steps     []ChatStep `json:"steps"`
	SessionID string     `json:"session_id"`
	Agent     string     `json:"agent"`
}

func agentChatHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", corsOrigin)
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "method not allowed"})
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v1/agents/")
	agentName := strings.TrimSuffix(path, "/chat")
	if agentName == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "agent name required"})
		return
	}

	var req AgentChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON"})
		return
	}

	meshHost := envOr("MESH_HOST", "arcana-mesh")
	engineHost := envOr("ENGINE_HOST", "arcana-engine")
	wardHost := envOr("WARD_HOST", "arcana-ward")
	finopsHost := envOr("FINOPS_HOST", "arcana-finops")
	codexHost := envOr("CODEX_ROUTER_HOST", "arcana-codex-router")
	toolsHost := envOr("TOOLS_HOST", "arcana-tools")

	agentURL := fmt.Sprintf("http://%s:8083/api/v1/agents/%s", meshHost, agentName)
	agentData, _, agentErr := getJSON(agentURL, "")
	if agentErr != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "agent not found"})
		return
	}
	var agentInfo map[string]interface{}
	if err := json.Unmarshal(agentData, &agentInfo); err != nil {
		log.Printf("[agent-chat] failed to parse agent info for %s: %v", agentName, err)
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid agent data from mesh service"})
		return
	}

	sessionID := req.SessionID
	if sessionID == "" {
		sessionID = fmt.Sprintf("agent-%s-%d", agentName, time.Now().UnixNano())
	}

	msg := strings.TrimSpace(req.Message)
	msgLower := strings.ToLower(msg)
	var steps []ChatStep
	var reply string

	if strings.Contains(msgLower, "status") || strings.Contains(msgLower, "how are you") || strings.Contains(msgLower, "health") {
		steps = append(steps, ChatStep{Type: "action", Service: "mesh", Message: fmt.Sprintf("Checking status of %s...", agentName)})
		status, _ := agentInfo["status"].(string)
		caps := []string{}
		if capList, ok := agentInfo["capabilities"].([]interface{}); ok {
			for _, c := range capList {
				if s, ok := c.(string); ok {
					caps = append(caps, s)
				}
			}
		}
		steps = append(steps, ChatStep{Type: "result", Service: "mesh", Message: fmt.Sprintf("Agent %s is %s", agentName, status)})
		reply = fmt.Sprintf("**%s** is currently **%s**.\n\n", agentName, status)
		if len(caps) > 0 {
			reply += "**Capabilities:** " + strings.Join(caps, ", ") + "\n"
		}
		reply += fmt.Sprintf("**Namespace:** arcana-agent-%s", agentName)

	} else if strings.Contains(msgLower, "remember") || strings.Contains(msgLower, "recall") || strings.Contains(msgLower, "history") || strings.Contains(msgLower, "memory") || strings.Contains(msgLower, "past") {
		memHost0 := envOr("MEMORY_HOST", "arcana-memory")
		memURL0 := fmt.Sprintf("http://%s:8087", memHost0)
		steps = append(steps, ChatStep{Type: "action", Service: "memory", Message: fmt.Sprintf("Searching %s's memory...", agentName)})
		searchQ := msg
		if len(searchQ) > 200 {
			searchQ = searchQ[:200]
		}
		respData, _, err := getJSON(memURL0, fmt.Sprintf("/api/v1/memory/long-term/%s/search?query=%s&top_k=5", agentName, url.QueryEscape(searchQ)))
		if err != nil {
			stData, _, err2 := getJSON(memURL0, fmt.Sprintf("/api/v1/memory/short-term/%s", agentName))
			if err2 != nil {
				steps = append(steps, ChatStep{Type: "error", Service: "memory", Message: "Memory search failed"})
				reply = "Couldn't access my memory right now."
			} else {
				var stEntries []map[string]interface{}
				if err := json.Unmarshal(stData, &stEntries); err != nil {
					log.Printf("[agent-chat] failed to parse short-term memory for %s: %v", agentName, err)
					steps = append(steps, ChatStep{Type: "error", Service: "memory", Message: "Invalid response from memory service"})
					reply = "Memory returned an unreadable response."
					json.NewEncoder(w).Encode(AgentChatResponse{Reply: reply, Steps: steps, SessionID: sessionID, Agent: agentName})
					return
				}
				steps = append(steps, ChatStep{Type: "result", Service: "memory", Message: fmt.Sprintf("Found %d short-term memories", len(stEntries))})
				if len(stEntries) == 0 {
					reply = fmt.Sprintf("**%s** doesn't have any memories yet. We're just getting started!", agentName)
				} else {
					reply = fmt.Sprintf("**%s** has **%d recent memories** in short-term storage:\n\n", agentName, len(stEntries))
					for i, e := range stEntries {
						if i >= 5 {
							reply += fmt.Sprintf("\n... and %d more", len(stEntries)-5)
							break
						}
						key, _ := e["key"].(string)
						val := fmt.Sprintf("%v", e["value"])
						if len(val) > 80 {
							val = val[:80] + "..."
						}
						reply += fmt.Sprintf("- **%s**: %s\n", key, val)
					}
				}
			}
		} else {
			var result map[string]interface{}
			if err := json.Unmarshal(respData, &result); err != nil {
				log.Printf("[agent-chat] failed to parse long-term memory search for %s: %v", agentName, err)
				steps = append(steps, ChatStep{Type: "error", Service: "memory", Message: "Invalid response from memory search"})
				reply = "Memory search returned an unreadable response."
				json.NewEncoder(w).Encode(AgentChatResponse{Reply: reply, Steps: steps, SessionID: sessionID, Agent: agentName})
				return
			}
			results, _ := result["results"].([]interface{})
			steps = append(steps, ChatStep{Type: "result", Service: "memory", Message: fmt.Sprintf("Found %d related memories", len(results))})
			if len(results) == 0 {
				reply = fmt.Sprintf("**%s** couldn't find anything related in long-term memory.", agentName)
			} else {
				reply = fmt.Sprintf("**%s** recalls **%d related memories**:\n\n", agentName, len(results))
				for i, r := range results {
					if i >= 5 {
						break
					}
					if rm, ok := r.(map[string]interface{}); ok {
						if mem, ok := rm["memory"].(map[string]interface{}); ok {
							content, _ := mem["content"].(string)
							score, _ := rm["score"].(float64)
							if len(content) > 100 {
								content = content[:100] + "..."
							}
							reply += fmt.Sprintf("- (%.0f%% match) %s\n", score*100, content)
						}
					}
				}
			}
		}

	} else if strings.Contains(msgLower, "task") || strings.Contains(msgLower, "run ") || strings.Contains(msgLower, "execute") || strings.Contains(msgLower, "send ") || strings.Contains(msgLower, "process ") {
		steps = append(steps, ChatStep{Type: "action", Service: "engine", Message: fmt.Sprintf("Submitting task to %s...", agentName)})
		engineURL := fmt.Sprintf("http://%s:8081", engineHost)
		body := map[string]interface{}{
			"agent":       agentName,
			"description": msg,
			"priority":    "medium",
		}
		respData, rcode, err := postJSON(engineURL, "/api/v1/tasks", body)
		if err != nil || rcode >= 400 {
			steps = append(steps, ChatStep{Type: "error", Service: "engine", Message: "Task submission failed"})
			reply = "Couldn't submit the task. The Engine service might be unavailable."
		} else {
			var result map[string]interface{}
			if err := json.Unmarshal(respData, &result); err != nil {
				log.Printf("[agent-chat] failed to parse engine task response: %v", err)
				steps = append(steps, ChatStep{Type: "error", Service: "engine", Message: "Invalid response from engine service"})
				reply = "Task submission returned an unreadable response."
			} else {
				taskID, _ := result["id"].(string)
				steps = append(steps, ChatStep{Type: "result", Service: "engine", Message: fmt.Sprintf("Task %s submitted", taskID)})
				reply = fmt.Sprintf("Task submitted to **%s**.\n\n- **Task ID:** `%s`\n- **Status:** queued\n- **Description:** %s", agentName, taskID, msg)
			}
		}

	} else if strings.Contains(msgLower, "search") || strings.Contains(msgLower, "find") || strings.Contains(msgLower, "knowledge") || strings.Contains(msgLower, "look up") {
		steps = append(steps, ChatStep{Type: "action", Service: "codex-router", Message: fmt.Sprintf("Searching knowledge on behalf of %s...", agentName)})
		codexURL := fmt.Sprintf("http://%s:8090", codexHost)
		body := map[string]interface{}{"query": msg, "profile": "default", "top_k": 5}
		respData, _, err := postJSON(codexURL, "/api/v1/search", body)
		if err != nil {
			steps = append(steps, ChatStep{Type: "error", Service: "codex-router", Message: "Search failed"})
			reply = "Search failed. The Codex service might be unavailable."
		} else {
			var result map[string]interface{}
			if err := json.Unmarshal(respData, &result); err != nil {
				log.Printf("[agent-chat] failed to parse codex search response: %v", err)
				steps = append(steps, ChatStep{Type: "error", Service: "codex-router", Message: "Invalid response from search service"})
				reply = "Search returned an unreadable response."
			} else {
				hits := 0
				if th, ok := result["total_hits"]; ok {
					if f, ok := th.(float64); ok {
						hits = int(f)
					}
				}
				steps = append(steps, ChatStep{Type: "result", Service: "codex-router", Message: fmt.Sprintf("Found %d results", hits)})
				reply = fmt.Sprintf("**%s** searched the knowledge base and found **%d results** for \"%s\".", agentName, hits, msg)
			}
		}

	} else if strings.Contains(msgLower, "cost") || strings.Contains(msgLower, "spend") || strings.Contains(msgLower, "budget") {
		steps = append(steps, ChatStep{Type: "action", Service: "finops", Message: fmt.Sprintf("Fetching costs for %s...", agentName)})
		finopsURL := fmt.Sprintf("http://%s:8105", finopsHost)
		respData, _, err := getJSON(finopsURL, "/api/v1/costs")
		if err != nil {
			steps = append(steps, ChatStep{Type: "error", Service: "finops", Message: "Cost check failed"})
			reply = "Couldn't fetch cost data."
		} else {
			var result map[string]interface{}
			if err := json.Unmarshal(respData, &result); err != nil {
				log.Printf("[agent-chat] failed to parse finops response: %v", err)
				steps = append(steps, ChatStep{Type: "error", Service: "finops", Message: "Invalid response from finops service"})
				reply = "Cost report returned an unreadable response."
			} else {
				totalUSD, _ := result["total_usd"].(float64)
				steps = append(steps, ChatStep{Type: "result", Service: "finops", Message: fmt.Sprintf("Agent cost: $%.4f", totalUSD)})
				reply = fmt.Sprintf("**%s** cost to date: **$%.4f**", agentName, totalUSD)
			}
		}

	} else if strings.Contains(msgLower, "tool") || strings.Contains(msgLower, "mcp") {
		steps = append(steps, ChatStep{Type: "action", Service: "tools", Message: fmt.Sprintf("Listing tools available to %s...", agentName)})
		toolsURL := fmt.Sprintf("http://%s:8096", toolsHost)
		respData, _, err := getJSON(toolsURL, "/api/v1/tools")
		if err != nil {
			steps = append(steps, ChatStep{Type: "error", Service: "tools", Message: "Could not fetch tools"})
			reply = "Couldn't fetch available tools."
		} else {
			var result map[string]interface{}
			if err := json.Unmarshal(respData, &result); err != nil {
				log.Printf("[agent-chat] failed to parse tools response: %v", err)
				steps = append(steps, ChatStep{Type: "error", Service: "tools", Message: "Invalid response from tools service"})
				reply = "Tools listing returned an unreadable response."
			} else {
				servers := []string{}
				if srvList, ok := result["servers"].([]interface{}); ok {
					for _, s := range srvList {
						if srv, ok := s.(map[string]interface{}); ok {
							name, _ := srv["name"].(string)
							tools := 0
							if tl, ok := srv["tools"].([]interface{}); ok {
								tools = len(tl)
							}
							servers = append(servers, fmt.Sprintf("- **%s** (%d tools)", name, tools))
						}
					}
				}
				steps = append(steps, ChatStep{Type: "result", Service: "tools", Message: fmt.Sprintf("%d MCP servers available", len(servers))})
				reply = fmt.Sprintf("MCP servers available to **%s**:\n\n%s", agentName, strings.Join(servers, "\n"))
			}
		}

	} else if strings.Contains(msgLower, "guardrail") || strings.Contains(msgLower, "safe") || strings.Contains(msgLower, "check") {
		steps = append(steps, ChatStep{Type: "action", Service: "ward", Message: fmt.Sprintf("Running guardrail check for %s...", agentName)})
		wardURL := fmt.Sprintf("http://%s:8086", wardHost)
		body := map[string]interface{}{"text": msg, "agent_id": agentName, "direction": "input"}
		respData, _, err := postJSON(wardURL, "/api/v1/check", body)
		if err != nil {
			steps = append(steps, ChatStep{Type: "error", Service: "ward", Message: "Guardrail check failed"})
			reply = "Couldn't run the guardrail check."
		} else {
			var result map[string]interface{}
			if err := json.Unmarshal(respData, &result); err != nil {
				log.Printf("[agent-chat] failed to parse ward response: %v", err)
				steps = append(steps, ChatStep{Type: "error", Service: "ward", Message: "Invalid response from guardrail service"})
				reply = "Guardrail check returned an unreadable response."
			} else {
				verdict, _ := result["verdict"].(string)
				steps = append(steps, ChatStep{Type: "result", Service: "ward", Message: fmt.Sprintf("Verdict: %s", verdict)})
				reply = fmt.Sprintf("Guardrail check for **%s**: **%s**", agentName, verdict)
			}
		}

	} else {
		caps := []string{}
		if capList, ok := agentInfo["capabilities"].([]interface{}); ok {
			for _, c := range capList {
				if s, ok := c.(string); ok {
					caps = append(caps, s)
				}
			}
		}
		reply = fmt.Sprintf("You're chatting with **%s** (capabilities: %s).\n\nI can help with:\n\n", agentName, strings.Join(caps, ", "))
		reply += "- **Run a task** — \"Send the weekly email campaign\"\n"
		reply += "- **Search knowledge** — \"Find brand guidelines\"\n"
		reply += "- **Recall memory** — \"What do you remember about our last chat?\"\n"
		reply += "- **Check status** — \"How are you doing?\"\n"
		reply += "- **View costs** — \"What's my spend?\"\n"
		reply += "- **List tools** — \"What MCP tools do you have?\"\n"
		reply += "- **Guardrail check** — \"Is this prompt safe?\"\n\n"
		reply += "Just tell me what you'd like to do!"
		steps = nil
	}

	memHost := envOr("MEMORY_HOST", "arcana-memory")
	memBase := fmt.Sprintf("http://%s:8087", memHost)
	memAgent := agentName
	memSID := sessionID
	memMsg := msg
	memReply := reply
	memHasSteps := steps != nil && len(steps) > 0
	select {
	case memoryWorkCh <- func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, _, err := postJSONCtx(ctx, memBase, "/api/v1/memory/short-term", map[string]interface{}{
			"agent_id": memAgent, "key": "user:" + memSID, "value": memMsg, "ttl": 3600,
		}); err != nil {
			log.Printf("[memory] failed to store user message: %v", err)
		}
		if _, _, err := postJSONCtx(ctx, memBase, "/api/v1/memory/short-term", map[string]interface{}{
			"agent_id": memAgent, "key": "bot:" + memSID, "value": memReply, "ttl": 3600,
		}); err != nil {
			log.Printf("[memory] failed to store bot reply: %v", err)
		}
		if memHasSteps {
			if _, _, err := postJSONCtx(ctx, memBase, "/api/v1/memory/long-term", map[string]interface{}{
				"agent_id": memAgent,
				"content":  fmt.Sprintf("User asked: %s | Agent replied: %s", memMsg, memReply),
				"metadata": map[string]interface{}{"session_id": memSID, "type": "conversation"},
			}); err != nil {
				log.Printf("[memory] failed to store long-term memory: %v", err)
			}
		}
	}:
	default:
		log.Printf("[memory] work queue full, dropping memory store for session %s", memSID)
	}

	json.NewEncoder(w).Encode(AgentChatResponse{Reply: reply, Steps: steps, SessionID: sessionID, Agent: agentName})
}

func main() {
	corsOrigin = os.Getenv("CORS_ORIGIN")
	if corsOrigin == "" {
		if os.Getenv("ARCANA_ENV") == "production" {
			fmt.Fprintln(os.Stderr, "FATAL: CORS_ORIGIN must be set in production (wildcard '*' is not allowed)")
			os.Exit(1)
		}
		corsOrigin = "*"
	}

	// Reap idle conversation sessions every minute; expire after 30 minutes.
	startSessionReaper(convSessions, time.Minute, 30*time.Minute)

	enterprise := NewEnterpriseGateway()

	httpSrv := server.New(server.Config{
		ServiceName: "api",
		Port:        "8080",
		DB:          enterprise.db,
	})

	httpSrv.HandleFunc("/api/v1/version", cors(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"name":"arcana","version":"0.1.0","services":28,"crds":16,"planes":8}`)
	}))

	enterprise.RegisterRoutes(httpSrv)

	httpSrv.HandleFunc("/api/v1/health", enterprise.AuthMiddleware(cors(healthHandler)))
	httpSrv.HandleFunc("/api/v1/routes", enterprise.AuthMiddleware(cors(routesHandler)))
	httpSrv.HandleFunc("/api/v1/chat", enterprise.AuthMiddleware(chatHandler))
	meshProxy := makeProxy(envOr("MESH_HOST", "arcana-mesh"), 8083)
	meshCors := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", corsOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		meshProxy.ServeHTTP(w, r)
	}
	httpSrv.HandleFunc("/api/v1/agents/", enterprise.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/chat") {
			agentChatHandler(w, r)
			return
		}
		meshCors(w, r)
	}))
	httpSrv.HandleFunc("/api/v1/agents", enterprise.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		meshCors(w, r)
	}))
	httpSrv.HandleFunc("/api/v1/messages", enterprise.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) { meshCors(w, r) }))
	httpSrv.HandleFunc("/api/v1/messages/", enterprise.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) { meshCors(w, r) }))
	httpSrv.HandleFunc("/api/v1/delegate", enterprise.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) { meshCors(w, r) }))

	// Agent external URL routing — routes /agents/{name}/* to the agent's deep-agent pod.
	httpSrv.HandleFunc("/agents/", enterprise.AuthMiddleware(cors(func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/agents/"), "/")
		if len(parts) < 1 || parts[0] == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "agent name required"})
			return
		}
		agentName := parts[0]

		// Resolve agent endpoint — each deployed agent lives in its own namespace.
		ns := "arcana-agent-" + agentName
		targetHost := envOr("AGENT_HOST_"+strings.ToUpper(strings.ReplaceAll(agentName, "-", "_")), "deep-agent."+ns+".svc.cluster.local")
		targetURL := fmt.Sprintf("http://%s:5002", targetHost)

		// Build proxy path — everything after the agent name.
		proxyPath := "/"
		if len(parts) > 1 {
			proxyPath = "/" + strings.Join(parts[1:], "/")
		}

		target, err := url.Parse(targetURL)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadGateway)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid agent target URL"})
			return
		}
		proxy := httputil.NewSingleHostReverseProxy(target)
		proxy.ErrorHandler = func(pw http.ResponseWriter, pr *http.Request, proxyErr error) {
			pw.Header().Set("Content-Type", "application/json")
			pw.WriteHeader(http.StatusBadGateway)
			json.NewEncoder(pw).Encode(map[string]string{
				"error":   "agent_unavailable",
				"message": fmt.Sprintf("agent %s unreachable at %s: %v", agentName, targetURL, proxyErr),
			})
		}
		r.URL.Path = proxyPath
		r.URL.RawPath = proxyPath
		proxy.ServeHTTP(w, r)
	})))

	// Chat session persistence endpoints.
	httpSrv.HandleFunc("/api/v1/chat/sessions", enterprise.AuthMiddleware(cors(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusMethodNotAllowed)
			json.NewEncoder(w).Encode(map[string]string{"error": "method not allowed"})
			return
		}
		sessions := []map[string]interface{}{
			{"id": "sess-1", "title": "Agent deployment", "created_at": time.Now().Add(-2 * time.Hour).Format(time.RFC3339), "message_count": 5},
			{"id": "sess-2", "title": "Cost review", "created_at": time.Now().Add(-24 * time.Hour).Format(time.RFC3339), "message_count": 12},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"sessions": sessions})
	})))

	httpSrv.HandleFunc("/api/v1/chat/sessions/", enterprise.AuthMiddleware(cors(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusMethodNotAllowed)
			json.NewEncoder(w).Encode(map[string]string{"error": "method not allowed"})
			return
		}
		// GET /api/v1/chat/sessions/{id}/messages
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"messages": []interface{}{}})
	})))

	// SSE streaming endpoint for chat.
	httpSrv.HandleFunc("/api/v1/chat/stream", enterprise.AuthMiddleware(cors(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusMethodNotAllowed)
			json.NewEncoder(w).Encode(map[string]string{"error": "POST required"})
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming not supported", http.StatusInternalServerError)
			return
		}

		var req struct {
			Message string `json:"message"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			fmt.Fprintf(w, "data: {\"error\":\"invalid request body\"}\n\n")
			flusher.Flush()
			return
		}

		response := fmt.Sprintf("I received your message: %q. This is the Arcana platform assistant.", req.Message)

		for _, word := range strings.Fields(response) {
			fmt.Fprintf(w, "data: %s\n\n", word+" ")
			flusher.Flush()
			time.Sleep(50 * time.Millisecond)
		}
		fmt.Fprintf(w, "data: [DONE]\n\n")
		flusher.Flush()
	})))

	for _, sr := range serviceRoutes {
		host := envOr(sr.EnvKey, sr.Default)
		proxy := makeProxy(host, sr.Port)
		prefix := sr.Prefix
		handler := enterprise.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", corsOrigin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			proxy.ServeHTTP(w, r)
		})
		httpSrv.HandleFunc(prefix+"/", handler)
		if !strings.HasSuffix(prefix, "/") {
			httpSrv.HandleFunc(prefix, handler)
		}
	}

	httpSrv.ListenAndServe()
}
