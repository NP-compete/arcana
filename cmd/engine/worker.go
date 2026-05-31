package main

import (
	"log"
	"os"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

const taskQueue = "arcana-engine"

// temporalClient is the package-level Temporal client, set during startup.
// nil means Temporal is unavailable and the engine falls back to in-memory mode.
var temporalClient client.Client

// startWorker initialises a Temporal client and starts a background worker.
// If Temporal is unreachable the function logs a warning and returns without
// error so that the service can still serve requests via the in-memory ReAct
// engine.
func startWorker(store *TaskStore, react *ReActEngine) {
	addr := os.Getenv("TEMPORAL_ADDRESS")
	if addr == "" {
		addr = "temporal.arcana.svc.cluster.local:7233"
	}

	c, err := client.Dial(client.Options{HostPort: addr})
	if err != nil {
		log.Printf("WARNING: Temporal unavailable, using in-memory mode: %v", err)
		return
	}

	temporalClient = c

	w := worker.New(c, taskQueue, worker.Options{})
	w.RegisterWorkflow(AgentTaskWorkflow)
	w.RegisterActivity(&Activities{store: store, react: react})

	go func() {
		if err := w.Run(worker.InterruptCh()); err != nil {
			log.Printf("Temporal worker stopped: %v", err)
		}
	}()
}
