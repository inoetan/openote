package notification

import (
	"context"

	"github.com/google/uuid"
)

// NotificationChannel identifies the delivery mechanism for notifications.
type NotificationChannel string

const (
	ChannelEmail          NotificationChannel = "email"
	ChannelSlackWebhook   NotificationChannel = "slack_webhook"
	ChannelGenericWebhook NotificationChannel = "generic_webhook"
)

// NotificationEvent identifies which execution lifecycle event triggers a notification.
type NotificationEvent string

const (
	EventOnSuccess NotificationEvent = "on_success"
	EventOnFailure NotificationEvent = "on_failure"
	EventOnStart   NotificationEvent = "on_start"
	EventOnTimeout NotificationEvent = "on_timeout"
)

// NotificationRule defines when and how to send a notification for a job.
type NotificationRule struct {
	ID      uuid.UUID              `json:"id"`
	JobID   uuid.UUID              `json:"job_id"`
	Event   NotificationEvent      `json:"event"`
	Channel NotificationChannel    `json:"channel"`
	Config  map[string]interface{} `json:"config"` // {recipients, url, headers, template}
}

// Repository defines data-access for NotificationRule.
type Repository interface {
	// ListByJob returns all notification rules for a given job.
	ListByJob(ctx context.Context, jobID uuid.UUID) ([]*NotificationRule, error)

	// Create inserts a new notification rule.
	Create(ctx context.Context, rule *NotificationRule) error

	// FindByID returns a notification rule by primary key.
	FindByID(ctx context.Context, id uuid.UUID) (*NotificationRule, error)

	// Update replaces a notification rule's fields.
	Update(ctx context.Context, rule *NotificationRule) error

	// Delete removes a notification rule by ID.
	Delete(ctx context.Context, id uuid.UUID) error
}
