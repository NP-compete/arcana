package main

import (
	"flag"
	"fmt"
	"os"
)

func handleAudit(args []string) {
	if len(args) == 0 {
		fmt.Println(`Usage: arcana audit <subcommand>

Subcommands:
  query [--agent NAME] [--since DATE] [--action ACTION]  Query audit logs`)
		return
	}

	switch args[0] {
	case "query":
		fs := flag.NewFlagSet("audit query", flag.ExitOnError)
		agent := fs.String("agent", "", "Filter by agent name")
		since := fs.String("since", "", "Start date (YYYY-MM-DD)")
		action := fs.String("action", "", "Filter by action type")
		limit := fs.Int("limit", 50, "Maximum number of results")
		if err := fs.Parse(args[1:]); err != nil {
			os.Exit(1)
		}
		path := "/api/v1/audit"
		sep := "?"
		if *agent != "" {
			path += sep + "agent=" + *agent
			sep = "&"
		}
		if *since != "" {
			path += sep + "since=" + *since
			sep = "&"
		}
		if *action != "" {
			path += sep + "action=" + *action
			sep = "&"
		}
		path += fmt.Sprintf("%slimit=%d", sep, *limit)
		resp := apiGet(path)
		printJSON(resp)

	default:
		fmt.Fprintf(os.Stderr, "Unknown audit subcommand: %s\n", args[0])
		os.Exit(1)
	}
}
