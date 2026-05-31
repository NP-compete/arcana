package main

import (
	"flag"
	"fmt"
	"os"
)

func handleCost(args []string) {
	if len(args) == 0 {
		fmt.Println(`Usage: arcana cost <subcommand>

Subcommands:
  report [--agent NAME] [--since DATE]  Get cost report
  budget set <agent> <amount>           Set daily budget for an agent
  budget get <agent>                    Get budget for an agent`)
		return
	}

	switch args[0] {
	case "report":
		fs := flag.NewFlagSet("cost report", flag.ExitOnError)
		agent := fs.String("agent", "", "Filter by agent name")
		since := fs.String("since", "", "Start date (YYYY-MM-DD)")
		if err := fs.Parse(args[1:]); err != nil {
			os.Exit(1)
		}
		path := "/api/v1/costs"
		sep := "?"
		if *agent != "" {
			path += sep + "agent=" + *agent
			sep = "&"
		}
		if *since != "" {
			path += sep + "since=" + *since
		}
		resp := apiGet(path)
		printJSON(resp)

	case "budget":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: arcana cost budget <set|get> ...")
			os.Exit(1)
		}
		switch args[1] {
		case "set":
			if len(args) < 4 {
				fmt.Fprintln(os.Stderr, "Usage: arcana cost budget set <agent> <amount>")
				os.Exit(1)
			}
			resp := apiPost("/api/v1/budget", map[string]interface{}{
				"agent":  args[2],
				"budget": args[3],
			})
			printJSON(resp)
		case "get":
			if len(args) < 3 {
				fmt.Fprintln(os.Stderr, "Usage: arcana cost budget get <agent>")
				os.Exit(1)
			}
			resp := apiGet("/api/v1/budget?agent=" + args[2])
			printJSON(resp)
		default:
			fmt.Fprintf(os.Stderr, "Unknown budget subcommand: %s\n", args[1])
			os.Exit(1)
		}

	default:
		fmt.Fprintf(os.Stderr, "Unknown cost subcommand: %s\n", args[0])
		os.Exit(1)
	}
}
