package temporal

import (
	"context"
	"fmt"
	"log"
	"os"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

// StartWorker connects to a Temporal server at hostPort, registers all Arcana
// workflows and activities, and starts the worker. It blocks until ctx is
// cancelled or a fatal error occurs.
//
// Returns an error if the Temporal connection or worker fails.
func StartWorker(ctx context.Context, hostPort string) error {
	if hostPort == "" {
		hostPort = os.Getenv("TEMPORAL_ADDRESS")
	}
	if hostPort == "" {
		hostPort = "temporal.arcana.svc.cluster.local:7233"
	}

	c, err := client.Dial(client.Options{HostPort: hostPort})
	if err != nil {
		return fmt.Errorf("dial temporal at %s: %w", hostPort, err)
	}

	w := worker.New(c, TaskQueue, worker.Options{
		MaxConcurrentActivityExecutionSize:     10,
		MaxConcurrentWorkflowTaskExecutionSize: 5,
	})

	// Register workflows.
	w.RegisterWorkflow(RunAgentWorkflow)
	w.RegisterWorkflow(EvaluateSkillWorkflow)
	w.RegisterWorkflow(PromoteAgentWorkflow)

	// Register activities (struct methods).
	acts := NewActivities()
	w.RegisterActivity(acts)

	log.Printf("temporal worker starting on task queue %q (server: %s)", TaskQueue, hostPort)

	// Run blocks until the worker stops. We tie it to the parent context so
	// that the caller can trigger graceful shutdown via cancellation.
	errCh := make(chan error, 1)
	go func() {
		errCh <- w.Run(worker.InterruptCh())
	}()

	select {
	case <-ctx.Done():
		w.Stop()
		return ctx.Err()
	case err := <-errCh:
		c.Close()
		if err != nil {
			return fmt.Errorf("temporal worker stopped: %w", err)
		}
		return nil
	}
}
