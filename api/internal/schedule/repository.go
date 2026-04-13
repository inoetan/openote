package schedule

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

// NewRepository returns a PostgreSQL-backed schedule.Repository.
func NewRepository(pool *pgxpool.Pool) Repository {
	return &pgRepository{pool: pool}
}

const scheduleColumns = `id, job_id, cron_expr, timezone, is_enabled, next_run_at, last_run_at`

func scanSchedule(row pgx.Row) (*Schedule, error) {
	s := &Schedule{}
	err := row.Scan(
		&s.ID, &s.JobID, &s.CronExpr, &s.Timezone,
		&s.IsEnabled, &s.NextRunAt, &s.LastRunAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("scan schedule: %w", err)
	}
	return s, nil
}

func (r *pgRepository) FindByJobID(ctx context.Context, jobID uuid.UUID) (*Schedule, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+scheduleColumns+` FROM schedules WHERE job_id = $1 LIMIT 1`,
		jobID,
	)
	return scanSchedule(row)
}

func (r *pgRepository) FindByID(ctx context.Context, id uuid.UUID) (*Schedule, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+scheduleColumns+` FROM schedules WHERE id = $1 LIMIT 1`,
		id,
	)
	return scanSchedule(row)
}

func (r *pgRepository) Upsert(ctx context.Context, s *Schedule) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}

	_, err := r.pool.Exec(ctx,
		`INSERT INTO schedules (id, job_id, cron_expr, timezone, is_enabled, next_run_at, last_run_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 ON CONFLICT (job_id) DO UPDATE SET
		   cron_expr   = EXCLUDED.cron_expr,
		   timezone    = EXCLUDED.timezone,
		   is_enabled  = EXCLUDED.is_enabled,
		   next_run_at = EXCLUDED.next_run_at,
		   last_run_at = EXCLUDED.last_run_at`,
		s.ID, s.JobID, s.CronExpr, s.Timezone,
		s.IsEnabled, s.NextRunAt, s.LastRunAt,
	)
	if err != nil {
		return fmt.Errorf("upsert schedule: %w", err)
	}
	return nil
}

func (r *pgRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM schedules WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete schedule: %w", err)
	}
	return nil
}

func (r *pgRepository) ListEnabled(ctx context.Context) ([]*Schedule, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT s.`+scheduleColumns+`
		 FROM schedules s
		 JOIN jobs j ON j.id = s.job_id
		 WHERE s.is_enabled = true AND j.is_active = true
		 ORDER BY s.id`,
	)
	if err != nil {
		return nil, fmt.Errorf("query enabled schedules: %w", err)
	}
	defer rows.Close()

	var result []*Schedule
	for rows.Next() {
		s := &Schedule{}
		if err := rows.Scan(
			&s.ID, &s.JobID, &s.CronExpr, &s.Timezone,
			&s.IsEnabled, &s.NextRunAt, &s.LastRunAt,
		); err != nil {
			return nil, fmt.Errorf("scan schedule row: %w", err)
		}
		result = append(result, s)
	}
	return result, rows.Err()
}

func (r *pgRepository) UpdateLastRun(ctx context.Context, id uuid.UUID, lastRunAt, nextRunAt time.Time) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE schedules SET last_run_at = $1, next_run_at = $2 WHERE id = $3`,
		lastRunAt, nextRunAt, id,
	)
	if err != nil {
		return fmt.Errorf("update schedule last_run: %w", err)
	}
	return nil
}
