package execution

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ExecutionStatus represents the lifecycle state of an execution or step.
type ExecutionStatus string

const (
	StatusQueued   ExecutionStatus = "queued"
	StatusRunning  ExecutionStatus = "running"
	StatusSuccess  ExecutionStatus = "success"
	StatusFailed   ExecutionStatus = "failed"
	StatusAborted  ExecutionStatus = "aborted"
	StatusTimedOut ExecutionStatus = "timed_out"
)

// TriggerType identifies how an execution was initiated.
type TriggerType string

const (
	TriggerSchedule TriggerType = "schedule"
	TriggerManual   TriggerType = "manual"
	TriggerWebhook  TriggerType = "webhook"
	TriggerAPI      TriggerType = "api"
)

// Execution tracks a single run of a job.
type Execution struct {
	ID            uuid.UUID       `json:"id"`
	JobID         uuid.UUID       `json:"job_id"`
	ProjectID     uuid.UUID       `json:"project_id"`
	TriggeredBy   TriggerType     `json:"triggered_by"`
	TriggeredUser *uuid.UUID      `json:"triggered_user,omitempty"`
	Status        ExecutionStatus `json:"status"`
	StartedAt     *time.Time      `json:"started_at,omitempty"`
	FinishedAt    *time.Time      `json:"finished_at,omitempty"`
	DurationMs    *int            `json:"duration_ms,omitempty"`
	WorkerID      string          `json:"worker_id"`
	CreatedAt     time.Time       `json:"created_at"`
}

// StepExecution tracks a single step within an execution.
type StepExecution struct {
	ID          uuid.UUID       `json:"id"`
	ExecutionID uuid.UUID       `json:"execution_id"`
	JobStepID   uuid.UUID       `json:"job_step_id"`
	StepOrder   int             `json:"step_order"`
	Status      ExecutionStatus `json:"status"`
	StartedAt   *time.Time      `json:"started_at,omitempty"`
	FinishedAt  *time.Time      `json:"finished_at,omitempty"`
	ExitCode    *int            `json:"exit_code,omitempty"`
	LogKey      string          `json:"log_key"`
}

// Repository defines data-access for Execution and StepExecution.
type Repository interface {
	// Create inserts a new execution record.
	Create(ctx context.Context, e *Execution) error

	// FindByID returns an execution by primary key.
	FindByID(ctx context.Context, id uuid.UUID) (*Execution, error)

	// ListByProject returns paginated executions for a project.
	ListByProject(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*Execution, error)

	// UpdateStatus updates the status (and optionally finished_at, duration_ms) of an execution.
	UpdateStatus(ctx context.Context, id uuid.UUID, status ExecutionStatus, finishedAt *time.Time, durationMs *int) error

	// CreateStep inserts a new step execution record.
	CreateStep(ctx context.Context, s *StepExecution) error

	// ListSteps returns all step executions for an execution.
	ListSteps(ctx context.Context, executionID uuid.UUID) ([]*StepExecution, error)
}
