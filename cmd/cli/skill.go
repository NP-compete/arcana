package main

import (
	"fmt"
	"os"
)

func handleSkill(args []string) {
	if len(args) == 0 {
		fmt.Println(`Usage: arcana skill <subcommand>

Subcommands:
  list                        List all skills
  create <name> [description] Create a new skill
  test <name>                 Run tests for a skill
  merge <source> <target>     Merge one skill into another
  prune                       Remove unused skills
  transfer <skill> <agent>    Transfer a skill to an agent`)
		return
	}

	switch args[0] {
	case "list":
		resp := apiGet("/api/v1/skills")
		printJSON(resp)

	case "create":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: arcana skill create <name> [description]")
			os.Exit(1)
		}
		payload := map[string]interface{}{
			"name":    args[1],
			"type":    "skills",
			"version": "1.0.0",
		}
		if len(args) >= 3 {
			payload["description"] = args[2]
		}
		resp := apiPost("/api/v1/catalog/skills", payload)
		printJSON(resp)

	case "test":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: arcana skill test <name>")
			os.Exit(1)
		}
		resp := apiPost("/api/v1/eval/run", map[string]interface{}{
			"skill_ref": args[1],
			"judges":    []string{"deterministic", "llm"},
		})
		printJSON(resp)

	case "merge":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: arcana skill merge <source> <target>")
			os.Exit(1)
		}
		resp := apiPost("/api/v1/skills/merge", map[string]interface{}{
			"source": args[1],
			"target": args[2],
		})
		printJSON(resp)

	case "prune":
		resp := apiPost("/api/v1/skills/prune", nil)
		printJSON(resp)

	case "transfer":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: arcana skill transfer <skill> <agent>")
			os.Exit(1)
		}
		resp := apiPost("/api/v1/skills/transfer", map[string]interface{}{
			"skill": args[1],
			"agent": args[2],
		})
		printJSON(resp)

	default:
		fmt.Fprintf(os.Stderr, "Unknown skill subcommand: %s\n", args[0])
		os.Exit(1)
	}
}
