package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/dontdude/goxec/internal/domain"
	"github.com/redis/go-redis/v9"
)

// RedisQueue implements domain.JobQueue using Redis Streams.
type RedisQueue struct {
	client *redis.Client
	stream string
	group string
}

// Ensure RedisQueue satisfies the interface
var _ domain.JobQueue = (*RedisQueue)(nil)

// NewRedisQueue returns a new Redis-backed queue adapter.
func NewRedisQueue(addr, stream, group string) (*RedisQueue) {
	rdb := redis.NewClient(&redis.Options{
		Addr: addr,
	})

	// Fail-fast ping check
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		panic(fmt.Sprintf("failed to connect to redis: %v", err))
	}

	return &RedisQueue{
		client: rdb,
		stream: stream,
		group: group,
	}
}

// Publish enqueues a job to the Redis stream using XADD (Producer)
func (r *RedisQueue) Publish(ctx context.Context, job domain.Job) error {
	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	// XADD appends to the stream.
	// We use "*" Id to let Redis generate a timestamp-based ID.
	err = r.client.XAdd(ctx, &redis.XAddArgs{
		Stream: r.stream,
		Values: map[string]interface{}{
			"job": data,
		},
	}).Err()

	if err != nil {
		return fmt.Errorf("redis publish failed: %w", err)
	}
	return nil
}

// Subscribe returns a channel of jobs using the XREADGROUP (Consumer).
func (r *RedisQueue) Subscribe(ctx context.Context) (<-chan domain.Job, error) {
	// 1. Ensure the Consumer Group exists
	// MkStream guarantees the stream exists even if empty.
	err := r.client.XGroupCreateMkStream(ctx, r.stream, r.group, "$").Err()
	if err != nil {
		// Ignore "BUSYGROUP Consumer Group name already exists" error
		if err.Error() != "BUSYGROUP Consumer Group name already exists" {
			return nil, fmt.Errorf("failed to create consumer group: %w", err)
		}
	} 

	// 2. Spawn a background listener
	outCh := make(chan domain.Job)

	// Generate a unique consumer name (e.g: hostname-pid)
	consumerId, _ := os.Hostname()
	if consumerId == "" {
		consumerId = fmt.Sprintf("consumer-%d", time.Now().UnixNano())
	}

	go func() {
		defer close(outCh)

		for {
			select {
			case <-ctx.Done():
				return
			default: 
				// XREADGROUP blocks until a message is available (Block: 0 means forever, but we use 2s to check context)
				streams, err := r.client.XReadGroup(ctx, &redis.XReadGroupArgs{
					Group:    r.group,
					Consumer: consumerId,
					Streams:  []string{r.stream, ">"}, // ">" means new messages
					Count:    1,
					Block:    2 * time.Second,
				}).Result()
				if err != nil {
					if err == redis.Nil {
						continue // Timeout, retry
					}
					// Check if context canceled during blocking call
					if ctx.Err() != nil {
						return
					}
					slog.Error("Redis read error", "error", err)
					time.Sleep(1 * time.Second) // Backoff
					continue
				}
				// Process Messages
				for _, stream := range streams {
					for _, msg := range stream.Messages {
						// Extract Job Data
						val, ok := msg.Values["job"].(string)
						if !ok {
							slog.Error("Invalid message format", "msgID", msg.ID)
							continue
						}
						var job domain.Job
						if err := json.Unmarshal([]byte(val), &job); err != nil {
							slog.Error("Failed to unmarshal job", "error", err)
							continue
						}
						
						// Capture the Redis Stream ID so we can ACK later
						job.RawID = msg.ID
						
						outCh <- job
					}
				}
			}
		}
	}()
	return outCh, nil
}

// Acknowledge confirms processing using XACK. 
func (r *RedisQueue) Acknowledge(ctx context.Context, jobID string) error {
	return r.client.XAck(ctx, r.stream, r.group, jobID).Err()
}

// Broadcast publishes the execution result to the "goxec:logs" channel.
func (r *RedisQueue) Broadcast(ctx context.Context, result domain.JobResult) error {
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal log: %w", err)
	}

	return r.client.Publish(ctx, "goxec:logs", data).Err()
}

// SubscribeLogs subscribes to "goxec:logs" and streams results to a Go channel.
func (r *RedisQueue) SubscribeLogs(ctx context.Context) (<-chan domain.JobResult, error) {
	// Create the PubSub connection
	pubsub := r.client.Subscribe(ctx, "goxec:logs")

	// Wait for confirmation that we are subscribed
	if _, err := pubsub.Receive(ctx); err != nil {
		return nil, fmt.Errorf("failed to subscribe to logs: %w", err)
	}

	// Create output channel
	outCh := make(chan domain.JobResult)

	// Spawn background listener
	go func() {
		defer close(outCh)
		defer pubsub.Close()

		ch := pubsub.Channel()

		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}

				var result domain.JobResult
				if err := json.Unmarshal([]byte(msg.Payload), &result); err != nil {
					slog.Error("Failed to unmarshal log", "error", err)
					continue
				}

				outCh <- result
			}
		}
	}()

	return outCh, nil
} 