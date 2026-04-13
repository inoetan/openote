package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pgRepository struct {
	pool *pgxpool.Pool
}

// NewRepository returns a PostgreSQL-backed, append-only audit.Repository.
func NewRepository(pool *pgxpool.Pool) Repository {
	return &pgRepository{pool: pool}
}

// Append inserts a new audit event. This is the only write operation allowed.
func (r *pgRepository) Append(ctx context.Context, event *AuditEvent) error {
	event.OccurredAt = time.Now().UTC()

	oldValueJSON, err := json.Marshal(event.OldValue)
	if err != nil {
		return fmt.Errorf("marshal old_value: %w", err)
	}
	newValueJSON, err := json.Marshal(event.NewValue)
	if err != nil {
		return fmt.Errorf("marshal new_value: %w", err)
	}

	err = r.pool.QueryRow(ctx,
		`INSERT INTO audit_events (project_id, user_id, action, resource_type, resource_id, old_value, new_value, ip_address, user_agent, occurred_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		 RETURNING id`,
		event.ProjectID, event.UserID, event.Action, event.ResourceType, event.ResourceID,
		oldValueJSON, newValueJSON, event.IPAddress, event.UserAgent, event.OccurredAt,
	).Scan(&event.ID)
	if err != nil {
		return fmt.Errorf("insert audit_event: %w", err)
	}
	return nil
}

// ListByProject returns a paginated list of audit events for a project,
// ordered by most recent first.
func (r *pgRepository) ListByProject(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*AuditEvent, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := r.pool.Query(ctx,
		`SELECT id, project_id, user_id, action, resource_type, resource_id,
		        old_value, new_value, ip_address, user_agent, occurred_at
		 FROM audit_events
		 WHERE project_id = $1
		 ORDER BY occurred_at DESC
		 LIMIT $2 OFFSET $3`,
		projectID, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("query audit_events: %w", err)
	}
	defer rows.Close()

	var result []*AuditEvent
	for rows.Next() {
		e := &AuditEvent{}
		var oldValueJSON, newValueJSON []byte
		if err := rows.Scan(
			&e.ID, &e.ProjectID, &e.UserID, &e.Action, &e.ResourceType, &e.ResourceID,
			&oldValueJSON, &newValueJSON, &e.IPAddress, &e.UserAgent, &e.OccurredAt,
		); err != nil {
			return nil, fmt.Errorf("scan audit_event row: %w", err)
		}
		if len(oldValueJSON) > 0 && string(oldValueJSON) != "null" {
			if err := json.Unmarshal(oldValueJSON, &e.OldValue); err != nil {
				return nil, fmt.Errorf("unmarshal old_value: %w", err)
			}
		}
		if len(newValueJSON) > 0 && string(newValueJSON) != "null" {
			if err := json.Unmarshal(newValueJSON, &e.NewValue); err != nil {
				return nil, fmt.Errorf("unmarshal new_value: %w", err)
			}
		}
		result = append(result, e)
	}
	return result, rows.Err()
}
