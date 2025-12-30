package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dontdude/goxec/internal/domain"
	"github.com/dontdude/goxec/internal/platform/docker"
	"github.com/dontdude/goxec/internal/worker"
)

func main() {
	// 1. Initialize Logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	slog.Info("Starting Goxec Worker Node...")

	// 2. Initialize Docker Client
	dockerClient := docker.NewClient()

	// 3. Initialize Worker Pool (Concurrency: 3)
	concurrency := 3
	pool := worker.NewPool(concurrency, dockerClient)
	pool.Start()
	defer pool.Stop() // Ensure cleanup on exit

	// 4. Handle Shutdown Signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	// 5. Submit Test Jobs
	// We'll create a Result Channel to read back results
	resultCh := make(chan domain.JobResult)

	go func() {
		// Verify: Submit 5 jobs (more than concurrency) to see buffering/wait
		for i := 1; i <= 5; i++ {
			jobID := fmt.Sprintf("job-%d", i)
			code := fmt.Sprintf("print('Hello from Job %d')", i)
			
			slog.Info("Submitting job", "jobID", jobID)
			pool.Submit(domain.Job{
				ID:       jobID,
				Code:     code,
				Language: "python",
				ResultCh: resultCh,
			})
			// Slight delay to simulate staggered arrival, or just blast them
			time.Sleep(100 * time.Millisecond)
		}
	}()

	// 6. Loop and Wait
	// In a real app, this would be consuming from Redis.
	// Here we just wait for 5 results or a signal.
	completed := 0
	for {
		select {
		case res := <-resultCh:
			completed++
			if res.Error != nil {
				slog.Error("Job failed", "error", res.Error)
			} else {
				slog.Info("Job completed", "output", res.Output)
			}
			if completed == 5 {
				slog.Info("All verification jobs completed. Exiting.")
				return
			}
		case <-sigCh:
			slog.Info("Shutdown signal received")
			// defer pool.Stop() will run now
			return
		}
	}
}
