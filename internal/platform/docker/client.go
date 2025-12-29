package docker

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/dontdude/goxec/internal/domain"
)

// Client wraps the official Docker SDK client.
type Client struct {
	cli *client.Client
}

// Check if Client implements domain.ContainerRunner
var _ domain.ContainerRunner = (*Client)(nil)

// NewClient initializes and returns a verified Docker client.
// It performs a connection check (Ping) upon initialization.
// If the Docker daemon is unreachable, the function panics to prevent the service from starting in a broken state
// (Fail-Fast).
func NewClient() *Client {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		slog.Error("Failed to create Docker client", "error", err)
		panic(err)
	}

	// Ping Docker to ensure connection
	ctx := context.Background()
	_, err = cli.Ping(ctx)
	if err != nil {
		slog.Error("Failed to connect to Docker Daemon", "error", err)
		panic(err)
	}

	slog.Info("Docker Client initialized successfully")
	return &Client{cli: cli}
}

// Run executes the provided code within an ephemeral Docker container.
// It enforces resource limits (memory) and context cancellation.
func (c *Client) Run(ctx context.Context, code string, language string) (string, error) {
	// 1. Pull Image
	// TODO: Extract image name resolution to a configuration or map.
	imageName := "python:alpine"
	
	slog.Info("Pulling image", "image", imageName)
	reader, err := c.cli.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		slog.Error("Failed to pull image", "image", imageName, "error", err)
		return "", fmt.Errorf("failed to pull image: %w", err)
	}
	// Drain the response body to ensure the pull completes properly.
	defer reader.Close()
	io.Copy(io.Discard, reader)

	// 2. Create Container with Limits
	// Configures a hard memory limit of 512MB via Cgroups to prevent resource exhaustion.
	slog.Info("Creating container", "image", imageName)
	resp, err := c.cli.ContainerCreate(ctx, &container.Config{
		Image: imageName,
		Cmd:   []string{"python", "-c", code},
		// Tty:   false, // We will use streaming later
	}, &container.HostConfig{
		Resources: container.Resources{
			Memory: 512 * 1024 * 1024, // 512MB
		},
	}, nil, nil, "")
	if err != nil {
		slog.Error("Failed to create container", "error", err)
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	slog.Info("Container created successfully", "containerID", resp.ID)

	// 3. Return Stub
	// Implementation of container start, attach, and wait pending.
	return "Docker Client Ready", nil
}
