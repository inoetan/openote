package project

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Project represents an organizational namespace that groups jobs and resources.
type Project struct {
	ID          uuid.UUID
	Name        string
	Description string
	CreatedBy   uuid.UUID
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Repository defines the data-access contract for Project persistence.
type Repository interface {
	// List returns all projects visible to the given userID.
	List(ctx context.Context, userID uuid.UUID) ([]*Project, error)

	// Create inserts a new project.
	Create(ctx context.Context, p *Project) error

	// FindByID returns a project by its primary key.
	FindByID(ctx context.Context, id uuid.UUID) (*Project, error)

	// Update modifies an existing project.
	Update(ctx context.Context, p *Project) error

	// Delete removes a project by ID.
	Delete(ctx context.Context, id uuid.UUID) error
}
