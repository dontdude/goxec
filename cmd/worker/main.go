package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/dontdude/goxec/internal/platform/docker"
)

func main() {
	// 1. Initialize Logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	slog.Info("Starting Goxec Worker...")

	// 2. Initialize Docker Client
	// This will panic if Docker is not available (Fail-Fast)
	dockerClient := docker.NewClient()
	slog.Info("Docker wrapper initialized")

	// 3. Manual Verification Test
	// Increased timeout to 60s to allow for cold image pulls
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	slog.Info("Running verification task...")
	code := "print('Hello from Goxec - Verified!')"
	output, err := dockerClient.Run(ctx, code, "python")
	if err != nil {
		slog.Error("Verification failed", "error", err)
		os.Exit(1)
	}

	// 4. Output Result
	slog.Info("Execution finished successfully")
	slog.Info("CONTAINER OUTPUT", "output", output)
}
