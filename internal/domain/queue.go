package domain

import "context"

// JobQueue defines the contract for a distributed job queue.
// It decouples the application from the underlying message broker (Redis, RabbitMQ, etc.).
type JobQueue interface {
	// Publish enqueues a job for processing.
	Publish(ctx context.Context, job Job) error

	// Subscribe returns a read-only channel that streams jobs from the queue.
	// It handles the details of consumer groups and acknowledgments internally.
	Subscribe(ctx context.Context) (<-chan Job, error)

	// Acknowledge confirms that a job has been successfully processed. 
	// This removes it from the Pending Entry list (PEL).
	Acknowledge(ctx context.Context, jobID string) error

	// Broadcast publishes the job execution result to the Pub/Sub channel.
	Broadcast(ctx context.Context, jobID string, output string) error

	// SubscribeLogs returns a channel that streams execution results from all workers.
	SubscribeLogs(ctx context.Context) (<-chan JobResult, error)
}