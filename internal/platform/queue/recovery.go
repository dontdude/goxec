package queue

import (
	"context"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

// StartRecoveryRoutine polls the PEL for stale jobs and reclaims them.
func (r *RedisQueue) StartRecoveryRoutine(ctx context.Context, interval time.Duration, maxAge time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Unique consumer ID for the recovery agent
	consumerName := "recovery-agent"

	slog.Info("Starting Redis Recovery Routine", "interval", interval, "maxAge", maxAge)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// XAUTOCLAIM: Finds messages pending for > maxAge
			// and claims them to this consumer to be processed.
			start := "-" // Start from beginning of stream
			
			for {
				// We claim batches of 10
				messages, nextStart, err := r.client.XAutoClaim(ctx, &redis.XAutoClaimArgs{
					Stream:   r.stream,
					Group:    r.group,
					MinIdle:  maxAge,
					Start:    start,
					Count:    10,
					Consumer: consumerName,
				}).Result()

				if err != nil {
					slog.Error("Recovery routine failed", "error", err)
					break
				}
				
				if len(messages) == 0 {
					break // No more stale messages
				}

				slog.Info("Recovered stale jobs", "count", len(messages))

				// Process recovered messages.
				// In a robust production system, you would:
				// 1. Inspect the retry count (XDeliveryCount).
				// 2. If retry count > MaxRetries, move to Dead Letter Queue (DLQ).
				// 3. Else, re-enqueue or process immediately.
				for _, msg := range messages {
					slog.Warn("Stale job claimed by recovery agent", "msgID", msg.ID)
					
					// For this demonstration, we ACK the message to remove it from the PEL
					// so it doesn't leak memory. In a real system, you might restart the job here.
					r.client.XAck(ctx, r.stream, r.group, msg.ID)
				}
				
				start = nextStart
				if start == "0-0" {
					break
				}
			}
		}
	}
}
