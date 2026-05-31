package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/NP-compete/arcana/pkg/db"
)

// ---------------------------------------------------------------------------
// Helper — create a Server with a DB-backed TaskStore and a ReActEngine.
// Tests that require a live PostgreSQL instance are skipped when the DB
// is unavailable.
// ---------------------------------------------------------------------------

func testDB(t *testing.T) *TaskStore {
	t.Helper()
	conn, err := db.Connect()
	if err != nil {
		t.Skipf("skipping: database not available: %v", err)
	}
	return NewTaskStore(conn)
}

func newTestServer(t *testing.T) (*Server, *TaskStore) {
	t.Helper()
	store := testDB(t)
	react := NewReActEngine(store)
	return NewServer(store, react), store
}

// ---------------------------------------------------------------------------
// Helper — register test routes on a mux matching main.go's routing.
// ---------------------------------------------------------------------------

func newTestMux(srv *Server) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/tasks", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			srv.handleSubmitTask(w, r)
		case http.MethodGet:
			srv.handleListTasks(w, r)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	})
	mux.HandleFunc("/api/v1/tasks/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/cancel") {
			srv.handleCancelTask(w, r)
			return
		}
		srv.handleGetTask(w, r)
	})
	return mux
}

// ---------------------------------------------------------------------------
// A3. Tests
// ---------------------------------------------------------------------------

func TestSubmitTask(t *testing.T) {
	srv, _ := newTestServer(t)
	mux := newTestMux(srv)

	body := `{"agent":"test-agent","input":{"text":"hello"},"model":{"model":"gpt-4o"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var task AgentTask
	json.NewDecoder(rec.Body).Decode(&task)
	if task.Agent != "test-agent" {
		t.Errorf("agent: got %q, want %q", task.Agent, "test-agent")
	}
	if task.ID == "" {
		t.Error("expected non-empty task ID")
	}
	// The handler fires react.Run in a goroutine, so by the time we decode
	// the response the status may have advanced from "pending" to "running".
	if task.Status != TaskStatusPending && task.Status != TaskStatusRunning {
		t.Errorf("status: got %q, want pending or running", task.Status)
	}
	if task.CreatedAt.IsZero() {
		t.Error("expected non-zero created_at")
	}
}

func TestSubmitTask_MissingAgent(t *testing.T) {
	srv, _ := newTestServer(t)
	mux := newTestMux(srv)

	body := `{"input":{"text":"hello"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	var resp ErrorResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Error != "agent is required" {
		t.Errorf("error: got %q, want %q", resp.Error, "agent is required")
	}
}

func TestSubmitTask_InvalidJSON(t *testing.T) {
	srv, _ := newTestServer(t)
	mux := newTestMux(srv)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", strings.NewReader("{bad json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	var resp ErrorResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Error != "invalid JSON body" {
		t.Errorf("error: got %q, want %q", resp.Error, "invalid JSON body")
	}
}

func TestListTasks(t *testing.T) {
	srv, store := newTestServer(t)
	mux := newTestMux(srv)

	// Seed some tasks.
	store.Create("agent-a", map[string]interface{}{"text": "task 1"}, ModelConfig{Model: "gpt-4o"})
	store.Create("agent-b", map[string]interface{}{"text": "task 2"}, ModelConfig{Model: "gpt-4o"})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var resp TaskListResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	// There may be existing tasks in the DB, so just verify we got at least 2.
	if resp.Total < 2 {
		t.Errorf("total: got %d, want at least 2", resp.Total)
	}
}

func TestGetTask(t *testing.T) {
	srv, store := newTestServer(t)
	mux := newTestMux(srv)

	task := store.Create("my-agent", map[string]interface{}{"q": "test"}, ModelConfig{Model: "gpt-4o"})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/"+task.ID, nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var got AgentTask
	json.NewDecoder(rec.Body).Decode(&got)
	if got.ID != task.ID {
		t.Errorf("id: got %q, want %q", got.ID, task.ID)
	}
	if got.Agent != "my-agent" {
		t.Errorf("agent: got %q, want %q", got.Agent, "my-agent")
	}
}

func TestGetTask_NotFound(t *testing.T) {
	srv, _ := newTestServer(t)
	mux := newTestMux(srv)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/nonexistent-id", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}

	var resp ErrorResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Error != "task not found" {
		t.Errorf("error: got %q, want %q", resp.Error, "task not found")
	}
}

func TestCancelTask(t *testing.T) {
	srv, store := newTestServer(t)
	mux := newTestMux(srv)

	task := store.Create("cancel-agent", nil, ModelConfig{})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/"+task.ID+"/cancel", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var updated AgentTask
	json.NewDecoder(rec.Body).Decode(&updated)
	if updated.Status != TaskStatusCancelled {
		t.Errorf("status: got %q, want %q", updated.Status, TaskStatusCancelled)
	}
}

func TestCancelTask_NotFound(t *testing.T) {
	srv, _ := newTestServer(t)
	mux := newTestMux(srv)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/nonexistent/cancel", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestCancelTask_AlreadyCompleted(t *testing.T) {
	srv, store := newTestServer(t)
	mux := newTestMux(srv)

	task := store.Create("done-agent", nil, ModelConfig{})
	store.Update(task.ID, func(t *AgentTask) {
		t.Status = TaskStatusCompleted
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/"+task.ID+"/cancel", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestCancelTask_AlreadyCancelled(t *testing.T) {
	srv, store := newTestServer(t)
	mux := newTestMux(srv)

	task := store.Create("re-cancel-agent", nil, ModelConfig{})
	store.Update(task.ID, func(t *AgentTask) {
		t.Status = TaskStatusCancelled
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/"+task.ID+"/cancel", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	// Cancelling an already-cancelled task returns 200 (idempotent).
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestSubmitTask_MethodNotAllowed(t *testing.T) {
	srv, _ := newTestServer(t)
	mux := newTestMux(srv)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/tasks", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// TaskStore unit tests
// ---------------------------------------------------------------------------

func TestTaskStore_Create(t *testing.T) {
	store := testDB(t)
	task := store.Create("agent-x", map[string]interface{}{"q": "test"}, ModelConfig{Model: "gpt-4o"})

	if task.ID == "" {
		t.Error("expected non-empty ID")
	}
	if task.Agent != "agent-x" {
		t.Errorf("agent: got %q, want %q", task.Agent, "agent-x")
	}
	if task.Status != TaskStatusPending {
		t.Errorf("status: got %q, want %q", task.Status, TaskStatusPending)
	}
}

func TestTaskStore_Get(t *testing.T) {
	store := testDB(t)
	created := store.Create("agent-y", nil, ModelConfig{})

	got, ok := store.Get(created.ID)
	if !ok {
		t.Fatal("expected to find task")
	}
	if got.ID != created.ID {
		t.Errorf("id mismatch: got %q, want %q", got.ID, created.ID)
	}
}

func TestTaskStore_Get_NotFound(t *testing.T) {
	store := testDB(t)
	_, ok := store.Get("nonexistent")
	if ok {
		t.Error("expected task not to be found")
	}
}

func TestTaskStore_Update(t *testing.T) {
	store := testDB(t)
	task := store.Create("agent-z", nil, ModelConfig{})

	ok := store.Update(task.ID, func(t *AgentTask) {
		t.Status = TaskStatusRunning
	})
	if !ok {
		t.Fatal("expected update to succeed")
	}

	got, _ := store.Get(task.ID)
	if got.Status != TaskStatusRunning {
		t.Errorf("status: got %q, want %q", got.Status, TaskStatusRunning)
	}
}

// ---------------------------------------------------------------------------
// ReActEngine basic test
// ---------------------------------------------------------------------------

func TestReActEngine_Run(t *testing.T) {
	store := testDB(t)
	react := NewReActEngine(store)

	task := store.Create("react-agent", map[string]interface{}{"q": "test"}, ModelConfig{Model: "gpt-4o"})

	// Run synchronously with 2 steps.
	react.Run(task.ID, 2)

	got, ok := store.Get(task.ID)
	if !ok {
		t.Fatal("expected task to exist")
	}
	if got.Status != TaskStatusCompleted {
		t.Errorf("status: got %q, want %q", got.Status, TaskStatusCompleted)
	}
	if got.TokensUsed <= 0 {
		t.Error("expected positive token count")
	}
	if got.Cost <= 0 {
		t.Error("expected positive cost")
	}
	if got.Result == nil {
		t.Error("expected non-nil result")
	}
}

func TestExtractPathParam(t *testing.T) {
	tests := []struct {
		path   string
		prefix string
		want   string
	}{
		{"/api/v1/tasks/abc-123", "/api/v1/tasks/", "abc-123"},
		{"/api/v1/tasks/abc-123/cancel", "/api/v1/tasks/", "abc-123"},
		{"/api/v1/tasks/", "/api/v1/tasks/", ""},
		{"/api/v1/other/xyz", "/api/v1/tasks/", ""},
	}
	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			got := extractPathParam(tc.path, tc.prefix)
			if got != tc.want {
				t.Errorf("extractPathParam(%q, %q) = %q, want %q", tc.path, tc.prefix, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Helper functions
// ---------------------------------------------------------------------------

func TestReActPhase(t *testing.T) {
	expected := []string{"plan", "act", "observe", "evaluate", "plan"}
	for i, want := range expected {
		got := reactPhase(i + 1)
		if got != want {
			t.Errorf("reactPhase(%d) = %q, want %q", i+1, got, want)
		}
	}
}

func TestEvaluateStep(t *testing.T) {
	if got := evaluateStep(5, 5); got != "goal_satisfied" {
		t.Errorf("last step: got %q, want %q", got, "goal_satisfied")
	}
	if got := evaluateStep(4, 10); got != "continue" {
		t.Errorf("step 4 of 10: got %q, want %q", got, "continue")
	}
	if got := evaluateStep(1, 5); got != "proceed" {
		t.Errorf("step 1 of 5: got %q, want %q", got, "proceed")
	}
}
