package schedule

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Schedule associates a cron expression with a job, controlling when it runs automatically.
type Schedule struct {
	ID        uuid.UUID  `json:"id"`
	JobID     uuid.UUID  `json:"job_id"`
	CronExpr  string     `json:"cron_expr"`
	Timezone  string     `json:"timezone"`
	IsEnabled bool       `json:"is_enabled"`
	NextRunAt *time.Time `json:"next_run_at,omitempty"`
	LastRunAt *time.Time `json:"last_run_at,omitempty"`
}

// Repository defines data-access for Schedule.
type Repository interface {
	// FindByJobID returns the schedule for the given job, or nil if none exists.
	FindByJobID(ctx context.Context, jobID uuid.UUID) (*Schedule, error)

	// FindByID returns a schedule by primary key.
	FindByID(ctx context.Context, id uuid.UUID) (*Schedule, error)

	// Upsert creates or replaces the schedule for a job.
	Upsert(ctx context.Context, s *Schedule) error

	// Delete removes a schedule by ID.
	Delete(ctx context.Context, id uuid.UUID) error

	// ListEnabled returns all schedules that are enabled and belong to active jobs.
	ListEnabled(ctx context.Context) ([]*Schedule, error)

	// UpdateLastRun records the last and next run times after a cron fires.
	UpdateLastRun(ctx context.Context, id uuid.UUID, lastRunAt, nextRunAt time.Time) error
}
