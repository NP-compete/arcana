package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
)

type TaskStore struct {
	db *sql.DB
}

func NewTaskStore(db *sql.DB) *TaskStore {
	return &TaskStore{db: db}
}

func (s *TaskStore) Create(agent string, input map[string]interface{}, model ModelConfig) *AgentTask {
	now := time.Now().UTC()
	task := &AgentTask{
		ID:        uuid.New().String(),
		Agent:     agent,
		Input:     input,
		Status:    TaskStatusPending,
		Model:     model,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if task.Input == nil {
		task.Input = make(map[string]interface{})
	}

	inputJSON, err := json.Marshal(task.Input)
	if err != nil {
		log.Printf("Create: marshal input: %v", err)
		return task
	}

	modelJSON, err := json.Marshal(task.Model)
	if err != nil {
		log.Printf("Create: marshal model_config: %v", err)
		return task
	}

	_, err = s.db.Exec(`
		INSERT INTO agent_tasks (id, agent, input, status, result, model_config, tokens_used, cost_usd, current_step, error, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NULL, $5, $6, $7, $8, $9, $10, $11)`,
		task.ID, task.Agent, inputJSON, string(task.Status), modelJSON,
		task.TokensUsed, task.Cost, task.CurrentStep, "", task.CreatedAt, task.UpdatedAt,
	)
	if err != nil {
		log.Printf("Create: insert agent_tasks: %v", err)
	}

	return task
}

func (s *TaskStore) Get(id string) (*AgentTask, bool) {
	row := s.db.QueryRow(`
		SELECT id, agent, input, status, result, model_config, tokens_used, cost_usd, current_step, error, created_at, updated_at
		FROM agent_tasks WHERE id = $1`, id)

	task, err := scanTask(row)
	if err == sql.ErrNoRows {
		return nil, false
	}
	if err != nil {
		log.Printf("Get %s: %v", id, err)
		return nil, false
	}
	return task, true
}

func (s *TaskStore) Update(id string, fn func(*AgentTask)) bool {
	task, ok := s.Get(id)
	if !ok {
		return false
	}

	fn(task)
	task.UpdatedAt = time.Now().UTC()

	inputJSON, err := json.Marshal(task.Input)
	if err != nil {
		log.Printf("Update: marshal input %s: %v", id, err)
		return false
	}

	var resultJSON interface{}
	if task.Result != nil {
		b, err := json.Marshal(task.Result)
		if err != nil {
			log.Printf("Update: marshal result %s: %v", id, err)
			return false
		}
		resultJSON = b
	}

	modelJSON, err := json.Marshal(task.Model)
	if err != nil {
		log.Printf("Update: marshal model_config %s: %v", id, err)
		return false
	}

	res, err := s.db.Exec(`
		UPDATE agent_tasks
		SET agent = $1, input = $2, status = $3, result = $4, model_config = $5,
		    tokens_used = $6, cost_usd = $7, current_step = $8, error = $9, updated_at = $10
		WHERE id = $11`,
		task.Agent, inputJSON, string(task.Status), resultJSON, modelJSON,
		task.TokensUsed, task.Cost, task.CurrentStep, task.Error, task.UpdatedAt, id,
	)
	if err != nil {
		log.Printf("Update %s: %v", id, err)
		return false
	}
	n, err := res.RowsAffected()
	if err != nil {
		log.Printf("Update rows affected %s: %v", id, err)
		return false
	}
	return n > 0
}

func (s *TaskStore) List(agent, status, since string, limit, offset int) []AgentTask {
	if limit <= 0 {
		limit = 50
	}

	where, args := buildTaskWhere(agent, status, since)
	query := fmt.Sprintf(`
		SELECT id, agent, input, status, result, model_config, tokens_used, cost_usd, current_step, error, created_at, updated_at
		FROM agent_tasks %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`, where, len(args)+1, len(args)+2)
	args = append(args, limit, offset)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		log.Printf("List: %v", err)
		return []AgentTask{}
	}
	defer rows.Close()

	result := make([]AgentTask, 0)
	for rows.Next() {
		task, err := scanTaskRow(rows)
		if err != nil {
			log.Printf("List scan: %v", err)
			continue
		}
		result = append(result, *task)
	}
	if err := rows.Err(); err != nil {
		log.Printf("List rows: %v", err)
	}
	return result
}

func (s *TaskStore) Count(agent, status, since string) int {
	where, args := buildTaskWhere(agent, status, since)
	query := fmt.Sprintf(`SELECT COUNT(*) FROM agent_tasks %s`, where)

	var count int
	err := s.db.QueryRow(query, args...).Scan(&count)
	if err != nil {
		log.Printf("Count: %v", err)
		return 0
	}
	return count
}

func buildTaskWhere(agent, status, since string) (string, []interface{}) {
	var conditions []string
	var args []interface{}
	idx := 1

	if agent != "" {
		conditions = append(conditions, fmt.Sprintf("agent = $%d", idx))
		args = append(args, agent)
		idx++
	}
	if status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", idx))
		args = append(args, status)
		idx++
	}
	if since != "" {
		if t, err := time.Parse(time.RFC3339, since); err == nil {
			conditions = append(conditions, fmt.Sprintf("created_at >= $%d", idx))
			args = append(args, t)
			idx++
		}
	}

	if len(conditions) == 0 {
		return "", args
	}
	return "WHERE " + strings.Join(conditions, " AND "), args
}

func scanTask(row *sql.Row) (*AgentTask, error) {
	var (
		id          string
		agent       string
		input       []byte
		status      string
		result      []byte
		modelConfig []byte
		tokensUsed  int
		costUSD     float64
		currentStep int
		errText     string
		createdAt   time.Time
		updatedAt   time.Time
	)
	err := row.Scan(&id, &agent, &input, &status, &result, &modelConfig, &tokensUsed, &costUSD, &currentStep, &errText, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	return buildTask(id, agent, input, status, result, modelConfig, tokensUsed, costUSD, currentStep, errText, createdAt, updatedAt)
}

func scanTaskRow(rows *sql.Rows) (*AgentTask, error) {
	var (
		id          string
		agent       string
		input       []byte
		status      string
		result      []byte
		modelConfig []byte
		tokensUsed  int
		costUSD     float64
		currentStep int
		errText     string
		createdAt   time.Time
		updatedAt   time.Time
	)
	err := rows.Scan(&id, &agent, &input, &status, &result, &modelConfig, &tokensUsed, &costUSD, &currentStep, &errText, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	return buildTask(id, agent, input, status, result, modelConfig, tokensUsed, costUSD, currentStep, errText, createdAt, updatedAt)
}

func buildTask(id, agent string, input []byte, status string, result, modelConfig []byte, tokensUsed int, costUSD float64, currentStep int, errText string, createdAt, updatedAt time.Time) (*AgentTask, error) {
	task := &AgentTask{
		ID:          id,
		Agent:       agent,
		Input:       map[string]interface{}{},
		Status:      TaskStatus(status),
		TokensUsed:  tokensUsed,
		Cost:        costUSD,
		CurrentStep: currentStep,
		Error:       errText,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}
	if len(input) > 0 {
		if err := json.Unmarshal(input, &task.Input); err != nil {
			return nil, err
		}
	}
	if len(result) > 0 {
		task.Result = map[string]interface{}{}
		if err := json.Unmarshal(result, &task.Result); err != nil {
			return nil, err
		}
	}
	if len(modelConfig) > 0 {
		if err := json.Unmarshal(modelConfig, &task.Model); err != nil {
			return nil, err
		}
	}
	return task, nil
}
