package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
)

type SchedulerStore struct {
	db *sql.DB
}

func NewSchedulerStore(db *sql.DB) *SchedulerStore {
	return &SchedulerStore{db: db}
}

func (s *SchedulerStore) ListAgents() []AgentSchedule {
	rows, err := s.db.Query(`
		SELECT agent_name, status, snapshot_path, suspended_at, resumed_at, created_at
		FROM agent_schedules ORDER BY agent_name`)
	if err != nil {
		log.Printf("ListAgents: %v", err)
		return []AgentSchedule{}
	}
	defer rows.Close()

	result := make([]AgentSchedule, 0)
	for rows.Next() {
		a, err := scanScheduleRow(rows)
		if err != nil {
			log.Printf("ListAgents scan: %v", err)
			continue
		}
		result = append(result, *a)
	}
	if err := rows.Err(); err != nil {
		log.Printf("ListAgents rows: %v", err)
	}
	return result
}

func (s *SchedulerStore) getOrCreate(name string) (*AgentSchedule, error) {
	row := s.db.QueryRow(`
		SELECT agent_name, status, snapshot_path, suspended_at, resumed_at, created_at
		FROM agent_schedules WHERE agent_name = $1`, name)

	a, err := scanSchedule(row)
	if err == sql.ErrNoRows {
		now := time.Now().UTC()
		_, err = s.db.Exec(`
			INSERT INTO agent_schedules (agent_name, status, snapshot_path, created_at)
			VALUES ($1, $2, '', $3)`,
			name, string(AgentIdle), now,
		)
		if err != nil {
			return nil, fmt.Errorf("create schedule: %w", err)
		}
		return &AgentSchedule{
			Name:       name,
			Status:     AgentIdle,
			Priority:   PriorityStandard,
			LastActive: now,
		}, nil
	}
	if err != nil {
		return nil, err
	}
	return a, nil
}

func (s *SchedulerStore) Suspend(name string) (*AgentSchedule, *Snapshot, error) {
	a, err := s.getOrCreate(name)
	if err != nil {
		return nil, nil, err
	}
	if a.Status == AgentSuspended {
		return nil, nil, fmt.Errorf("agent already suspended")
	}

	now := time.Now().UTC()
	snapID := uuid.New().String()
	path := fmt.Sprintf("/snapshots/%s/%s.json", name, snapID)

	_, err = s.db.Exec(`
		UPDATE agent_schedules
		SET status = $1, snapshot_path = $2, suspended_at = $3
		WHERE agent_name = $4`,
		string(AgentSuspended), path, now, name,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("suspend: %w", err)
	}

	a.Status = AgentSuspended
	a.SnapshotPath = path
	a.LastActive = now

	snap := &Snapshot{
		ID:        snapID,
		AgentName: name,
		Path:      path,
		CreatedAt: now,
		SizeBytes: 4096,
	}
	return a, snap, nil
}

func (s *SchedulerStore) Resume(name string) (*AgentSchedule, error) {
	a, err := s.getOrCreate(name)
	if err != nil {
		return nil, err
	}
	if a.Status != AgentSuspended {
		return nil, fmt.Errorf("agent is not suspended")
	}

	now := time.Now().UTC()
	_, err = s.db.Exec(`
		UPDATE agent_schedules
		SET status = $1, snapshot_path = '', resumed_at = $2
		WHERE agent_name = $3`,
		string(AgentRunning), now, name,
	)
	if err != nil {
		return nil, fmt.Errorf("resume: %w", err)
	}

	a.Status = AgentRunning
	a.SnapshotPath = ""
	a.IdleSince = nil
	a.LastActive = now
	return a, nil
}

func (s *SchedulerStore) SetPriority(name string, priority PriorityClass) (*AgentSchedule, error) {
	a, err := s.getOrCreate(name)
	if err != nil {
		return nil, err
	}
	// Priority is not stored in the DB schema (agent_schedules doesn't have a priority column).
	// We update the in-memory representation only.
	a.Priority = priority
	return a, nil
}

func (s *SchedulerStore) ListSnapshots() []Snapshot {
	// Snapshots are derived from agent_schedules with non-empty snapshot_path.
	rows, err := s.db.Query(`
		SELECT agent_name, snapshot_path, suspended_at
		FROM agent_schedules
		WHERE snapshot_path != '' AND snapshot_path IS NOT NULL
		ORDER BY suspended_at DESC`)
	if err != nil {
		log.Printf("ListSnapshots: %v", err)
		return []Snapshot{}
	}
	defer rows.Close()

	result := make([]Snapshot, 0)
	for rows.Next() {
		var agentName, path string
		var suspendedAt sql.NullTime
		if err := rows.Scan(&agentName, &path, &suspendedAt); err != nil {
			log.Printf("ListSnapshots scan: %v", err)
			continue
		}
		createdAt := time.Now().UTC()
		if suspendedAt.Valid {
			createdAt = suspendedAt.Time
		}
		result = append(result, Snapshot{
			ID:        uuid.New().String(),
			AgentName: agentName,
			Path:      path,
			CreatedAt: createdAt,
			SizeBytes: 4096,
		})
	}
	return result
}

func scanSchedule(row *sql.Row) (*AgentSchedule, error) {
	var (
		name         string
		status       string
		snapshotPath string
		suspendedAt  sql.NullTime
		resumedAt    sql.NullTime
		createdAt    time.Time
	)
	err := row.Scan(&name, &status, &snapshotPath, &suspendedAt, &resumedAt, &createdAt)
	if err != nil {
		return nil, err
	}
	return buildSchedule(name, status, snapshotPath, suspendedAt, resumedAt, createdAt), nil
}

func scanScheduleRow(rows *sql.Rows) (*AgentSchedule, error) {
	var (
		name         string
		status       string
		snapshotPath string
		suspendedAt  sql.NullTime
		resumedAt    sql.NullTime
		createdAt    time.Time
	)
	err := rows.Scan(&name, &status, &snapshotPath, &suspendedAt, &resumedAt, &createdAt)
	if err != nil {
		return nil, err
	}
	return buildSchedule(name, status, snapshotPath, suspendedAt, resumedAt, createdAt), nil
}

func buildSchedule(name, status, snapshotPath string, suspendedAt, resumedAt sql.NullTime, createdAt time.Time) *AgentSchedule {
	a := &AgentSchedule{
		Name:         name,
		Status:       AgentStatus(status),
		Priority:     PriorityStandard,
		SnapshotPath: snapshotPath,
		LastActive:   createdAt,
	}
	if suspendedAt.Valid {
		t := suspendedAt.Time
		a.IdleSince = &t
		a.LastActive = t
	}
	if resumedAt.Valid {
		a.LastActive = resumedAt.Time
	}
	return a
}
