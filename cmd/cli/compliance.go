package main

import (
	"flag"
	"fmt"
	"os"
)

func handleCompliance(args []string) {
	if len(args) == 0 {
		fmt.Println(`Usage: arcana compliance <subcommand>

Subcommands:
  report [--framework soc2|gdpr|hipaa]  Generate a compliance report`)
		return
	}

	switch args[0] {
	case "report":
		fs := flag.NewFlagSet("compliance report", flag.ExitOnError)
		framework := fs.String("framework", "soc2", "Compliance framework (soc2, gdpr, hipaa)")
		if err := fs.Parse(args[1:]); err != nil {
			os.Exit(1)
		}
		resp := apiGet("/api/v1/audit/compliance?framework=" + *framework)
		printJSON(resp)

	default:
		fmt.Fprintf(os.Stderr, "Unknown compliance subcommand: %s\n", args[0])
		os.Exit(1)
	}
}
