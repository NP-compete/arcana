package main

import (
	"fmt"
	"os"
)

func handlePlatform(args []string) {
	if len(args) == 0 {
		fmt.Println(`Usage: arcana platform <subcommand>

Subcommands:
  status    Show platform health and service status
  backup    Create a platform backup
  version   Show platform version information`)
		return
	}

	switch args[0] {
	case "status":
		resp := apiGet("/api/v1/health")
		printJSON(resp)

	case "backup":
		if len(args) >= 2 && args[1] == "create" {
			resp := apiPost("/api/v1/scheduler/backup", map[string]interface{}{
				"type": "full",
			})
			printJSON(resp)
		} else {
			resp := apiPost("/api/v1/scheduler/backup", map[string]interface{}{
				"type": "full",
			})
			printJSON(resp)
		}

	case "version":
		resp := apiGet("/api/v1/version")
		printJSON(resp)

	default:
		fmt.Fprintf(os.Stderr, "Unknown platform subcommand: %s\n", args[0])
		os.Exit(1)
	}
}
