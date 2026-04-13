package project

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// pgRepository is the PostgreSQL implementation of project.Repository.
type pgRepository struct {
	pool *pgxpool.Pool
}

// NewRepository returns a new PostgreSQL-backed project.Repository.
func NewRepository(pool *pgxpool.Pool) Repository {
	return &pgRepository{pool: pool}
}

const projectColumns = `id, name, description, created_by, created_at, updated_at`

func scanProject(row pgx.Row) (*Project, error) {
	p := &Project{}
	err := row.Scan(
		&p.ID,
		&p.Name,
		&p.Description,
		&p.CreatedBy,
		&p.CreatedAt,
		&p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("scan project: %w", err)
	}
	return p, nil
}

// List returns all projects visible to the given userID.
// Currently returns all projects; scope by user_roles in a future iteration.
func (r *pgRepository) List(ctx context.Context, userID uuid.UUID) ([]*Project, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+projectColumns+` FROM projects ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("query projects: %w", err)
	}
	defer rows.Close()

	var projects []*Project
	for rows.Next() {
		p := &Project{}
		if err := rows.Scan(
			&p.ID, &p.Name, &p.Description, &p.CreatedBy,
			&p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan project row: %w", err)
		}
		projects = append(projects, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	return projects, nil
}

// Create inserts a new project.
func (r *pgRepository) Create(ctx context.Context, p *Project) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	now := time.Now().UTC()
	p.CreatedAt = now
	p.UpdatedAt = now

	_, err := r.pool.Exec(ctx,
		`INSERT INTO projects (id, name, description, created_by, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		p.ID, p.Name, p.Description, p.CreatedBy, p.CreatedAt, p.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert project: %w", err)
	}
	return nil
}

// FindByID returns a project by its primary key.
func (r *pgRepository) FindByID(ctx context.Context, id uuid.UUID) (*Project, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+projectColumns+` FROM projects WHERE id = $1 LIMIT 1`,
		id,
	)
	return scanProject(row)
}

// Update modifies an existing project.
func (r *pgRepository) Update(ctx context.Context, p *Project) error {
	p.UpdatedAt = time.Now().UTC()
	_, err := r.pool.Exec(ctx,
		`UPDATE projects SET name=$1, description=$2, updated_at=$3 WHERE id=$4`,
		p.Name, p.Description, p.UpdatedAt, p.ID,
	)
	if err != nil {
		return fmt.Errorf("update project: %w", err)
	}
	return nil
}

// Delete removes a project by ID.
func (r *pgRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM projects WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete project: %w", err)
	}
	return nil
}
