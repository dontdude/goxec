package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/dontdude/goxec/internal/domain"
	"github.com/dontdude/goxec/internal/platform/docker"
	"github.com/dontdude/goxec/internal/platform/queue"
	"github.com/dontdude/goxec/internal/worker"
)

func main() {
	// 1. Initialize Logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	slog.Info("Starting Goxec Worker Node...")

	// 2. Initialize Config/Adapters
	redisAddr := "localhost:6379"
	dockerClient := docker.NewClient()
	redisQ := queue.NewRedisQueue(redisAddr, "goxec:jobs", "goxec:workers")

	// 3. Initialize Worker Pool (Concurrency: 3)
	concurrency := 3
	pool := worker.NewPool(concurrency, dockerClient)
	pool.Start()
	defer pool.Stop() // Ensure cleanup on exit

	// 4. Subscribe to Jobs
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	jobsCh, err := redisQ.Subscribe(ctx)
	if err != nil {
		slog.Error("Failed to subscribe to queue", "error", err)
		os.Exit(1)
	}

	slog.Info("Worker Node listening for jobs...")

	// 5. Handle Shutdown Signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	// 6. Goroutine to Channel Jobs from Redis -> Worker Pool
	go func() {
		for job := range jobsCh {
			slog.Info("Received job from Redis", "jobID", job.ID)
			
			// We need a Result Channel to handle the output
			// For now, we just print it to stdout to verify End-to-End
			resCh := make(chan domain.JobResult)
			job.ResultCh = resCh
			
			pool.Submit(job)

			// Spawn a goroutine to log the result and acknowledge the job.
			// The original job struct is passed to access the RawID.
			go func(j domain.Job, ch <-chan domain.JobResult) {
				result := <-ch
				
				// 1. Log job execution result
				if result.Error != nil {
					// Ack failure scenarios to prevent infinite redelivery (until DLQ is implemented).
					slog.Error("Job Execution Failed", "jobID", j.ID, "error", result.Error)
				} else {
					slog.Info("Job Successfully Executed", "jobID", j.ID, "output", result.Output)
				}

				// 2. Broadcast to Pub/Sub
				// Send the output back to customers.
				output := result.Output
				if result.Error != nil {
					output = fmt.Sprintf("Error: %v", result.Error)
				}

				if err := redisQ.Broadcast(context.Background(), j.ID, output); err != nil {
					slog.Error("Failed to broadcast log", "jobID", j.ID, "error", err)
				}

				// 3. Acknowledge processing completion to Redis.
				slog.Info("Acknowledging job", "jobID", j.ID, "streamID", j.RawID)
				if err := redisQ.Acknowledge(context.Background(), j.RawID); err != nil {
					slog.Error("Failed to ACK job", "jobID", j.ID, "error", err)
				}
			}(job, resCh)
		}
	}()

	// Wait for termination
	<-sigCh
	slog.Info("Shutdown signal received")
	cancel() // Stop Redis subscriber
	// atomic shutdown of pool is handled by defer
}