package main

import (
	"fmt"
	"os"
)

func handleModel(args []string) {
	if len(args) == 0 {
		fmt.Println(`Usage: arcana model <subcommand>

Subcommands:
  list                List all models
  train <experiment>  Start a training experiment
  eval <model>        Evaluate a model
  promote <model>     Promote a model to production
  serve <model>       Start serving a model`)
		return
	}

	switch args[0] {
	case "list":
		resp := apiGet("/api/v1/models")
		printJSON(resp)

	case "train":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: arcana model train <experiment>")
			os.Exit(1)
		}
		resp := apiPost("/api/v1/experiments", map[string]interface{}{
			"name": args[1],
		})
		printJSON(resp)

	case "eval":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: arcana model eval <model>")
			os.Exit(1)
		}
		resp := apiPost("/api/v1/eval/run", map[string]interface{}{
			"model_ref": args[1],
			"judges":    []string{"deterministic", "llm"},
		})
		printJSON(resp)

	case "promote":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: arcana model promote <model>")
			os.Exit(1)
		}
		resp := apiPost("/api/v1/promotions", map[string]interface{}{
			"model":       args[1],
			"target":      "production",
			"auto_approve": false,
		})
		printJSON(resp)

	case "serve":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: arcana model serve <model>")
			os.Exit(1)
		}
		resp := apiPost("/api/v1/models/serve", map[string]interface{}{
			"model": args[1],
		})
		printJSON(resp)

	default:
		fmt.Fprintf(os.Stderr, "Unknown model subcommand: %s\n", args[0])
		os.Exit(1)
	}
}
