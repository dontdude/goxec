package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/dontdude/goxec/internal/domain"
	"github.com/dontdude/goxec/internal/platform/queue"
)

func main() {
	// 1. Initialize Logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// 2. Initialize Redis Queue (Producer Mode)
	// We point to localhost:6379 since we are running outside the container network
	redisQ := queue.NewRedisQueue("localhost:6379", "goxec:jobs", "goxec:workers")

	// 3. Publish Jobs
	for i := 1; i <= 5; i++ {
		job := domain.Job{
			ID:       fmt.Sprintf("job-%d", i),
			Code:     fmt.Sprintf("print('Hello from Redis Job %d')", i),
			Language: "python",
		}

		slog.Info("Publishing job", "jobID", job.ID)
		if err := redisQ.Publish(context.Background(), job); err != nil {
			slog.Error("Failed to publish job", "error", err)
			os.Exit(1)
		}
	}

	slog.Info("Successfully published 5 jobs")
}