package docker

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
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
	// Configures PidsLimit of 64 to prevent fork bombs.
	slog.Info("Creating container", "image", imageName)
	resp, err := c.cli.ContainerCreate(ctx, &container.Config{
		Image: imageName,
		Cmd:   []string{"python", "-c", code},
		// Tty must be false to allow multiplexed stdout/stderr for stdcopy
		Tty: false,
	}, &container.HostConfig{
		Resources: container.Resources{
			Memory:    512 * 1024 * 1024, // 512MB
			PidsLimit: pointInt64(64),    // Fork Bomb protection
		},
	}, nil, nil, "")
	if err != nil {
		slog.Error("Failed to create container", "error", err)
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	containerID := resp.ID
	slog.Info("Container created", "containerID", containerID)

	// CLEANUP: Ensure container is removed even if we crash or timeout.
	// Force: true guarantees removal even if the container is still running (stuck).
	defer func() {
		slog.Info("Removing container", "containerID", containerID)
		if err := c.cli.ContainerRemove(context.Background(), containerID, container.RemoveOptions{Force: true}); err != nil {
			slog.Error("Failed to remove container", "containerID", containerID, "error", err)
		}
	}()

	// 3. Start Container
	if err := c.cli.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return "", fmt.Errorf("failed to start container: %w", err)
	}

	// 4. Wait for Execution (Blocking)
	// We use a select channel to handle both container exit and context cancellation (timeout).
	statusCh, errCh := c.cli.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return "", fmt.Errorf("error waiting for container: %w", err)
		}
	case <-statusCh:
		// Container exited successfully (or passed execution)
	case <-ctx.Done():
		// Context timeout, or cancellation by user
		return "", fmt.Errorf("execution timed out: %w", ctx.Err())
	}

	// 5. Fetch Logs
	// We fetch both Stdout and Stderr.
	out, err := c.cli.ContainerLogs(ctx, containerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	})
	if err != nil {
		return "", fmt.Errorf("failed to get logs: %w", err)
	}
	defer out.Close()

	// 6. Demultiplex Logs (stdcopy)
	// Docker streams combine stdout/stderr headers. stdcopy splits them.
	// We use a limited buffer to prevent OOM (1MB limit).
	const maxLogSize = 1 * 1024 * 1024 // 1MB

	stdoutBuf := &limitedBuffer{buf: new(bytes.Buffer), limit: maxLogSize}
	stderrBuf := &limitedBuffer{buf: new(bytes.Buffer), limit: maxLogSize}

	if _, err := stdcopy.StdCopy(stdoutBuf, stderrBuf, out); err != nil {
		// Ignore limit "errors" as they are just execution limits, not system failures.
		// Real system failures from stdcopy are rare but should be logged.
		if !errors.Is(err, errLogLimitExceeded) {
			return "", fmt.Errorf("failed to demultiplex logs: %w", err)
		}
		slog.Warn("Log limit exceeded", "containerID", containerID)
	}

	return stdoutBuf.String() + stderrBuf.String(), nil
}

// limitedBuffer is a custom writer that enforces a hard size limit.
type limitedBuffer struct {
	buf   *bytes.Buffer
	limit int
}

// errLogLimitExceeded is the sentinel error when logs are truncated.
var errLogLimitExceeded = errors.New("log limit exceeded")

func (l *limitedBuffer) Write(p []byte) (n int, err error) {
	if l.buf.Len()+len(p) > l.limit {
		// Calculate how much we CAN write before hitting the limit
		remaining := l.limit - l.buf.Len()
		if remaining > 0 {
			l.buf.Write(p[:remaining])
			l.buf.WriteString("\n<LOG TRUNCATED>")
		}
		return remaining, errLogLimitExceeded
	}
	return l.buf.Write(p)
}

func (l *limitedBuffer) String() string {
	return l.buf.String()
}

// Helper to get a pointer to an int64 (needed for HostConfig)
func pointInt64(i int64) *int64 {
	return &i
}
