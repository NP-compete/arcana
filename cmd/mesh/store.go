package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type MeshStore struct {
	db *sql.DB
}

func NewMeshStore(db *sql.DB) *MeshStore {
	return &MeshStore{db: db}
}

func (s *MeshStore) RegisterAgent(tenant, name string, capabilities, protocols []string, status AgentStatus, agentType AgentType, deepCfg *DeepAgentConfig) *MeshAgent {
	now := time.Now().UTC()
	if agentType == "" {
		agentType = AgentTypeStandard
	}
	if capabilities == nil {
		capabilities = []string{}
	}
	if protocols == nil {
		protocols = []string{}
	}

	var deepConfig interface{}
	if deepCfg != nil {
		b, err := json.Marshal(deepCfg)
		if err != nil {
			log.Printf("RegisterAgent: marshal deep_config for %s: %v", name, err)
		} else {
			deepConfig = b
		}
	}

	_, err := s.db.Exec(`
		INSERT INTO agents (tenant, name, agent_type, capabilities, protocols, status, deep_config, registered_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (tenant, name) DO UPDATE SET
			agent_type = EXCLUDED.agent_type,
			capabilities = EXCLUDED.capabilities,
			protocols = EXCLUDED.protocols,
			status = EXCLUDED.status,
			deep_config = EXCLUDED.deep_config,
			registered_at = EXCLUDED.registered_at`,
		tenant, name, string(agentType), pq.Array(capabilities), pq.Array(protocols), string(status), deepConfig, now,
	)
	if err != nil {
		log.Printf("RegisterAgent: upsert %s: %v", name, err)
	}

	return &MeshAgent{
		Name:         name,
		Tenant:       tenant,
		AgentType:    agentType,
		Capabilities: capabilities,
		Protocols:    protocols,
		Status:       status,
		RegisteredAt: now,
		DeepConfig:   deepCfg,
	}
}

func (s *MeshStore) GetAgent(tenant, name string) (*MeshAgent, bool) {
	agent, err := scanAgent(s.db.QueryRow(`
		SELECT tenant, name, agent_type, capabilities, protocols, status, deep_config, registered_at
		FROM agents WHERE tenant = $1 AND name = $2`, tenant, name))
	if err == sql.ErrNoRows {
		return nil, false
	}
	if err != nil {
		log.Printf("GetAgent %s/%s: %v", tenant, name, err)
		return nil, false
	}
	return agent, true
}

func (s *MeshStore) DeleteAgent(tenant, name string) bool {
	res, err := s.db.Exec(`DELETE FROM agents WHERE tenant = $1 AND name = $2`, tenant, name)
	if err != nil {
		log.Printf("DeleteAgent %s/%s: %v", tenant, name, err)
		return false
	}
	n, err := res.RowsAffected()
	if err != nil {
		log.Printf("DeleteAgent %s/%s rows affected: %v", tenant, name, err)
		return false
	}
	return n > 0
}

func (s *MeshStore) ListAgents(tenant string) []MeshAgent {
	rows, err := s.db.Query(`
		SELECT tenant, name, agent_type, capabilities, protocols, status, deep_config, registered_at
		FROM agents WHERE tenant = $1 ORDER BY registered_at`, tenant)
	if err != nil {
		log.Printf("ListAgents %s: %v", tenant, err)
		return []MeshAgent{}
	}
	defer rows.Close()

	result := make([]MeshAgent, 0)
	for rows.Next() {
		agent, err := scanAgentRow(rows)
		if err != nil {
			log.Printf("ListAgents scan: %v", err)
			continue
		}
		result = append(result, *agent)
	}
	if err := rows.Err(); err != nil {
		log.Printf("ListAgents rows: %v", err)
	}
	return result
}

func (s *MeshStore) SendMessage(tenant, from, to string, payload map[string]interface{}, protocol string) (*Message, error) {
	exists, err := s.agentExists(tenant, to)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errAgentNotFound
	}

	if payload == nil {
		payload = make(map[string]interface{})
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	msg := Message{
		ID:        uuid.New().String(),
		Tenant:    tenant,
		From:      from,
		To:        to,
		Payload:   payload,
		Protocol:  protocol,
		CreatedAt: now,
		Delivered: false,
	}

	_, err = s.db.Exec(`
		INSERT INTO messages (id, tenant, from_agent, to_agent, payload, protocol, delivered, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, FALSE, $7)`,
		msg.ID, msg.Tenant, msg.From, msg.To, payloadJSON, msg.Protocol, msg.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

func (s *MeshStore) GetPendingMessages(tenant, agentName string) []Message {
	rows, err := s.db.Query(`
		SELECT id, tenant, from_agent, to_agent, payload, protocol, delivered, created_at
		FROM messages
		WHERE tenant = $1 AND to_agent = $2 AND delivered = FALSE
		ORDER BY created_at`, tenant, agentName)
	if err != nil {
		log.Printf("GetPendingMessages %s/%s: %v", tenant, agentName, err)
		return []Message{}
	}
	defer rows.Close()

	return scanMessages(rows)
}

func (s *MeshStore) MarkMessagesDelivered(tenant, agentName string) {
	_, err := s.db.Exec(`
		UPDATE messages SET delivered = TRUE
		WHERE tenant = $1 AND to_agent = $2 AND delivered = FALSE`, tenant, agentName)
	if err != nil {
		log.Printf("MarkMessagesDelivered %s/%s: %v", tenant, agentName, err)
	}
}

func (s *MeshStore) CreateDelegation(tenant, fromAgent, toAgent, taskType string, payload map[string]interface{}) (*DelegationTask, error) {
	exists, err := s.agentExists(tenant, toAgent)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errAgentNotFound
	}

	if payload == nil {
		payload = make(map[string]interface{})
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	task := &DelegationTask{
		ID:        uuid.New().String(),
		Tenant:    tenant,
		FromAgent: fromAgent,
		ToAgent:   toAgent,
		TaskType:  taskType,
		Payload:   payload,
		Status:    DelegationPending,
		CreatedAt: now,
		UpdatedAt: now,
	}

	_, err = s.db.Exec(`
		INSERT INTO delegations (id, tenant, from_agent, to_agent, task_type, payload, result, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NULL, $7, $8, $9)`,
		task.ID, task.Tenant, task.FromAgent, task.ToAgent, task.TaskType, payloadJSON, string(task.Status), task.CreatedAt, task.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return task, nil
}

func (s *MeshStore) GetDelegation(id string) (*DelegationTask, bool) {
	task, err := scanDelegation(s.db.QueryRow(`
		SELECT id, tenant, from_agent, to_agent, task_type, payload, result, status, created_at, updated_at
		FROM delegations WHERE id = $1`, id))
	if err == sql.ErrNoRows {
		return nil, false
	}
	if err != nil {
		log.Printf("GetDelegation %s: %v", id, err)
		return nil, false
	}
	return task, true
}

func (s *MeshStore) UpdateDelegation(id string, fn func(*DelegationTask)) bool {
	task, err := scanDelegation(s.db.QueryRow(`
		SELECT id, tenant, from_agent, to_agent, task_type, payload, result, status, created_at, updated_at
		FROM delegations WHERE id = $1`, id))
	if err == sql.ErrNoRows {
		return false
	}
	if err != nil {
		log.Printf("UpdateDelegation select %s: %v", id, err)
		return false
	}

	fn(task)
	task.UpdatedAt = time.Now().UTC()

	payloadJSON, err := json.Marshal(task.Payload)
	if err != nil {
		log.Printf("UpdateDelegation marshal payload %s: %v", id, err)
		return false
	}

	var resultJSON interface{}
	if task.Result != nil {
		b, err := json.Marshal(task.Result)
		if err != nil {
			log.Printf("UpdateDelegation marshal result %s: %v", id, err)
			return false
		}
		resultJSON = b
	}

	res, err := s.db.Exec(`
		UPDATE delegations
		SET status = $1, payload = $2, result = $3, updated_at = $4
		WHERE id = $5`,
		string(task.Status), payloadJSON, resultJSON, task.UpdatedAt, id,
	)
	if err != nil {
		log.Printf("UpdateDelegation update %s: %v", id, err)
		return false
	}
	n, err := res.RowsAffected()
	if err != nil {
		log.Printf("UpdateDelegation rows affected %s: %v", id, err)
		return false
	}
	return n > 0
}

func (s *MeshStore) CountMessagesToAgent(tenant, agentName string) int {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM messages WHERE tenant = $1 AND to_agent = $2`, tenant, agentName).Scan(&count)
	if err != nil {
		log.Printf("CountMessagesToAgent %s/%s: %v", tenant, agentName, err)
		return 0
	}
	return count
}

func (s *MeshStore) CountDelegationsForAgent(tenant, agentName string) int {
	var count int
	err := s.db.QueryRow(`
		SELECT COUNT(*) FROM delegations
		WHERE tenant = $1 AND (from_agent = $2 OR to_agent = $2)`, tenant, agentName).Scan(&count)
	if err != nil {
		log.Printf("CountDelegationsForAgent %s/%s: %v", tenant, agentName, err)
		return 0
	}
	return count
}

func (s *MeshStore) UpdateAgentStatus(tenant, name string, status AgentStatus) {
	_, err := s.db.Exec(`UPDATE agents SET status = $1 WHERE tenant = $2 AND name = $3`, string(status), tenant, name)
	if err != nil {
		log.Printf("UpdateAgentStatus %s/%s: %v", tenant, name, err)
	}
}

func (s *MeshStore) agentExists(tenant, name string) (bool, error) {
	var one int
	err := s.db.QueryRow(`SELECT 1 FROM agents WHERE tenant = $1 AND name = $2`, tenant, name).Scan(&one)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func scanAgent(row *sql.Row) (*MeshAgent, error) {
	var (
		tenant       string
		name         string
		agentType    string
		capabilities []string
		protocols    []string
		status       string
		deepConfig   []byte
		registeredAt time.Time
	)
	err := row.Scan(&tenant, &name, &agentType, pq.Array(&capabilities), pq.Array(&protocols), &status, &deepConfig, &registeredAt)
	if err != nil {
		return nil, err
	}
	return buildAgent(tenant, name, agentType, capabilities, protocols, status, deepConfig, registeredAt)
}

func scanAgentRow(rows *sql.Rows) (*MeshAgent, error) {
	var (
		tenant       string
		name         string
		agentType    string
		capabilities []string
		protocols    []string
		status       string
		deepConfig   []byte
		registeredAt time.Time
	)
	err := rows.Scan(&tenant, &name, &agentType, pq.Array(&capabilities), pq.Array(&protocols), &status, &deepConfig, &registeredAt)
	if err != nil {
		return nil, err
	}
	return buildAgent(tenant, name, agentType, capabilities, protocols, status, deepConfig, registeredAt)
}

func buildAgent(tenant, name, agentType string, capabilities, protocols []string, status string, deepConfig []byte, registeredAt time.Time) (*MeshAgent, error) {
	agent := &MeshAgent{
		Name:         name,
		Tenant:       tenant,
		AgentType:    AgentType(agentType),
		Capabilities: capabilities,
		Protocols:    protocols,
		Status:       AgentStatus(status),
		RegisteredAt: registeredAt,
	}
	if len(deepConfig) > 0 {
		var cfg DeepAgentConfig
		if err := json.Unmarshal(deepConfig, &cfg); err != nil {
			return nil, err
		}
		agent.DeepConfig = &cfg
	}
	return agent, nil
}

func scanMessages(rows *sql.Rows) []Message {
	messages := make([]Message, 0)
	for rows.Next() {
		msg, err := scanMessageRow(rows)
		if err != nil {
			log.Printf("scanMessages: %v", err)
			continue
		}
		messages = append(messages, *msg)
	}
	if err := rows.Err(); err != nil {
		log.Printf("scanMessages rows: %v", err)
	}
	return messages
}

func scanMessageRow(rows *sql.Rows) (*Message, error) {
	var (
		id        string
		tenant    string
		fromAgent string
		toAgent   string
		payload   []byte
		protocol  sql.NullString
		delivered bool
		createdAt time.Time
	)
	err := rows.Scan(&id, &tenant, &fromAgent, &toAgent, &payload, &protocol, &delivered, &createdAt)
	if err != nil {
		return nil, err
	}
	return buildMessage(id, tenant, fromAgent, toAgent, payload, protocol, delivered, createdAt)
}

func buildMessage(id, tenant, fromAgent, toAgent string, payload []byte, protocol sql.NullString, delivered bool, createdAt time.Time) (*Message, error) {
	msg := &Message{
		ID:        id,
		Tenant:    tenant,
		From:      fromAgent,
		To:        toAgent,
		Payload:   map[string]interface{}{},
		CreatedAt: createdAt,
		Delivered: delivered,
	}
	if protocol.Valid {
		msg.Protocol = protocol.String
	}
	if len(payload) > 0 {
		if err := json.Unmarshal(payload, &msg.Payload); err != nil {
			return nil, err
		}
	}
	return msg, nil
}

func scanDelegation(row *sql.Row) (*DelegationTask, error) {
	var (
		id        string
		tenant    string
		fromAgent string
		toAgent   string
		taskType  sql.NullString
		payload   []byte
		result    []byte
		status    string
		createdAt time.Time
		updatedAt time.Time
	)
	err := row.Scan(&id, &tenant, &fromAgent, &toAgent, &taskType, &payload, &result, &status, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	return buildDelegation(id, tenant, fromAgent, toAgent, taskType, payload, result, status, createdAt, updatedAt)
}

func buildDelegation(id, tenant, fromAgent, toAgent string, taskType sql.NullString, payload, result []byte, status string, createdAt, updatedAt time.Time) (*DelegationTask, error) {
	task := &DelegationTask{
		ID:        id,
		Tenant:    tenant,
		FromAgent: fromAgent,
		ToAgent:   toAgent,
		Payload:   map[string]interface{}{},
		Status:    DelegationStatus(status),
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
	if taskType.Valid {
		task.TaskType = taskType.String
	}
	if len(payload) > 0 {
		if err := json.Unmarshal(payload, &task.Payload); err != nil {
			return nil, err
		}
	}
	if len(result) > 0 {
		task.Result = map[string]interface{}{}
		if err := json.Unmarshal(result, &task.Result); err != nil {
			return nil, err
		}
	}
	return task, nil
}

// --- Health monitoring store methods ---

func (s *MeshStore) RecordHealthEvent(tenant, agentName, eventType string, restartCount, readyReplicas, desiredReplicas int, failureReason, podPhase string) {
	_, err := s.db.Exec(`
		INSERT INTO agent_health_events (tenant, agent_name, event_type, restart_count, ready_replicas, desired_replicas, failure_reason, pod_phase)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		tenant, agentName, eventType, restartCount, readyReplicas, desiredReplicas, failureReason, podPhase)
	if err != nil {
		log.Printf("RecordHealthEvent %s/%s: %v", tenant, agentName, err)
	}
}

func (s *MeshStore) UpdateAgentHealth(tenant, name string, restartCount int, podPhase, failureReason string, isHealthy bool) {
	now := time.Now().UTC()
	if isHealthy {
		_, err := s.db.Exec(`
			UPDATE agents SET restart_count = $1, pod_phase = $2, last_healthy_at = $3, last_failure_reason = ''
			WHERE tenant = $4 AND name = $5`,
			restartCount, podPhase, now, tenant, name)
		if err != nil {
			log.Printf("UpdateAgentHealth %s/%s: %v", tenant, name, err)
		}
	} else {
		_, err := s.db.Exec(`
			UPDATE agents SET restart_count = $1, pod_phase = $2, last_failure_at = $3, last_failure_reason = $4
			WHERE tenant = $5 AND name = $6`,
			restartCount, podPhase, now, failureReason, tenant, name)
		if err != nil {
			log.Printf("UpdateAgentHealth %s/%s: %v", tenant, name, err)
		}
	}
}

func (s *MeshStore) GetAgentHealthSummary(tenant, name string) *AgentHealthSummary {
	var (
		restartCount      int
		lastHealthyAt     sql.NullTime
		lastFailureAt     sql.NullTime
		lastFailureReason sql.NullString
		podPhase          sql.NullString
	)
	err := s.db.QueryRow(`
		SELECT COALESCE(restart_count,0), last_healthy_at, last_failure_at, last_failure_reason, pod_phase
		FROM agents WHERE tenant = $1 AND name = $2`, tenant, name).
		Scan(&restartCount, &lastHealthyAt, &lastFailureAt, &lastFailureReason, &podPhase)
	if err != nil {
		log.Printf("GetAgentHealthSummary %s/%s: %v", tenant, name, err)
		return nil
	}
	summary := &AgentHealthSummary{
		AgentName:    name,
		RestartCount: restartCount,
		PodPhase:     podPhase.String,
	}
	if lastHealthyAt.Valid {
		summary.LastHealthyAt = &lastHealthyAt.Time
	}
	if lastFailureAt.Valid {
		summary.LastFailureAt = &lastFailureAt.Time
	}
	if lastFailureReason.Valid {
		summary.LastFailureReason = lastFailureReason.String
	}
	if podPhase.String == "Running" && restartCount == 0 {
		summary.Status = "healthy"
	} else if lastFailureReason.Valid && lastFailureReason.String != "" {
		summary.Status = "unhealthy"
	} else if restartCount > 0 {
		summary.Status = "degraded"
	} else {
		summary.Status = "unknown"
	}
	return summary
}

func (s *MeshStore) GetAgentHealthEvents(tenant, name string, limit int) []AgentHealthEvent {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.Query(`
		SELECT id, tenant, agent_name, event_type, restart_count, ready_replicas, desired_replicas, failure_reason, pod_phase, created_at
		FROM agent_health_events
		WHERE tenant = $1 AND agent_name = $2
		ORDER BY created_at DESC LIMIT $3`, tenant, name, limit)
	if err != nil {
		log.Printf("GetAgentHealthEvents %s/%s: %v", tenant, name, err)
		return []AgentHealthEvent{}
	}
	defer rows.Close()
	events := make([]AgentHealthEvent, 0)
	for rows.Next() {
		var e AgentHealthEvent
		if err := rows.Scan(&e.ID, &e.Tenant, &e.AgentName, &e.EventType, &e.RestartCount,
			&e.ReadyReplicas, &e.DesiredReplicas, &e.FailureReason, &e.PodPhase, &e.CreatedAt); err != nil {
			log.Printf("GetAgentHealthEvents scan: %v", err)
			continue
		}
		events = append(events, e)
	}
	return events
}

func (s *MeshStore) GetAgentsHealthOverview(tenant string) AgentsHealthOverview {
	var overview AgentsHealthOverview
	s.db.QueryRow(`SELECT COUNT(*) FROM agents WHERE tenant = $1`, tenant).Scan(&overview.TotalAgents)
	s.db.QueryRow(`SELECT COUNT(*) FROM agents WHERE tenant = $1 AND pod_phase = 'Running' AND COALESCE(restart_count,0) = 0`, tenant).Scan(&overview.HealthyAgents)
	overview.UnhealthyAgents = overview.TotalAgents - overview.HealthyAgents
	s.db.QueryRow(`SELECT COALESCE(SUM(restart_count),0) FROM agents WHERE tenant = $1`, tenant).Scan(&overview.TotalRestarts)
	return overview
}

// --- Playground session store methods ---

func (s *MeshStore) CreatePlaygroundSession(session *PlaygroundSession) {
	configJSON, _ := json.Marshal(session.AgentConfig)
	_, err := s.db.Exec(`
		INSERT INTO playground_sessions (id, tenant, agent_name, agent_config, budget_limit, status, created_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		session.ID, session.Tenant, session.AgentName, configJSON,
		session.BudgetLimit, session.Status, session.CreatedAt, session.ExpiresAt)
	if err != nil {
		log.Printf("CreatePlaygroundSession: %v", err)
	}
}

func (s *MeshStore) GetPlaygroundSession(tenant, id string) *PlaygroundSession {
	var (
		agentConfig []byte
		session     PlaygroundSession
	)
	err := s.db.QueryRow(`
		SELECT id, tenant, agent_name, agent_config, budget_limit, budget_used, tokens_used, message_count, status, created_at, expires_at
		FROM playground_sessions WHERE id = $1 AND tenant = $2`, id, tenant).
		Scan(&session.ID, &session.Tenant, &session.AgentName, &agentConfig,
			&session.BudgetLimit, &session.BudgetUsed, &session.TokensUsed,
			&session.MsgCount, &session.Status, &session.CreatedAt, &session.ExpiresAt)
	if err != nil {
		return nil
	}
	if len(agentConfig) > 0 {
		json.Unmarshal(agentConfig, &session.AgentConfig)
	}
	if time.Now().After(session.ExpiresAt) && session.Status == "active" {
		s.EndPlaygroundSession(tenant, id)
		session.Status = "expired"
	}
	return &session
}

func (s *MeshStore) UpdatePlaygroundUsage(tenant, id string, tokens int, cost float64) {
	_, err := s.db.Exec(`
		UPDATE playground_sessions
		SET tokens_used = tokens_used + $1, budget_used = budget_used + $2, message_count = message_count + 1
		WHERE id = $3 AND tenant = $4`,
		tokens, cost, id, tenant)
	if err != nil {
		log.Printf("UpdatePlaygroundUsage: %v", err)
	}
}

func (s *MeshStore) EndPlaygroundSession(tenant, id string) {
	_, err := s.db.Exec(`UPDATE playground_sessions SET status = 'ended' WHERE id = $1 AND tenant = $2`, id, tenant)
	if err != nil {
		log.Printf("EndPlaygroundSession: %v", err)
	}
}

var errAgentNotFound = &agentNotFoundError{}

type agentNotFoundError struct{}

func (e *agentNotFoundError) Error() string {
	return "agent not found"
}
