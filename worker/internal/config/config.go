package config

import (
	"os"
	"strconv"
)

// Config holds all worker configuration loaded from environment variables.
type Config struct {
	// DBDsn is the PostgreSQL connection DSN.
	// Env var: DB_DSN
	DBDsn string

	// RedisURL is the Redis connection URL.
	// Env var: REDIS_URL
	RedisURL string

	// WorkerID uniquely identifies this worker instance.
	// Env var: WORKER_ID (defaults to hostname)
	WorkerID string

	// Concurrency is the number of parallel job executor goroutines.
	// Env var: WORKER_CONCURRENCY (defaults to 4)
	Concurrency int
}

// Load reads configuration from environment variables and returns a Config.
// Missing optional variables fall back to their defaults.
func Load() *Config {
	workerID := os.Getenv("WORKER_ID")
	if workerID == "" {
		if hostname, err := os.Hostname(); err == nil {
			workerID = hostname
		} else {
			workerID = "worker-unknown"
		}
	}

	return &Config{
		DBDsn:       os.Getenv("DB_DSN"),
		RedisURL:    os.Getenv("REDIS_URL"),
		WorkerID:    workerID,
		Concurrency: getEnvInt("WORKER_CONCURRENCY", 4),
	}
}

func getEnvInt(key string, defaultVal int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 {
		return defaultVal
	}
	return v
}
