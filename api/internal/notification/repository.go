package notification

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pgRepository struct {
	pool *pgxpool.Pool
}

// NewRepository returns a PostgreSQL-backed notification.Repository.
func NewRepository(pool *pgxpool.Pool) Repository {
	return &pgRepository{pool: pool}
}

const ruleColumns = `id, job_id, event, channel, config`

func scanRule(row pgx.Row) (*NotificationRule, error) {
	r := &NotificationRule{}
	var configJSON []byte
	err := row.Scan(&r.ID, &r.JobID, &r.Event, &r.Channel, &configJSON)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("scan notification_rule: %w", err)
	}
	if len(configJSON) > 0 {
		if err := json.Unmarshal(configJSON, &r.Config); err != nil {
			return nil, fmt.Errorf("unmarshal rule config: %w", err)
		}
	}
	return r, nil
}

func (r *pgRepository) ListByJob(ctx context.Context, jobID uuid.UUID) ([]*NotificationRule, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+ruleColumns+` FROM notification_rules WHERE job_id = $1`,
		jobID,
	)
	if err != nil {
		return nil, fmt.Errorf("query notification_rules: %w", err)
	}
	defer rows.Close()

	var result []*NotificationRule
	for rows.Next() {
		rule := &NotificationRule{}
		var configJSON []byte
		if err := rows.Scan(&rule.ID, &rule.JobID, &rule.Event, &rule.Channel, &configJSON); err != nil {
			return nil, fmt.Errorf("scan rule row: %w", err)
		}
		if len(configJSON) > 0 {
			if err := json.Unmarshal(configJSON, &rule.Config); err != nil {
				return nil, fmt.Errorf("unmarshal rule config: %w", err)
			}
		}
		result = append(result, rule)
	}
	return result, rows.Err()
}

func (r *pgRepository) Create(ctx context.Context, rule *NotificationRule) error {
	if rule.ID == uuid.Nil {
		rule.ID = uuid.New()
	}

	configJSON, err := json.Marshal(rule.Config)
	if err != nil {
		return fmt.Errorf("marshal rule config: %w", err)
	}

	_, err = r.pool.Exec(ctx,
		`INSERT INTO notification_rules (id, job_id, event, channel, config)
		 VALUES ($1, $2, $3, $4, $5)`,
		rule.ID, rule.JobID, rule.Event, rule.Channel, configJSON,
	)
	if err != nil {
		return fmt.Errorf("insert notification_rule: %w", err)
	}
	return nil
}

func (r *pgRepository) FindByID(ctx context.Context, id uuid.UUID) (*NotificationRule, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+ruleColumns+` FROM notification_rules WHERE id = $1 LIMIT 1`,
		id,
	)
	return scanRule(row)
}

func (r *pgRepository) Update(ctx context.Context, rule *NotificationRule) error {
	configJSON, err := json.Marshal(rule.Config)
	if err != nil {
		return fmt.Errorf("marshal rule config: %w", err)
	}

	_, err = r.pool.Exec(ctx,
		`UPDATE notification_rules SET event=$1, channel=$2, config=$3 WHERE id=$4`,
		rule.Event, rule.Channel, configJSON, rule.ID,
	)
	if err != nil {
		return fmt.Errorf("update notification_rule: %w", err)
	}
	return nil
}

func (r *pgRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM notification_rules WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete notification_rule: %w", err)
	}
	return nil
}
