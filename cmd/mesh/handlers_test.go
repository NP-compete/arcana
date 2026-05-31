package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// In-memory mock store — replaces *MeshStore for tests that cannot connect to
// PostgreSQL. It implements the exact method signatures the Server handlers
// call, so we can inject it via a thin interface wrapper.
// ---------------------------------------------------------------------------

type mockStore struct {
	mu          sync.RWMutex
	agents      map[string]*MeshAgent // key: "tenant/name"
	messages    []Message
	delegations map[string]*DelegationTask
	nextMsgID   int
	nextDelID   int
}

func newMockStore() *mockStore {
	return &mockStore{
		agents:      make(map[string]*MeshAgent),
		messages:    make([]Message, 0),
		delegations: make(map[string]*DelegationTask),
	}
}

func agentKey(tenant, name string) string {
	return tenant + "/" + name
}

func (m *mockStore) RegisterAgent(tenant, name string, capabilities, protocols []string, status AgentStatus, agentType AgentType, deepCfg *DeepAgentConfig) *MeshAgent {
	m.mu.Lock()
	defer m.mu.Unlock()
	if capabilities == nil {
		capabilities = []string{}
	}
	if protocols == nil {
		protocols = []string{}
	}
	agent := &MeshAgent{
		Name:         name,
		Tenant:       tenant,
		AgentType:    agentType,
		Capabilities: capabilities,
		Protocols:    protocols,
		Status:       status,
		RegisteredAt: time.Now().UTC(),
		DeepConfig:   deepCfg,
	}
	m.agents[agentKey(tenant, name)] = agent
	return agent
}

func (m *mockStore) GetAgent(tenant, name string) (*MeshAgent, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	a, ok := m.agents[agentKey(tenant, name)]
	if !ok {
		return nil, false
	}
	copy := *a
	return &copy, true
}

func (m *mockStore) DeleteAgent(tenant, name string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := agentKey(tenant, name)
	if _, ok := m.agents[key]; !ok {
		return false
	}
	delete(m.agents, key)
	return true
}

func (m *mockStore) ListAgents(tenant string) []MeshAgent {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]MeshAgent, 0, len(m.agents))
	for _, a := range m.agents {
		if a.Tenant == tenant {
			result = append(result, *a)
		}
	}
	return result
}

func (m *mockStore) SendMessage(tenant, from, to string, payload map[string]interface{}, protocol string) (*Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.agents[agentKey(tenant, to)]; !ok {
		return nil, &agentNotFoundError{}
	}
	if payload == nil {
		payload = make(map[string]interface{})
	}
	m.nextMsgID++
	msg := Message{
		ID:        fmt.Sprintf("msg-%d", m.nextMsgID),
		Tenant:    tenant,
		From:      from,
		To:        to,
		Payload:   payload,
		Protocol:  protocol,
		CreatedAt: time.Now().UTC(),
		Delivered: false,
	}
	m.messages = append(m.messages, msg)
	return &msg, nil
}

func (m *mockStore) GetPendingMessages(tenant, agentName string) []Message {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []Message
	for _, msg := range m.messages {
		if msg.Tenant == tenant && msg.To == agentName && !msg.Delivered {
			result = append(result, msg)
		}
	}
	if result == nil {
		return []Message{}
	}
	return result
}

func (m *mockStore) CreateDelegation(tenant, fromAgent, toAgent, taskType string, payload map[string]interface{}) (*DelegationTask, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.agents[agentKey(tenant, toAgent)]; !ok {
		return nil, &agentNotFoundError{}
	}
	if payload == nil {
		payload = make(map[string]interface{})
	}
	m.nextDelID++
	now := time.Now().UTC()
	task := &DelegationTask{
		ID:        fmt.Sprintf("del-%d", m.nextDelID),
		Tenant:    tenant,
		FromAgent: fromAgent,
		ToAgent:   toAgent,
		TaskType:  taskType,
		Payload:   payload,
		Status:    DelegationPending,
		CreatedAt: now,
		UpdatedAt: now,
	}
	m.delegations[task.ID] = task
	return task, nil
}

func (m *mockStore) UpdateDelegation(id string, fn func(*DelegationTask)) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	task, ok := m.delegations[id]
	if !ok {
		return false
	}
	fn(task)
	task.UpdatedAt = time.Now().UTC()
	return true
}

func (m *mockStore) CountMessagesToAgent(tenant, agentName string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	count := 0
	for _, msg := range m.messages {
		if msg.Tenant == tenant && msg.To == agentName {
			count++
		}
	}
	return count
}

func (m *mockStore) CountDelegationsForAgent(tenant, agentName string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	count := 0
	for _, d := range m.delegations {
		if d.Tenant == tenant && (d.FromAgent == agentName || d.ToAgent == agentName) {
			count++
		}
	}
	return count
}

// ---------------------------------------------------------------------------
// testServer builds a *Server backed by the mock store. The handlers call
// s.store.<method>, and since the real MeshStore is a concrete type, we need
// to ensure our Server uses the mockStore. Because Server.store is *MeshStore
// (concrete), we cannot substitute directly. Instead we create a thin wrapper
// Server that routes each handler through a local mux, duplicating the handler
// registration from main.go but pointed at the mock.
// ---------------------------------------------------------------------------

type testServer struct {
	mock *mockStore
	mux  *http.ServeMux
}

func newTestServer() *testServer {
	ms := newMockStore()
	mux := http.NewServeMux()

	// Register handlers that mirror main.go's routing, but backed by the mock.
	mux.HandleFunc("/api/v1/agents/register", func(w http.ResponseWriter, r *http.Request) {
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
		agent := ms.RegisterAgent(tenant, req.Name, req.Capabilities, req.Protocols, status, agentType, req.DeepConfig)
		writeJSON(w, http.StatusCreated, agent)
	})

	mux.HandleFunc("/api/v1/agents", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		tenant := extractTenant(r)
		agents := ms.ListAgents(tenant)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"agents": agents,
			"total":  len(agents),
		})
	})

	mux.HandleFunc("/api/v1/agents/", func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/api/v1/agents/")
		if name == "" || strings.Contains(name, "/") {
			writeError(w, http.StatusBadRequest, "agent name required")
			return
		}
		tenant := extractTenant(r)
		if r.Method == http.MethodDelete {
			if !ms.DeleteAgent(tenant, name) {
				writeError(w, http.StatusNotFound, "agent not found")
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "name": name})
			return
		}
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		agent, ok := ms.GetAgent(tenant, name)
		if !ok {
			writeError(w, http.StatusNotFound, "agent not found")
			return
		}
		writeJSON(w, http.StatusOK, agent)
	})

	mux.HandleFunc("/api/v1/messages", func(w http.ResponseWriter, r *http.Request) {
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
		msg, err := ms.SendMessage(tenant, req.From, req.To, req.Payload, req.Protocol)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, msg)
	})

	mux.HandleFunc("/api/v1/delegate", func(w http.ResponseWriter, r *http.Request) {
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
		task, err := ms.CreateDelegation(tenant, req.FromAgent, req.ToAgent, req.TaskType, req.Payload)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusAccepted, task)
	})

	return &testServer{mock: ms, mux: mux}
}

func (ts *testServer) do(req *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	ts.mux.ServeHTTP(rec, req)
	return rec
}

// ---------------------------------------------------------------------------
// Helper — register a test agent via the mock store directly.
// ---------------------------------------------------------------------------

func (ts *testServer) seedAgent(name string) {
	ts.seedAgentInTenant("default", name)
}

func (ts *testServer) seedAgentInTenant(tenant, name string) {
	ts.mock.RegisterAgent(tenant, name, []string{"search"}, []string{"a2a"}, AgentStatusActive, AgentTypeStandard, nil)
}

// ---------------------------------------------------------------------------
// A2. Tests
// ---------------------------------------------------------------------------

func TestRegisterAgent(t *testing.T) {
	ts := newTestServer()
	body := `{"name":"test-agent","capabilities":["search","summarize"],"protocols":["a2a"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rec := ts.do(req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var agent MeshAgent
	json.NewDecoder(rec.Body).Decode(&agent)
	if agent.Name != "test-agent" {
		t.Errorf("name: got %q, want %q", agent.Name, "test-agent")
	}
	if agent.Tenant != "default" {
		t.Errorf("tenant: got %q, want %q", agent.Tenant, "default")
	}
	if len(agent.Capabilities) != 2 {
		t.Errorf("capabilities: got %d, want 2", len(agent.Capabilities))
	}
	if agent.Status != AgentStatusActive {
		t.Errorf("status: got %q, want %q", agent.Status, AgentStatusActive)
	}
}

func TestRegisterAgent_WithTenantHeader(t *testing.T) {
	ts := newTestServer()
	body := `{"name":"test-agent","capabilities":["search"],"protocols":["a2a"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Arcana-Tenant", "acme-corp")

	rec := ts.do(req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var agent MeshAgent
	json.NewDecoder(rec.Body).Decode(&agent)
	if agent.Tenant != "acme-corp" {
		t.Errorf("tenant: got %q, want %q", agent.Tenant, "acme-corp")
	}
}

func TestRegisterAgent_InvalidJSON(t *testing.T) {
	ts := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents/register", strings.NewReader("{bad json"))
	req.Header.Set("Content-Type", "application/json")

	rec := ts.do(req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	var body map[string]string
	json.NewDecoder(rec.Body).Decode(&body)
	if body["error"] != "invalid JSON body" {
		t.Errorf("error: got %q, want %q", body["error"], "invalid JSON body")
	}
}

func TestRegisterAgent_MissingName(t *testing.T) {
	ts := newTestServer()
	body := `{"capabilities":["search"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rec := ts.do(req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	var resp map[string]string
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["error"] != "name is required" {
		t.Errorf("error: got %q, want %q", resp["error"], "name is required")
	}
}

func TestListAgents(t *testing.T) {
	ts := newTestServer()
	ts.seedAgent("agent-a")
	ts.seedAgent("agent-b")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil)
	rec := ts.do(req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp struct {
		Agents []MeshAgent `json:"agents"`
		Total  int         `json:"total"`
	}
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Total != 2 {
		t.Errorf("total: got %d, want 2", resp.Total)
	}
	if len(resp.Agents) != 2 {
		t.Errorf("agents: got %d, want 2", len(resp.Agents))
	}
}

func TestListAgents_TenantIsolation(t *testing.T) {
	ts := newTestServer()
	ts.seedAgentInTenant("tenant-a", "agent-1")
	ts.seedAgentInTenant("tenant-a", "agent-2")
	ts.seedAgentInTenant("tenant-b", "agent-3")

	// List agents for tenant-a — should only see 2
	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil)
	req.Header.Set("X-Arcana-Tenant", "tenant-a")
	rec := ts.do(req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var resp struct {
		Agents []MeshAgent `json:"agents"`
		Total  int         `json:"total"`
	}
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Total != 2 {
		t.Errorf("tenant-a total: got %d, want 2", resp.Total)
	}

	// List agents for tenant-b — should only see 1
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil)
	req2.Header.Set("X-Arcana-Tenant", "tenant-b")
	rec2 := ts.do(req2)

	var resp2 struct {
		Agents []MeshAgent `json:"agents"`
		Total  int         `json:"total"`
	}
	json.NewDecoder(rec2.Body).Decode(&resp2)
	if resp2.Total != 1 {
		t.Errorf("tenant-b total: got %d, want 1", resp2.Total)
	}
}

func TestListAgents_Empty(t *testing.T) {
	ts := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil)
	rec := ts.do(req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var resp struct {
		Agents []MeshAgent `json:"agents"`
		Total  int         `json:"total"`
	}
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Total != 0 {
		t.Errorf("total: got %d, want 0", resp.Total)
	}
}

func TestGetAgent(t *testing.T) {
	ts := newTestServer()
	ts.seedAgent("my-agent")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents/my-agent", nil)
	rec := ts.do(req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}
	var agent MeshAgent
	json.NewDecoder(rec.Body).Decode(&agent)
	if agent.Name != "my-agent" {
		t.Errorf("name: got %q, want %q", agent.Name, "my-agent")
	}
}

func TestGetAgent_TenantIsolation(t *testing.T) {
	ts := newTestServer()
	ts.seedAgentInTenant("tenant-a", "shared-name")

	// Access from tenant-b should not find tenant-a's agent
	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents/shared-name", nil)
	req.Header.Set("X-Arcana-Tenant", "tenant-b")
	rec := ts.do(req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestGetAgent_NotFound(t *testing.T) {
	ts := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents/does-not-exist", nil)
	rec := ts.do(req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
	var body map[string]string
	json.NewDecoder(rec.Body).Decode(&body)
	if body["error"] != "agent not found" {
		t.Errorf("error: got %q, want %q", body["error"], "agent not found")
	}
}

func TestDeleteAgent(t *testing.T) {
	ts := newTestServer()
	ts.seedAgent("to-delete")

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/agents/to-delete", nil)
	rec := ts.do(req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}
	var body map[string]string
	json.NewDecoder(rec.Body).Decode(&body)
	if body["status"] != "deleted" {
		t.Errorf("status: got %q, want %q", body["status"], "deleted")
	}

	// Confirm agent is gone.
	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/agents/to-delete", nil)
	getRec := ts.do(getReq)
	if getRec.Code != http.StatusNotFound {
		t.Errorf("expected 404 after deletion, got %d", getRec.Code)
	}
}

func TestDeleteAgent_TenantIsolation(t *testing.T) {
	ts := newTestServer()
	ts.seedAgentInTenant("tenant-a", "agent-x")

	// Attempt to delete from tenant-b should fail
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/agents/agent-x", nil)
	req.Header.Set("X-Arcana-Tenant", "tenant-b")
	rec := ts.do(req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d; body: %s", rec.Code, rec.Body.String())
	}

	// Agent should still exist in tenant-a
	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/agents/agent-x", nil)
	getReq.Header.Set("X-Arcana-Tenant", "tenant-a")
	getRec := ts.do(getReq)
	if getRec.Code != http.StatusOK {
		t.Errorf("expected 200 (agent still exists in tenant-a), got %d", getRec.Code)
	}
}

func TestDeleteAgent_NotFound(t *testing.T) {
	ts := newTestServer()

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/agents/ghost", nil)
	rec := ts.do(req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestSendMessage(t *testing.T) {
	ts := newTestServer()
	ts.seedAgent("receiver")

	body := `{"from":"sender","to":"receiver","payload":{"text":"hello"},"protocol":"a2a"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/messages", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rec := ts.do(req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d; body: %s", rec.Code, rec.Body.String())
	}
	var msg Message
	json.NewDecoder(rec.Body).Decode(&msg)
	if msg.From != "sender" {
		t.Errorf("from: got %q, want %q", msg.From, "sender")
	}
	if msg.To != "receiver" {
		t.Errorf("to: got %q, want %q", msg.To, "receiver")
	}
	if msg.Protocol != "a2a" {
		t.Errorf("protocol: got %q, want %q", msg.Protocol, "a2a")
	}
	if msg.Tenant != "default" {
		t.Errorf("tenant: got %q, want %q", msg.Tenant, "default")
	}
}

func TestSendMessage_TenantIsolation(t *testing.T) {
	ts := newTestServer()
	ts.seedAgentInTenant("tenant-a", "receiver")

	// Sending from tenant-b should fail because receiver does not exist in tenant-b
	body := `{"from":"sender","to":"receiver","payload":{"text":"hello"},"protocol":"a2a"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/messages", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Arcana-Tenant", "tenant-b")

	rec := ts.do(req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestSendMessage_MissingFields(t *testing.T) {
	ts := newTestServer()

	body := `{"from":"sender","protocol":"a2a"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/messages", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rec := ts.do(req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestSendMessage_InvalidProtocol(t *testing.T) {
	ts := newTestServer()
	ts.seedAgent("receiver")

	body := `{"from":"sender","to":"receiver","protocol":"http"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/messages", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rec := ts.do(req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	var resp map[string]string
	json.NewDecoder(rec.Body).Decode(&resp)
	if !strings.Contains(resp["error"], "protocol must be a2a or acp") {
		t.Errorf("unexpected error: %s", resp["error"])
	}
}

func TestSendMessage_AgentNotFound(t *testing.T) {
	ts := newTestServer()

	body := `{"from":"sender","to":"nonexistent","payload":{},"protocol":"a2a"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/messages", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rec := ts.do(req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestDelegate(t *testing.T) {
	ts := newTestServer()
	ts.seedAgent("worker")

	body := `{"from_agent":"manager","to_agent":"worker","task_type":"summarize","payload":{"text":"hello"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/delegate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rec := ts.do(req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d; body: %s", rec.Code, rec.Body.String())
	}
	var task DelegationTask
	json.NewDecoder(rec.Body).Decode(&task)
	if task.FromAgent != "manager" {
		t.Errorf("from_agent: got %q, want %q", task.FromAgent, "manager")
	}
	if task.ToAgent != "worker" {
		t.Errorf("to_agent: got %q, want %q", task.ToAgent, "worker")
	}
	if task.TaskType != "summarize" {
		t.Errorf("task_type: got %q, want %q", task.TaskType, "summarize")
	}
	if task.Status != DelegationPending {
		t.Errorf("status: got %q, want %q", task.Status, DelegationPending)
	}
	if task.Tenant != "default" {
		t.Errorf("tenant: got %q, want %q", task.Tenant, "default")
	}
}

func TestDelegate_TenantIsolation(t *testing.T) {
	ts := newTestServer()
	ts.seedAgentInTenant("tenant-a", "worker")

	// Delegate from tenant-b should fail because worker does not exist in tenant-b
	body := `{"from_agent":"manager","to_agent":"worker","task_type":"summarize","payload":{}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/delegate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Arcana-Tenant", "tenant-b")

	rec := ts.do(req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestDelegate_MissingFields(t *testing.T) {
	ts := newTestServer()

	body := `{"from_agent":"manager"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/delegate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rec := ts.do(req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestDelegate_MissingTaskType(t *testing.T) {
	ts := newTestServer()
	ts.seedAgent("worker")

	body := `{"from_agent":"manager","to_agent":"worker"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/delegate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rec := ts.do(req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	var resp map[string]string
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["error"] != "task_type is required" {
		t.Errorf("error: got %q, want %q", resp["error"], "task_type is required")
	}
}

func TestDelegate_AgentNotFound(t *testing.T) {
	ts := newTestServer()

	body := `{"from_agent":"manager","to_agent":"ghost","task_type":"summarize"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/delegate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rec := ts.do(req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestDelegate_InvalidJSON(t *testing.T) {
	ts := newTestServer()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/delegate", strings.NewReader("{bad"))
	req.Header.Set("Content-Type", "application/json")

	rec := ts.do(req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestExtractTenant(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		expected string
	}{
		{name: "empty header defaults to default", header: "", expected: "default"},
		{name: "explicit tenant header", header: "acme-corp", expected: "acme-corp"},
		{name: "whitespace-only is treated as set", header: " ", expected: " "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.header != "" {
				req.Header.Set("X-Arcana-Tenant", tt.header)
			}
			got := extractTenant(req)
			if got != tt.expected {
				t.Errorf("extractTenant: got %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestSameNameDifferentTenants(t *testing.T) {
	ts := newTestServer()

	// Register same agent name in two different tenants
	body := `{"name":"shared-agent","capabilities":["search"],"protocols":["a2a"]}`

	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/agents/register", strings.NewReader(body))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("X-Arcana-Tenant", "tenant-a")
	rec1 := ts.do(req1)
	if rec1.Code != http.StatusCreated {
		t.Fatalf("tenant-a register: expected 201, got %d", rec1.Code)
	}

	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/agents/register", strings.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("X-Arcana-Tenant", "tenant-b")
	rec2 := ts.do(req2)
	if rec2.Code != http.StatusCreated {
		t.Fatalf("tenant-b register: expected 201, got %d", rec2.Code)
	}

	// Both tenants should see their own agent
	getA := httptest.NewRequest(http.MethodGet, "/api/v1/agents/shared-agent", nil)
	getA.Header.Set("X-Arcana-Tenant", "tenant-a")
	recA := ts.do(getA)
	if recA.Code != http.StatusOK {
		t.Errorf("tenant-a get: expected 200, got %d", recA.Code)
	}

	getB := httptest.NewRequest(http.MethodGet, "/api/v1/agents/shared-agent", nil)
	getB.Header.Set("X-Arcana-Tenant", "tenant-b")
	recB := ts.do(getB)
	if recB.Code != http.StatusOK {
		t.Errorf("tenant-b get: expected 200, got %d", recB.Code)
	}

	// Delete from tenant-a should not affect tenant-b
	delA := httptest.NewRequest(http.MethodDelete, "/api/v1/agents/shared-agent", nil)
	delA.Header.Set("X-Arcana-Tenant", "tenant-a")
	ts.do(delA)

	getBAfterDel := httptest.NewRequest(http.MethodGet, "/api/v1/agents/shared-agent", nil)
	getBAfterDel.Header.Set("X-Arcana-Tenant", "tenant-b")
	recBAfterDel := ts.do(getBAfterDel)
	if recBAfterDel.Code != http.StatusOK {
		t.Errorf("tenant-b get after tenant-a deletion: expected 200, got %d", recBAfterDel.Code)
	}

	getAAfterDel := httptest.NewRequest(http.MethodGet, "/api/v1/agents/shared-agent", nil)
	getAAfterDel.Header.Set("X-Arcana-Tenant", "tenant-a")
	recAAfterDel := ts.do(getAAfterDel)
	if recAAfterDel.Code != http.StatusNotFound {
		t.Errorf("tenant-a get after deletion: expected 404, got %d", recAAfterDel.Code)
	}
}
