package main

import (
	"fmt"
	"os"
)

func handleAgent(args []string) {
	if len(args) == 0 {
		fmt.Println(`Usage: arcana agent <subcommand>

Subcommands:
  deploy <blueprint.yaml>   Deploy an agent from a blueprint
  list                      List all agents
  status <name>             Get detailed agent status
  suspend <name>            Suspend a running agent
  resume <name>             Resume a suspended agent
  delete <name>             Delete an agent`)
		return
	}

	switch args[0] {
	case "deploy":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: arcana agent deploy <blueprint.yaml>")
			os.Exit(1)
		}
		data, err := os.ReadFile(args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file %s: %v\n", args[1], err)
			os.Exit(1)
		}
		resp := apiPost("/api/v1/blueprints", map[string]string{"yaml": string(data)})
		printJSON(resp)

	case "list":
		resp := apiGet("/api/v1/agents")
		printJSON(resp)

	case "status":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: arcana agent status <name>")
			os.Exit(1)
		}
		resp := apiGet("/api/v1/agents/" + args[1] + "/detail")
		printJSON(resp)

	case "suspend":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: arcana agent suspend <name>")
			os.Exit(1)
		}
		resp := apiPost("/api/v1/agents/suspend/"+args[1], nil)
		printJSON(resp)

	case "resume":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: arcana agent resume <name>")
			os.Exit(1)
		}
		resp := apiPost("/api/v1/agents/resume/"+args[1], nil)
		printJSON(resp)

	case "delete":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: arcana agent delete <name>")
			os.Exit(1)
		}
		resp := apiDelete("/api/v1/agents/" + args[1])
		printJSON(resp)

	default:
		fmt.Fprintf(os.Stderr, "Unknown agent subcommand: %s\n", args[0])
		os.Exit(1)
	}
}
