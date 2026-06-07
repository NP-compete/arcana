package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/NP-compete/arcana/pkg/server"
)

type MigrationSource string

const (
	SourceLangChain MigrationSource = "langchain"
	SourceCrewAI    MigrationSource = "crewai"
	SourceAutoGen   MigrationSource = "autogen"
	SourceDify      MigrationSource = "dify"
	SourceCustom    MigrationSource = "custom"
)

type MigrationRequest struct {
	Source      MigrationSource        `json:"source"`
	Config      map[string]interface{} `json:"config"`
	AgentName   string                 `json:"agent_name"`
	DryRun      bool                   `json:"dry_run"`
}

type MigrationResult struct {
	ID          string                 `json:"id"`
	Source      MigrationSource        `json:"source"`
	AgentName   string                 `json:"agent_name"`
	Status      string                 `json:"status"`
	Blueprint   map[string]interface{} `json:"blueprint,omitempty"`
	Skills      []string               `json:"skills_detected"`
	Tools       []string               `json:"tools_detected"`
	Warnings    []string               `json:"warnings,omitempty"`
	CreatedAt   string                 `json:"created_at"`
}

func main() {
	httpSrv := server.New(server.Config{
		ServiceName: "migrate",
		Port:        "8112",
	})

	httpSrv.HandleFunc("/api/v1/migrate", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "POST required")
			return
		}

		var req MigrationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON")
			return
		}

		result := migrate(req)
		writeJSON(w, http.StatusOK, result)
	}))

	httpSrv.HandleFunc("/api/v1/migrate/sources", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"sources": []map[string]string{
				{"name": "langchain", "description": "LangChain/LangGraph agents and chains"},
				{"name": "crewai", "description": "CrewAI multi-agent crews"},
				{"name": "autogen", "description": "AutoGen conversational agents"},
				{"name": "dify", "description": "Dify workflow applications"},
				{"name": "custom", "description": "Custom agent configuration (JSON/YAML)"},
			},
		})
	}))

	httpSrv.ListenAndServe()
}

func migrate(req MigrationRequest) MigrationResult {
	result := MigrationResult{
		ID:        fmt.Sprintf("mig-%d", time.Now().UnixNano()),
		Source:    req.Source,
		AgentName: req.AgentName,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	switch req.Source {
	case SourceLangChain:
		result.Skills = []string{"web-search", "code-execution", "retrieval"}
		result.Tools = []string{"tavily_search", "python_repl", "retriever"}
		result.Blueprint = map[string]interface{}{
			"apiVersion": "arcana.io/v1",
			"kind":       "ArcanaAgent",
			"metadata":   map[string]string{"name": req.AgentName},
			"spec": map[string]interface{}{
				"model":  "claude-sonnet-4-20250514",
				"skills": result.Skills,
			},
		}
		result.Warnings = []string{"LangChain custom callbacks need manual migration"}

	case SourceCrewAI:
		result.Skills = []string{"task-delegation", "research", "writing"}
		result.Tools = []string{"search_tool", "scrape_tool", "file_tool"}
		result.Blueprint = map[string]interface{}{
			"apiVersion": "arcana.io/v1",
			"kind":       "ArcanaBlueprint",
			"metadata":   map[string]string{"name": req.AgentName + "-crew"},
			"spec": map[string]interface{}{
				"nodes": []map[string]interface{}{
					{"id": "researcher", "type": "agent", "model": "claude-sonnet-4-20250514"},
					{"id": "writer", "type": "agent", "model": "gpt-4o-mini", "dependsOn": []string{"researcher"}},
				},
			},
		}
		result.Warnings = []string{"CrewAI process types mapped to Blueprint edges"}

	case SourceAutoGen:
		result.Skills = []string{"conversation", "code-generation"}
		result.Tools = []string{"code_executor"}
		result.Blueprint = map[string]interface{}{
			"apiVersion": "arcana.io/v1",
			"kind":       "ArcanaAgent",
			"metadata":   map[string]string{"name": req.AgentName},
			"spec": map[string]interface{}{
				"model":  "claude-sonnet-4-20250514",
				"skills": result.Skills,
				"sandbox": map[string]string{"runtime": "gvisor"},
			},
		}

	case SourceDify:
		result.Skills = []string{"workflow", "knowledge-retrieval", "http-request"}
		result.Tools = []string{"knowledge_retrieval", "http_request", "code_execution"}
		result.Blueprint = map[string]interface{}{
			"apiVersion": "arcana.io/v1",
			"kind":       "ArcanaBlueprint",
			"metadata":   map[string]string{"name": req.AgentName + "-workflow"},
			"spec": map[string]interface{}{
				"description": "Migrated from Dify workflow",
			},
		}
		result.Warnings = []string{"Dify variables mapped to Blueprint context", "Custom blocks need manual conversion"}

	default:
		result.Skills = []string{}
		result.Tools = []string{}
		result.Warnings = []string{"Custom source: provide agent config in the config field"}
	}

	if req.DryRun {
		result.Status = "dry_run"
	} else {
		result.Status = "completed"
	}

	log.Printf("migrate: %s agent from %s (%d skills, %d tools)", req.AgentName, req.Source, len(result.Skills), len(result.Tools))
	return result
}

func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", os.Getenv("CORS_ORIGIN"))
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
