package logstream

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// Writer is an io.Writer that persists log lines to PostgreSQL and
// broadcasts them in real-time over Redis PubSub.
//
// A single sequence counter is shared across both stdout and stderr writers
// for the same execution so that interleaved output has a total order.
type Writer struct {
	db          *pgxpool.Pool
	redis       *redis.Client
	executionID uuid.UUID
	stepExecID  uuid.UUID
	stream      string        // "stdout" or "stderr"
	sequence    *atomic.Int64 // shared counter across stdout+stderr

	// leftover holds an incomplete line carried over from a previous Write call.
	leftover []byte
}

// logMessage is the JSON payload published to the Redis PubSub channel.
type logMessage struct {
	Seq     int64     `json:"seq"`
	Stream  string    `json:"stream"`
	Content string    `json:"content"`
	Ts      time.Time `json:"ts"`
}

// New returns a Writer ready to receive bytes for the given execution/step/stream.
// seq must be shared between the stdout and stderr writers of the same execution.
func New(
	db *pgxpool.Pool,
	rdb *redis.Client,
	executionID uuid.UUID,
	stepExecID uuid.UUID,
	stream string,
	seq *atomic.Int64,
) *Writer {
	return &Writer{
		db:          db,
		redis:       rdb,
		executionID: executionID,
		stepExecID:  stepExecID,
		stream:      stream,
		sequence:    seq,
	}
}

// Write implements io.Writer.  It buffers partial lines across calls and
// flushes each complete newline-terminated line.
func (w *Writer) Write(p []byte) (int, error) {
	n := len(p)
	data := append(w.leftover, p...)
	w.leftover = nil

	for {
		idx := bytes.IndexByte(data, '\n')
		if idx < 0 {
			// No complete line yet – stash remainder.
			w.leftover = data
			break
		}
		line := data[:idx]
		data = data[idx+1:]

		if err := w.writeLine(context.Background(), string(line)); err != nil {
			// Best-effort: log errors are non-fatal to the process execution.
			fmt.Printf("logstream write error: %v\n", err)
		}
	}
	return n, nil
}

// Flush persists any buffered incomplete line (last line without trailing newline).
func (w *Writer) Flush() {
	if len(w.leftover) > 0 {
		if err := w.writeLine(context.Background(), string(w.leftover)); err != nil {
			fmt.Printf("logstream flush error: %v\n", err)
		}
		w.leftover = nil
	}
}

// writeLine persists one log line and publishes it to Redis.
func (w *Writer) writeLine(ctx context.Context, content string) error {
	seq := w.sequence.Add(1)
	now := time.Now().UTC()

	// 1. Persist to execution_logs table.
	_, err := w.db.Exec(ctx,
		`INSERT INTO execution_logs
		   (execution_id, step_exec_id, sequence, stream, content, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		w.executionID, w.stepExecID, seq, w.stream, content, now,
	)
	if err != nil {
		return fmt.Errorf("insert execution_log: %w", err)
	}

	// 2. Publish to Redis PubSub so connected clients get live output.
	msg := logMessage{
		Seq:     seq,
		Stream:  w.stream,
		Content: content,
		Ts:      now,
	}
	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal log message: %w", err)
	}

	channel := fmt.Sprintf("exec:%s:logs", w.executionID)
	if err := w.redis.Publish(ctx, channel, payload).Err(); err != nil {
		// Redis publish failure is non-fatal: logs are already persisted in DB.
		fmt.Printf("redis publish error on channel %s: %v\n", channel, err)
	}

	return nil
}
