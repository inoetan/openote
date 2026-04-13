package execution

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pgRepository struct {
	pool *pgxpool.Pool
}

// NewRepository returns a PostgreSQL-backed execution.Repository.
func NewRepository(pool *pgxpool.Pool) Repository {
	return &pgRepository{pool: pool}
}

const execColumns = `id, job_id, project_id, triggered_by, triggered_user, status, started_at, finished_at, duration_ms, worker_id, created_at`

func scanExecution(row pgx.Row) (*Execution, error) {
	e := &Execution{}
	err := row.Scan(
		&e.ID, &e.JobID, &e.ProjectID, &e.TriggeredBy,
		&e.TriggeredUser, &e.Status, &e.StartedAt, &e.FinishedAt,
		&e.DurationMs, &e.WorkerID, &e.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("scan execution: %w", err)
	}
	return e, nil
}

func (r *pgRepository) Create(ctx context.Context, e *Execution) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	e.CreatedAt = time.Now().UTC()

	_, err := r.pool.Exec(ctx,
		`INSERT INTO executions (id, job_id, project_id, triggered_by, triggered_user, status, started_at, finished_at, duration_ms, worker_id, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		e.ID, e.JobID, e.ProjectID, e.TriggeredBy, e.TriggeredUser,
		e.Status, e.StartedAt, e.FinishedAt, e.DurationMs, e.WorkerID, e.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert execution: %w", err)
	}
	return nil
}

func (r *pgRepository) FindByID(ctx context.Context, id uuid.UUID) (*Execution, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+execColumns+` FROM executions WHERE id = $1 LIMIT 1`,
		id,
	)
	return scanExecution(row)
}

func (r *pgRepository) ListByProject(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*Execution, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := r.pool.Query(ctx,
		`SELECT `+execColumns+` FROM executions WHERE project_id = $1
		 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		projectID, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("query executions: %w", err)
	}
	defer rows.Close()

	var result []*Execution
	for rows.Next() {
		e := &Execution{}
		if err := rows.Scan(
			&e.ID, &e.JobID, &e.ProjectID, &e.TriggeredBy,
			&e.TriggeredUser, &e.Status, &e.StartedAt, &e.FinishedAt,
			&e.DurationMs, &e.WorkerID, &e.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan execution row: %w", err)
		}
		result = append(result, e)
	}
	return result, rows.Err()
}

func (r *pgRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status ExecutionStatus, finishedAt *time.Time, durationMs *int) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE executions SET status=$1, finished_at=$2, duration_ms=$3 WHERE id=$4`,
		status, finishedAt, durationMs, id,
	)
	if err != nil {
		return fmt.Errorf("update execution status: %w", err)
	}
	return nil
}

func (r *pgRepository) CreateStep(ctx context.Context, s *StepExecution) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}

	_, err := r.pool.Exec(ctx,
		`INSERT INTO step_executions (id, execution_id, job_step_id, step_order, status, started_at, finished_at, exit_code, log_key)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		s.ID, s.ExecutionID, s.JobStepID, s.StepOrder, s.Status,
		s.StartedAt, s.FinishedAt, s.ExitCode, s.LogKey,
	)
	if err != nil {
		return fmt.Errorf("insert step_execution: %w", err)
	}
	return nil
}

func (r *pgRepository) ListSteps(ctx context.Context, executionID uuid.UUID) ([]*StepExecution, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, execution_id, job_step_id, step_order, status, started_at, finished_at, exit_code, log_key
		 FROM step_executions WHERE execution_id = $1 ORDER BY step_order`,
		executionID,
	)
	if err != nil {
		return nil, fmt.Errorf("query step_executions: %w", err)
	}
	defer rows.Close()

	var result []*StepExecution
	for rows.Next() {
		s := &StepExecution{}
		if err := rows.Scan(
			&s.ID, &s.ExecutionID, &s.JobStepID, &s.StepOrder, &s.Status,
			&s.StartedAt, &s.FinishedAt, &s.ExitCode, &s.LogKey,
		); err != nil {
			return nil, fmt.Errorf("scan step_execution row: %w", err)
		}
		result = append(result, s)
	}
	return result, rows.Err()
}
