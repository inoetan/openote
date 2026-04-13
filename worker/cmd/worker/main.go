package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/inoetan/openote/worker/internal/config"
	"github.com/inoetan/openote/worker/internal/consumer"
	"github.com/inoetan/openote/worker/internal/executor"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	shutdownTimeout = 30 * time.Second
)

func main() {
	cfg := config.Load()
	if err := run(cfg); err != nil {
		log.Fatalf("worker: fatal error: %v", err)
	}
}

func run(cfg *config.Config) error {
	log.Printf("worker starting (id=%s concurrency=%d)", cfg.WorkerID, cfg.Concurrency)

	// --- Connect to PostgreSQL ---
	ctx := context.Background()

	db, err := newDBPool(ctx, cfg.DBDsn)
	if err != nil {
		return fmt.Errorf("connect postgres: %w", err)
	}
	defer db.Close()
	log.Println("worker: postgres connected")

	// --- Connect to Redis ---
	rdb, err := newRedisClient(ctx, cfg.RedisURL)
	if err != nil {
		return fmt.Errorf("connect redis: %w", err)
	}
	defer func() { _ = rdb.Close() }()
	log.Println("worker: redis connected")

	// --- Ensure consumer group exists ---
	if err := ensureConsumerGroup(ctx, rdb); err != nil {
		return fmt.Errorf("ensure consumer group: %w", err)
	}

	// --- Build shared executor and consumer factory ---
	exec := executor.New(db, rdb)

	// --- Graceful shutdown setup ---
	rootCtx, rootCancel := context.WithCancel(ctx)
	defer rootCancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

	// --- Start N worker goroutines ---
	var wg sync.WaitGroup
	for i := 0; i < cfg.Concurrency; i++ {
		workerName := fmt.Sprintf("%s-%d", cfg.WorkerID, i)
		c := consumer.New(rdb, db, exec, workerName)
		wg.Add(1)
		go func(c *consumer.Consumer, name string) {
			defer wg.Done()
			log.Printf("worker goroutine %s started", name)
			if err := c.Run(rootCtx); err != nil && rootCtx.Err() == nil {
				log.Printf("worker goroutine %s error: %v", name, err)
			}
			log.Printf("worker goroutine %s stopped", name)
		}(c, workerName)
	}

	// --- Block until signal ---
	sig := <-sigCh
	log.Printf("worker: received signal %s – starting graceful shutdown", sig)
	rootCancel()

	// Wait for goroutines with a hard deadline.
	shutdownDone := make(chan struct{})
	go func() {
		wg.Wait()
		close(shutdownDone)
	}()

	select {
	case <-shutdownDone:
		log.Println("worker: all goroutines stopped cleanly")
	case <-time.After(shutdownTimeout):
		log.Println("worker: shutdown timeout exceeded – forcing exit")
	}

	return nil
}

// newDBPool creates a pgxpool.Pool with sensible defaults.
func newDBPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse pgx config: %w", err)
	}
	cfg.MaxConns = 20
	cfg.MinConns = 2
	cfg.MaxConnLifetime = 30 * time.Minute
	cfg.MaxConnIdleTime = 10 * time.Minute
	cfg.HealthCheckPeriod = 1 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create pgx pool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return pool, nil
}

// newRedisClient parses the Redis URL and verifies connectivity.
func newRedisClient(ctx context.Context, url string) (*redis.Client, error) {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}
	client := redis.NewClient(opts)

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if _, err := client.Ping(pingCtx).Result(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("ping redis: %w", err)
	}
	return client, nil
}

// ensureConsumerGroup creates the consumer group and stream (MKSTREAM) if they
// do not yet exist.  BUSYGROUP errors (group already exists) are ignored.
func ensureConsumerGroup(ctx context.Context, rdb *redis.Client) error {
	err := rdb.XGroupCreateMkStream(ctx, "executions:queue", "workers", "$").Err()
	if err != nil && !isBusyGroupError(err) {
		return fmt.Errorf("XGROUP CREATE: %w", err)
	}
	return nil
}

// isBusyGroupError reports whether err is the Redis BUSYGROUP error that is
// returned when the consumer group already exists.
func isBusyGroupError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "BUSYGROUP")
}
