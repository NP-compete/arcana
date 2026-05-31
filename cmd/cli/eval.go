package main

import (
	"flag"
	"fmt"
	"os"
)

func handleEval(args []string) {
	if len(args) == 0 {
		fmt.Println(`Usage: arcana eval <subcommand>

Subcommands:
  run <skill>         Run an evaluation for a skill
  report [--run-id]   Get evaluation report`)
		return
	}

	switch args[0] {
	case "run":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: arcana eval run <skill>")
			os.Exit(1)
		}
		resp := apiPost("/api/v1/eval/run", map[string]interface{}{
			"skill_ref": args[1],
			"judges":    []string{"deterministic", "llm"},
		})
		printJSON(resp)

	case "report":
		fs := flag.NewFlagSet("eval report", flag.ExitOnError)
		runID := fs.String("run-id", "", "Evaluation run ID")
		if err := fs.Parse(args[1:]); err != nil {
			os.Exit(1)
		}
		path := "/api/v1/eval/report"
		if *runID != "" {
			path += "?run_id=" + *runID
		}
		resp := apiGet(path)
		printJSON(resp)

	default:
		fmt.Fprintf(os.Stderr, "Unknown eval subcommand: %s\n", args[0])
		os.Exit(1)
	}
}
