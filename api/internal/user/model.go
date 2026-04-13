package user

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// User represents a system user account.
type User struct {
	ID           uuid.UUID
	Username     string
	Email        string
	PasswordHash string
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Repository defines the data-access contract for User persistence.
type Repository interface {
	// FindByUsername returns the user with the given username, or nil if not found.
	FindByUsername(ctx context.Context, username string) (*User, error)

	// FindByID returns the user with the given string UUID, or nil if not found.
	FindByID(ctx context.Context, id string) (*User, error)

	// Create inserts a new user into the store.
	Create(ctx context.Context, u *User) error

	// Update modifies an existing user record.
	Update(ctx context.Context, u *User) error

	// Delete removes a user by ID.
	Delete(ctx context.Context, id uuid.UUID) error

	// List returns all users.
	List(ctx context.Context) ([]*User, error)
}
