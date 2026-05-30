package main

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type SandboxStore struct {
	mu      sync.RWMutex
	execs   map[string]*execRecord
}

func NewSandboxStore() *SandboxStore {
	return &SandboxStore{
		execs: make(map[string]*execRecord),
	}
}

func (s *SandboxStore) Execute(req ExecRequest) *ExecResult {
	if req.TimeoutMs <= 0 {
		req.TimeoutMs = 5000
	}
	if req.TimeoutMs > 30000 {
		req.TimeoutMs = 30000
	}

	id := uuid.New().String()
	logs := []string{
		fmt.Sprintf("[%s] sandbox execution started", time.Now().UTC().Format(time.RFC3339)),
		fmt.Sprintf("[%s] language=%s timeout_ms=%d", time.Now().UTC().Format(time.RFC3339), req.Language, req.TimeoutMs),
	}

	record := &execRecord{
		result: ExecResult{
			ID:       id,
			Status:   ExecRunning,
			Language: req.Language,
		},
		logs:      logs,
		createdAt: time.Now().UTC(),
	}

	s.mu.Lock()
	s.execs[id] = record
	s.mu.Unlock()

	start := time.Now()
	result, execLogs := s.runInProcess(req)
	duration := time.Since(start).Milliseconds()

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	result.ID = id
	result.Status = ExecCompleted
	result.DurationMs = duration
	result.MemoryUsedBytes = int64(memStats.Alloc)

	if result.ExitCode != 0 {
		result.Status = ExecFailed
	}

	s.mu.Lock()
	record.result = result
	record.logs = append(record.logs, execLogs...)
	record.logs = append(record.logs, fmt.Sprintf("[%s] execution finished status=%s exit_code=%d",
		time.Now().UTC().Format(time.RFC3339), result.Status, result.ExitCode))
	s.mu.Unlock()

	return &result
}

func (s *SandboxStore) runInProcess(req ExecRequest) (ExecResult, []string) {
	logs := []string{}
	lang := strings.ToLower(req.Language)

	switch lang {
	case "python", "python3":
		return s.runPython(req, &logs)
	case "javascript", "js", "node":
		return s.runJavaScript(req, &logs)
	case "bash", "sh", "shell":
		return s.runShell(req, &logs)
	case "go":
		return s.runSimulated(req, &logs, "go")
	default:
		logs = append(logs, fmt.Sprintf("[%s] unsupported language: %s", time.Now().UTC().Format(time.RFC3339), req.Language))
		return ExecResult{
			Stdout:   "",
			Stderr:   fmt.Sprintf("unsupported language: %s", req.Language),
			ExitCode: 1,
		}, logs
	}
}

func (s *SandboxStore) runPython(req ExecRequest, logs *[]string) (ExecResult, []string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(req.TimeoutMs)*time.Millisecond)
	defer cancel()

	cmd := exec.CommandContext(ctx, "python3", "-c", req.Code)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if req.Inputs != nil {
		for k, v := range req.Inputs {
			cmd.Env = append(cmd.Env, fmt.Sprintf("INPUT_%s=%v", strings.ToUpper(k), v))
		}
	}

	err := cmd.Run()
	result := ExecResult{Stdout: stdout.String(), Stderr: stderr.String(), ExitCode: 0}
	if err != nil {
		result.ExitCode = 1
		if ctx.Err() == context.DeadlineExceeded {
			result.Status = ExecTimeout
			result.Stderr = "execution timed out"
		} else if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		}
	}
	*logs = append(*logs, fmt.Sprintf("[%s] python execution completed", time.Now().UTC().Format(time.RFC3339)))
	return result, *logs
}

func (s *SandboxStore) runJavaScript(req ExecRequest, logs *[]string) (ExecResult, []string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(req.TimeoutMs)*time.Millisecond)
	defer cancel()

	cmd := exec.CommandContext(ctx, "node", "-e", req.Code)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	result := ExecResult{Stdout: stdout.String(), Stderr: stderr.String(), ExitCode: 0}
	if err != nil {
		result.ExitCode = 1
		if ctx.Err() == context.DeadlineExceeded {
			result.Status = ExecTimeout
			result.Stderr = "execution timed out"
		}
	}
	*logs = append(*logs, fmt.Sprintf("[%s] javascript execution completed", time.Now().UTC().Format(time.RFC3339)))
	return result, *logs
}

func (s *SandboxStore) runShell(req ExecRequest, logs *[]string) (ExecResult, []string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(req.TimeoutMs)*time.Millisecond)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", req.Code)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	result := ExecResult{Stdout: stdout.String(), Stderr: stderr.String(), ExitCode: 0}
	if err != nil {
		result.ExitCode = 1
		if ctx.Err() == context.DeadlineExceeded {
			result.Status = ExecTimeout
			result.Stderr = "execution timed out"
		}
	}
	*logs = append(*logs, fmt.Sprintf("[%s] shell execution completed", time.Now().UTC().Format(time.RFC3339)))
	return result, *logs
}

func (s *SandboxStore) runSimulated(req ExecRequest, logs *[]string, lang string) (ExecResult, []string) {
	output := fmt.Sprintf("Simulated %s execution:\n%s\n", lang, req.Code)
	if req.Inputs != nil {
		output += fmt.Sprintf("Inputs: %v\n", req.Inputs)
	}
	*logs = append(*logs, fmt.Sprintf("[%s] simulated %s execution", time.Now().UTC().Format(time.RFC3339), lang))
	return ExecResult{Stdout: output, ExitCode: 0}, *logs
}

func (s *SandboxStore) GetResult(id string) (*ExecResult, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rec, ok := s.execs[id]
	if !ok {
		return nil, false
	}
	copy := rec.result
	return &copy, true
}

func (s *SandboxStore) GetLogs(id string) ([]string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rec, ok := s.execs[id]
	if !ok {
		return nil, false
	}
	logsCopy := make([]string, len(rec.logs))
	copy(logsCopy, rec.logs)
	return logsCopy, true
}
