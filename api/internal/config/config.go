package config

import (
	"os"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	// DBDsn is the PostgreSQL connection DSN.
	// Env var: DB_DSN
	DBDsn string

	// RedisURL is the Redis connection URL.
	// Env var: REDIS_URL
	RedisURL string

	// JWTSecret is the secret used to sign and verify JWT tokens.
	// Env var: JWT_SECRET
	JWTSecret string

	// Port is the HTTP server listen port (default: "8080").
	// Env var: PORT
	Port string

	// MigrationsPath is the filesystem path to SQL migration files (default: "migrations").
	// Env var: MIGRATIONS_PATH
	MigrationsPath string
}

// Load reads configuration from environment variables and returns a Config.
// Missing optional variables fall back to their defaults.
func Load() *Config {
	cfg := &Config{
		DBDsn:          os.Getenv("DB_DSN"),
		RedisURL:       os.Getenv("REDIS_URL"),
		JWTSecret:      os.Getenv("JWT_SECRET"),
		Port:           getEnvOrDefault("PORT", "8080"),
		MigrationsPath: getEnvOrDefault("MIGRATIONS_PATH", "migrations"),
	}
	return cfg
}

func getEnvOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
