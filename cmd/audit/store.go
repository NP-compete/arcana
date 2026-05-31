package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

type AuditStore struct {
	db       *sql.DB
	lastHash string
}

func NewAuditStore(db *sql.DB) *AuditStore {
	// Recover lastHash from the most recent entry.
	var lastHash string
	err := db.QueryRow(`SELECT entry_hash FROM audit_log ORDER BY id DESC LIMIT 1`).Scan(&lastHash)
	if err != nil {
		lastHash = "genesis"
	}
	return &AuditStore{db: db, lastHash: lastHash}
}

func (s *AuditStore) computeHash(entry AuditEntry) string {
	data, _ := json.Marshal(struct {
		ID          string    `json:"id"`
		Timestamp   time.Time `json:"timestamp"`
		Tenant      string    `json:"tenant"`
		Agent       string    `json:"agent"`
		Action      string    `json:"action"`
		Tool        string    `json:"tool"`
		InputHash   string    `json:"input_hash"`
		OutputHash  string    `json:"output_hash"`
		WardVerdict string    `json:"ward_verdict"`
		OPAVerdict  string    `json:"opa_verdict"`
		CostUSD     float64   `json:"cost_usd"`
		Tokens      int64     `json:"tokens"`
		Model       string    `json:"model"`
		SessionID   string    `json:"session_id"`
		WorkflowID  string    `json:"workflow_id"`
		PrevHash    string    `json:"prev_hash"`
	}{entry.ID, entry.Timestamp, entry.Tenant, entry.Agent, entry.Action, entry.Tool,
		entry.InputHash, entry.OutputHash, entry.WardVerdict, entry.OPAVerdict,
		entry.CostUSD, entry.Tokens, entry.Model, entry.SessionID, entry.WorkflowID, entry.PrevHash})
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// buildDetail serializes extra audit fields that don't map directly to audit_log columns
// into a JSON string stored in the detail column.
func buildDetail(entry AuditEntry) string {
	detail := map[string]interface{}{
		"input_hash":  entry.InputHash,
		"output_hash": entry.OutputHash,
		"ward_verdict": entry.WardVerdict,
		"opa_verdict":  entry.OPAVerdict,
		"cost_usd":    entry.CostUSD,
		"tokens":      entry.Tokens,
		"model":       entry.Model,
		"session_id":  entry.SessionID,
		"workflow_id": entry.WorkflowID,
		"tool":        entry.Tool,
	}
	b, _ := json.Marshal(detail)
	return string(b)
}

// parseDetail reconstructs extra audit fields from the detail JSON column.
func parseDetail(detail string, entry *AuditEntry) {
	if detail == "" {
		return
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(detail), &m); err != nil {
		return
	}
	if v, ok := m["input_hash"].(string); ok {
		entry.InputHash = v
	}
	if v, ok := m["output_hash"].(string); ok {
		entry.OutputHash = v
	}
	if v, ok := m["ward_verdict"].(string); ok {
		entry.WardVerdict = v
	}
	if v, ok := m["opa_verdict"].(string); ok {
		entry.OPAVerdict = v
	}
	if v, ok := m["cost_usd"].(float64); ok {
		entry.CostUSD = v
	}
	if v, ok := m["tokens"].(float64); ok {
		entry.Tokens = int64(v)
	}
	if v, ok := m["model"].(string); ok {
		entry.Model = v
	}
	if v, ok := m["session_id"].(string); ok {
		entry.SessionID = v
	}
	if v, ok := m["workflow_id"].(string); ok {
		entry.WorkflowID = v
	}
	if v, ok := m["tool"].(string); ok {
		entry.Tool = v
	}
}

func (s *AuditStore) Append(req AppendAuditRequest) AuditEntry {
	ts := req.Timestamp
	if ts.IsZero() {
		ts = time.Now().UTC()
	}

	entry := AuditEntry{
		Timestamp:   ts,
		Tenant:      req.Tenant,
		Agent:       req.Agent,
		Action:      req.Action,
		Tool:        req.Tool,
		InputHash:   req.InputHash,
		OutputHash:  req.OutputHash,
		WardVerdict: req.WardVerdict,
		OPAVerdict:  req.OPAVerdict,
		CostUSD:     req.CostUSD,
		Tokens:      req.Tokens,
		Model:       req.Model,
		SessionID:   req.SessionID,
		WorkflowID:  req.WorkflowID,
		PrevHash:    s.lastHash,
	}
	// Use a temporary ID for hash computation, then replace with DB-assigned BIGSERIAL.
	entry.ID = fmt.Sprintf("temp-%d", ts.UnixNano())
	entry.EntryHash = s.computeHash(entry)

	detail := buildDetail(entry)

	var dbID int64
	err := s.db.QueryRow(`
		INSERT INTO audit_log (actor, tenant, action, resource, detail, ip, entry_hash, prev_hash, created_at)
		VALUES ($1, $2, $3, $4, $5, '', $6, $7, $8)
		RETURNING id`,
		entry.Agent, entry.Tenant, entry.Action, entry.Tool, detail,
		entry.EntryHash, entry.PrevHash, entry.Timestamp,
	).Scan(&dbID)
	if err != nil {
		log.Printf("Append: insert audit_log: %v", err)
		return entry
	}

	entry.ID = fmt.Sprintf("%d", dbID)
	s.lastHash = entry.EntryHash
	return entry
}

func (s *AuditStore) Get(id string) (*AuditEntry, bool) {
	row := s.db.QueryRow(`
		SELECT id, actor, tenant, action, resource, detail, entry_hash, prev_hash, created_at
		FROM audit_log WHERE id::text = $1`, id)

	entry, err := scanAuditRow(row)
	if err == sql.ErrNoRows {
		return nil, false
	}
	if err != nil {
		log.Printf("Get %s: %v", id, err)
		return nil, false
	}
	return entry, true
}

func (s *AuditStore) Query(q AuditQuery) ([]AuditEntry, int) {
	where, args := buildAuditWhere(q)

	// Get total count first.
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM audit_log %s`, where)
	var total int
	err := s.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		log.Printf("Query count: %v", err)
		return []AuditEntry{}, 0
	}

	limit := q.Limit
	if limit <= 0 {
		limit = 100
	}
	offset := q.Offset
	if offset < 0 {
		offset = 0
	}

	dataQuery := fmt.Sprintf(`
		SELECT id, actor, tenant, action, resource, detail, entry_hash, prev_hash, created_at
		FROM audit_log %s
		ORDER BY id
		LIMIT $%d OFFSET $%d`, where, len(args)+1, len(args)+2)
	args = append(args, limit, offset)

	rows, err := s.db.Query(dataQuery, args...)
	if err != nil {
		log.Printf("Query: %v", err)
		return []AuditEntry{}, total
	}
	defer rows.Close()

	result := make([]AuditEntry, 0)
	for rows.Next() {
		entry, err := scanAuditRowFromRows(rows)
		if err != nil {
			log.Printf("Query scan: %v", err)
			continue
		}
		result = append(result, *entry)
	}
	return result, total
}

func (s *AuditStore) Stats() AuditStats {
	stats := AuditStats{
		ByAgent:   make(map[string]int64),
		ByVerdict: make(map[string]int64),
	}

	err := s.db.QueryRow(`SELECT COUNT(*) FROM audit_log`).Scan(&stats.TotalEntries)
	if err != nil {
		log.Printf("Stats count: %v", err)
		return stats
	}

	// Aggregate by agent (actor).
	rows, err := s.db.Query(`SELECT actor, COUNT(*) FROM audit_log GROUP BY actor`)
	if err != nil {
		log.Printf("Stats by_agent: %v", err)
		return stats
	}
	defer rows.Close()
	for rows.Next() {
		var agent string
		var count int64
		if err := rows.Scan(&agent, &count); err != nil {
			continue
		}
		stats.ByAgent[agent] = count
	}

	// Total cost from detail JSON. This scans all rows; acceptable for summary endpoint.
	costRows, err := s.db.Query(`SELECT detail FROM audit_log WHERE detail != '' AND detail IS NOT NULL`)
	if err != nil {
		log.Printf("Stats cost: %v", err)
		return stats
	}
	defer costRows.Close()
	for costRows.Next() {
		var detail string
		if err := costRows.Scan(&detail); err != nil {
			continue
		}
		var m map[string]interface{}
		if err := json.Unmarshal([]byte(detail), &m); err != nil {
			continue
		}
		if c, ok := m["cost_usd"].(float64); ok {
			stats.TotalCost += c
		}
		if v, ok := m["ward_verdict"].(string); ok && v != "" {
			stats.ByVerdict[v]++
		}
	}

	return stats
}

func (s *AuditStore) VerifyChain() (bool, string) {
	rows, err := s.db.Query(`
		SELECT id, actor, tenant, action, resource, detail, entry_hash, prev_hash, created_at
		FROM audit_log ORDER BY id`)
	if err != nil {
		return false, fmt.Sprintf("query error: %v", err)
	}
	defer rows.Close()

	prev := "genesis"
	idx := 0
	for rows.Next() {
		entry, err := scanAuditRowFromRows(rows)
		if err != nil {
			return false, fmt.Sprintf("scan error at entry %d: %v", idx, err)
		}
		if entry.PrevHash != prev {
			return false, fmt.Sprintf("chain broken at entry %d: expected prev_hash %s, got %s", idx, prev, entry.PrevHash)
		}
		expected := s.computeHash(*entry)
		if entry.EntryHash != expected {
			return false, fmt.Sprintf("hash mismatch at entry %d", idx)
		}
		prev = entry.EntryHash
		idx++
	}
	return true, ""
}

func buildAuditWhere(q AuditQuery) (string, []interface{}) {
	var conditions []string
	var args []interface{}
	idx := 1

	if q.Agent != "" {
		conditions = append(conditions, fmt.Sprintf("actor = $%d", idx))
		args = append(args, q.Agent)
		idx++
	}
	if q.Action != "" {
		conditions = append(conditions, fmt.Sprintf("action = $%d", idx))
		args = append(args, q.Action)
		idx++
	}
	if q.Since != "" {
		if t, err := time.Parse(time.RFC3339, q.Since); err == nil {
			conditions = append(conditions, fmt.Sprintf("created_at >= $%d", idx))
			args = append(args, t)
			idx++
		}
	}
	if q.Until != "" {
		if t, err := time.Parse(time.RFC3339, q.Until); err == nil {
			conditions = append(conditions, fmt.Sprintf("created_at <= $%d", idx))
			args = append(args, t)
			idx++
		}
	}
	if q.Verdict != "" {
		conditions = append(conditions, fmt.Sprintf("detail LIKE $%d", idx))
		args = append(args, "%"+q.Verdict+"%")
		idx++
	}

	if len(conditions) == 0 {
		return "", args
	}
	where := "WHERE "
	for i, c := range conditions {
		if i > 0 {
			where += " AND "
		}
		where += c
	}
	return where, args
}

func scanAuditRow(row *sql.Row) (*AuditEntry, error) {
	var (
		id        int64
		actor     string
		tenant    string
		action    string
		resource  sql.NullString
		detail    sql.NullString
		entryHash string
		prevHash  string
		createdAt time.Time
	)
	err := row.Scan(&id, &actor, &tenant, &action, &resource, &detail, &entryHash, &prevHash, &createdAt)
	if err != nil {
		return nil, err
	}
	return buildAuditEntry(id, actor, tenant, action, resource, detail, entryHash, prevHash, createdAt), nil
}

func scanAuditRowFromRows(rows *sql.Rows) (*AuditEntry, error) {
	var (
		id        int64
		actor     string
		tenant    string
		action    string
		resource  sql.NullString
		detail    sql.NullString
		entryHash string
		prevHash  string
		createdAt time.Time
	)
	err := rows.Scan(&id, &actor, &tenant, &action, &resource, &detail, &entryHash, &prevHash, &createdAt)
	if err != nil {
		return nil, err
	}
	return buildAuditEntry(id, actor, tenant, action, resource, detail, entryHash, prevHash, createdAt), nil
}

func buildAuditEntry(id int64, actor, tenant, action string, resource, detail sql.NullString, entryHash, prevHash string, createdAt time.Time) *AuditEntry {
	entry := &AuditEntry{
		ID:        fmt.Sprintf("%d", id),
		Timestamp: createdAt,
		Tenant:    tenant,
		Agent:     actor,
		Action:    action,
		PrevHash:  prevHash,
		EntryHash: entryHash,
	}
	if resource.Valid {
		entry.Tool = resource.String
	}
	if detail.Valid {
		parseDetail(detail.String, entry)
	}
	return entry
}
