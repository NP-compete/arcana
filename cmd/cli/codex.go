package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func handleCodex(args []string) {
	if len(args) == 0 {
		fmt.Println(`Usage: arcana codex <subcommand>

Subcommands:
  search <query> [--top-k N] [--profile NAME]  Search the knowledge base`)
		return
	}

	switch args[0] {
	case "search":
		fs := flag.NewFlagSet("codex search", flag.ExitOnError)
		topK := fs.Int("top-k", 5, "Number of results to return")
		profile := fs.String("profile", "default", "Search profile to use")
		if err := fs.Parse(args[1:]); err != nil {
			os.Exit(1)
		}
		query := strings.Join(fs.Args(), " ")
		if query == "" {
			fmt.Fprintln(os.Stderr, "Usage: arcana codex search <query> [--top-k N] [--profile NAME]")
			os.Exit(1)
		}
		resp := apiPost("/api/v1/search", map[string]interface{}{
			"query":   query,
			"top_k":   strconv.Itoa(*topK),
			"profile": *profile,
		})
		printJSON(resp)

	default:
		fmt.Fprintf(os.Stderr, "Unknown codex subcommand: %s\n", args[0])
		os.Exit(1)
	}
}
