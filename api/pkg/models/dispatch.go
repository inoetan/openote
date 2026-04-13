package models

// JobDispatch is the payload pushed to the Redis Stream when a job needs execution.
// It is shared between the API server and the worker process.
type JobDispatch struct {
	ExecutionID string `json:"execution_id"`
	JobID       string `json:"job_id"`
	ProjectID   string `json:"project_id"`
}

const (
	// RedisStreamKey is the Redis Stream key where job dispatch messages are published.
	RedisStreamKey = "executions:queue"

	// RedisStreamGroup is the consumer group name used by workers.
	RedisStreamGroup = "workers"

	// RedisPubSubPrefix is the prefix for execution-specific PubSub channels.
	// Full channel: exec:<execution_id>
	RedisPubSubPrefix = "exec:"

	// RedisSchedulerLock is the Redis key used as a distributed lock so only
	// one API instance fires scheduled jobs at a time.
	RedisSchedulerLock = "scheduler:leader"
)
