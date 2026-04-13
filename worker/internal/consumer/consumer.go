package consumer

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/inoetan/openote/worker/internal/executor"
)

const (
	// streamKey is the Redis Stream name where the API publishes job dispatch events.
	streamKey = "executions:queue"

	// groupName is the consumer group shared across all worker instances.
	groupName = "workers"

	// readCount is the maximum number of messages fetched per XREADGROUP call.
	readCount = 1

	// blockDuration is how long XREADGROUP will block when no messages are ready.
	blockDuration = 5 * time.Second

	// pendingClaimAge is the minimum idle time before a pending message is
	// reclaimed from a crashed/stalled worker on startup.
	pendingClaimAge = 60 * time.Second
)

// Consumer reads job dispatch messages from a Redis Stream consumer group,
// delegates execution to the Executor, and ACKs each message.
type Consumer struct {
	redis    *redis.Client
	db       *pgxpool.Pool
	executor *executor.Executor
	workerID string
}

// New returns an initialized Consumer.
func New(
	rdb *redis.Client,
	db *pgxpool.Pool,
	exec *executor.Executor,
	workerID string,
) *Consumer {
	return &Consumer{
		redis:    rdb,
		db:       db,
		executor: exec,
		workerID: workerID,
	}
}

// Run blocks, reading messages from the stream until ctx is cancelled.
// It first calls reclaimPending to recover any jobs that were in-flight on a
// previously crashed worker.
func (c *Consumer) Run(ctx context.Context) error {
	// Recover pending messages from this or other stalled workers.
	if err := c.reclaimPending(ctx); err != nil {
		// Non-fatal: log and continue with normal consumption.
		log.Printf("consumer: reclaim pending error: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		msgs, err := c.redis.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    groupName,
			Consumer: c.workerID,
			Streams:  []string{streamKey, ">"},
			Count:    readCount,
			Block:    blockDuration,
		}).Result()

		if err != nil {
			if errors.Is(err, redis.Nil) {
				// Timeout with no messages – loop and try again.
				continue
			}
			if ctx.Err() != nil {
				// Context cancelled during block – clean shutdown.
				return ctx.Err()
			}
			log.Printf("consumer: XREADGROUP error: %v", err)
			// Brief back-off before retrying to avoid tight error loops.
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(2 * time.Second):
			}
			continue
		}

		for _, stream := range msgs {
			for _, msg := range stream.Messages {
				c.processMessage(ctx, msg)
			}
		}
	}
}

// processMessage parses a single stream message, calls Execute, and ACKs it.
// Failures are recorded in the DB by the executor; we always ACK to prevent
// infinite re-delivery of broken messages.
func (c *Consumer) processMessage(ctx context.Context, msg redis.XMessage) {
	req, err := parseDispatch(msg, c.workerID)
	if err != nil {
		log.Printf("consumer: parse message %s: %v – ACKing to discard", msg.ID, err)
		c.ack(ctx, msg.ID)
		return
	}

	log.Printf("consumer: executing job %s (execution %s) from message %s",
		req.JobID, req.ExecutionID, msg.ID)

	if execErr := c.executor.Execute(ctx, req); execErr != nil {
		// Execution failure is already recorded in PostgreSQL by the executor.
		// We still ACK so the message is not re-delivered.
		log.Printf("consumer: execution %s error: %v", req.ExecutionID, execErr)
	}

	c.ack(ctx, msg.ID)
}

// ack acknowledges a processed message so it is removed from the PEL.
func (c *Consumer) ack(ctx context.Context, msgID string) {
	if err := c.redis.XAck(ctx, streamKey, groupName, msgID).Err(); err != nil {
		log.Printf("consumer: XACK %s error: %v", msgID, err)
	}
}

// reclaimPending uses XAUTOCLAIM (or XCLAIM fallback) to take ownership of
// messages that have been pending for longer than pendingClaimAge.  This
// handles recovery when a worker crashed mid-execution.
func (c *Consumer) reclaimPending(ctx context.Context) error {
	// XAUTOCLAIM is available in Redis 6.2+.  We use it here; for older Redis
	// the function returns gracefully on NOSCRIPT / unknown command errors.
	start := "0-0"
	for {
		res, err := c.redis.XAutoClaim(ctx, &redis.XAutoClaimArgs{
			Stream:   streamKey,
			Group:    groupName,
			Consumer: c.workerID,
			MinIdle:  pendingClaimAge,
			Start:    start,
			Count:    100,
		}).Result()
		if err != nil {
			// XAUTOCLAIM is not available on Redis < 6.2 – not fatal.
			log.Printf("consumer: XAUTOCLAIM unavailable or error (skipping recovery): %v", err)
			return nil
		}

		for _, msg := range res.Messages {
			log.Printf("consumer: reclaiming stale message %s", msg.ID)
			c.processMessage(ctx, msg)
		}

		// res.NextStartID == "0-0" means we've consumed all pending entries.
		if res.NextStartID == "0-0" || len(res.Messages) == 0 {
			break
		}
		start = res.NextStartID
	}
	return nil
}

// parseDispatch extracts an ExecutionRequest from a Redis Stream message.
// The message values must contain execution_id, job_id, and project_id fields.
func parseDispatch(msg redis.XMessage, workerID string) (executor.ExecutionRequest, error) {
	get := func(key string) (string, error) {
		v, ok := msg.Values[key]
		if !ok {
			return "", fmt.Errorf("missing field %q", key)
		}
		s, ok := v.(string)
		if !ok {
			return "", fmt.Errorf("field %q is not a string", key)
		}
		return s, nil
	}

	execIDStr, err := get("execution_id")
	if err != nil {
		return executor.ExecutionRequest{}, err
	}
	jobIDStr, err := get("job_id")
	if err != nil {
		return executor.ExecutionRequest{}, err
	}
	projectIDStr, err := get("project_id")
	if err != nil {
		return executor.ExecutionRequest{}, err
	}

	execID, err := uuid.Parse(execIDStr)
	if err != nil {
		return executor.ExecutionRequest{}, fmt.Errorf("parse execution_id: %w", err)
	}
	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		return executor.ExecutionRequest{}, fmt.Errorf("parse job_id: %w", err)
	}
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		return executor.ExecutionRequest{}, fmt.Errorf("parse project_id: %w", err)
	}

	return executor.ExecutionRequest{
		ExecutionID: execID,
		JobID:       jobID,
		ProjectID:   projectID,
		WorkerID:    workerID,
	}, nil
}
