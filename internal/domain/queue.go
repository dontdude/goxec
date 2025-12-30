package domain

import "context"

// JobQueue defines the contract for a distributed job queue.
// It decouples the application form the underlying message broker (Redis, RabbitMQ, etc.).
type JobQueue interface {
	// Publish enqueues a job for processing.
	Publish(ctx context.Context, job Job) error

	// Subscribe returns a read-only channel that streams jobs from the queue.
	// It handles the details of consumer groups and acknowledgments internally.
	Subscribe(ctx context.Context) (<-chan Job, error)
}