package audit

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// AuditEvent records an immutable log of a user action on a resource.
type AuditEvent struct {
	ID           int64                  `json:"id"`
	ProjectID    *uuid.UUID             `json:"project_id,omitempty"`
	UserID       *uuid.UUID             `json:"user_id,omitempty"`
	Action       string                 `json:"action"`
	ResourceType string                 `json:"resource_type"`
	ResourceID   *uuid.UUID             `json:"resource_id,omitempty"`
	OldValue     map[string]interface{} `json:"old_value,omitempty"`
	NewValue     map[string]interface{} `json:"new_value,omitempty"`
	IPAddress    string                 `json:"ip_address"`
	UserAgent    string                 `json:"user_agent"`
	OccurredAt   time.Time              `json:"occurred_at"`
}

// Repository defines append-only data-access for AuditEvent.
type Repository interface {
	// Append inserts a new audit event. No updates or deletes are permitted.
	Append(ctx context.Context, event *AuditEvent) error

	// ListByProject returns a paginated slice of audit events for a project.
	ListByProject(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*AuditEvent, error)
}
