package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"time"
)

type BlueprintStore struct {
	db *sql.DB
}

func NewBlueprintStore(db *sql.DB) *BlueprintStore {
	return &BlueprintStore{db: db}
}

func (s *BlueprintStore) Save(bp *Blueprint) {
	now := time.Now().UTC()

	nodesJSON, err := json.Marshal(bp.Nodes)
	if err != nil {
		log.Printf("Save: marshal nodes: %v", err)
		return
	}
	edgesJSON, err := json.Marshal(bp.Edges)
	if err != nil {
		log.Printf("Save: marshal edges: %v", err)
		return
	}

	_, err = s.db.Exec(`
		INSERT INTO blueprints (name, description, nodes, edges, fallback, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (name) DO UPDATE SET
			description = EXCLUDED.description,
			nodes = EXCLUDED.nodes,
			edges = EXCLUDED.edges,
			fallback = EXCLUDED.fallback,
			updated_at = EXCLUDED.updated_at`,
		bp.Name, bp.Description, nodesJSON, edgesJSON, bp.Fallback, now, now,
	)
	if err != nil {
		log.Printf("Save: upsert blueprint %s: %v", bp.Name, err)
		return
	}

	bp.UpdatedAt = now
	if bp.CreatedAt.IsZero() {
		bp.CreatedAt = now
	}
}

func (s *BlueprintStore) Get(name string) (*Blueprint, bool) {
	row := s.db.QueryRow(`
		SELECT name, description, nodes, edges, fallback, created_at, updated_at
		FROM blueprints WHERE name = $1`, name)

	bp, err := scanBlueprint(row)
	if err == sql.ErrNoRows {
		return nil, false
	}
	if err != nil {
		log.Printf("Get %s: %v", name, err)
		return nil, false
	}
	return bp, true
}

func (s *BlueprintStore) Delete(name string) bool {
	res, err := s.db.Exec(`DELETE FROM blueprints WHERE name = $1`, name)
	if err != nil {
		log.Printf("Delete %s: %v", name, err)
		return false
	}
	n, err := res.RowsAffected()
	if err != nil {
		log.Printf("Delete %s rows affected: %v", name, err)
		return false
	}
	return n > 0
}

func (s *BlueprintStore) List() []Blueprint {
	rows, err := s.db.Query(`
		SELECT name, description, nodes, edges, fallback, created_at, updated_at
		FROM blueprints ORDER BY created_at`)
	if err != nil {
		log.Printf("List: %v", err)
		return []Blueprint{}
	}
	defer rows.Close()

	result := make([]Blueprint, 0)
	for rows.Next() {
		bp, err := scanBlueprintRow(rows)
		if err != nil {
			log.Printf("List scan: %v", err)
			continue
		}
		result = append(result, *bp)
	}
	if err := rows.Err(); err != nil {
		log.Printf("List rows: %v", err)
	}
	return result
}

func scanBlueprint(row *sql.Row) (*Blueprint, error) {
	var (
		name        string
		description string
		nodes       []byte
		edges       []byte
		fallback    string
		createdAt   time.Time
		updatedAt   time.Time
	)
	err := row.Scan(&name, &description, &nodes, &edges, &fallback, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	return buildBlueprint(name, description, nodes, edges, fallback, createdAt, updatedAt)
}

func scanBlueprintRow(rows *sql.Rows) (*Blueprint, error) {
	var (
		name        string
		description string
		nodes       []byte
		edges       []byte
		fallback    string
		createdAt   time.Time
		updatedAt   time.Time
	)
	err := rows.Scan(&name, &description, &nodes, &edges, &fallback, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	return buildBlueprint(name, description, nodes, edges, fallback, createdAt, updatedAt)
}

func buildBlueprint(name, description string, nodes, edges []byte, fallback string, createdAt, updatedAt time.Time) (*Blueprint, error) {
	bp := &Blueprint{
		Name:        name,
		Description: description,
		Fallback:    fallback,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}
	if len(nodes) > 0 {
		if err := json.Unmarshal(nodes, &bp.Nodes); err != nil {
			return nil, err
		}
	}
	if bp.Nodes == nil {
		bp.Nodes = []Node{}
	}
	if len(edges) > 0 {
		if err := json.Unmarshal(edges, &bp.Edges); err != nil {
			return nil, err
		}
	}
	if bp.Edges == nil {
		bp.Edges = []Edge{}
	}
	return bp, nil
}
