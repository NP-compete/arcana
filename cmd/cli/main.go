package main

import (
	"fmt"
	"os"
)

var apiBase string

func init() {
	apiBase = os.Getenv("ARCANA_API_URL")
	if apiBase == "" {
		apiBase = "http://localhost:8080"
	}
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "agent":
		handleAgent(args)
	case "skill":
		handleSkill(args)
	case "model":
		handleModel(args)
	case "eval":
		handleEval(args)
	case "cost":
		handleCost(args)
	case "audit":
		handleAudit(args)
	case "platform":
		handlePlatform(args)
	case "codex":
		handleCodex(args)
	case "compliance":
		handleCompliance(args)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`Arcana CLI — AI Agent Platform

Usage: arcana <command> <subcommand> [flags]

Commands:
  agent       Manage agents (deploy, list, status, suspend, resume)
  skill       Manage skills (list, create, test, merge, prune)
  model       Manage models (list, train, eval, promote, serve)
  eval        Run evaluations (run, report)
  cost        Cost reporting (report, budget)
  audit       Query audit logs
  codex       Search knowledge base
  platform    Platform operations (status, upgrade, backup)
  compliance  Compliance reports (report)

Environment:
  ARCANA_API_URL    API base URL (default: http://localhost:8080)
  ARCANA_API_KEY    API key for authentication`)
}
