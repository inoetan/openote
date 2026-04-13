package job

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// NodeType identifies the kind of work a node definition performs.
type NodeType string

const (
	NodeTypeShellScript  NodeType = "shell_script"
	NodeTypeShellCommand NodeType = "shell_command"
	NodeTypeHTTPRequest  NodeType = "http_request"
)

// NodeDefinition describes a reusable task template within a project.
type NodeDefinition struct {
	ID          uuid.UUID              `json:"id"`
	ProjectID   uuid.UUID              `json:"project_id"`
	Name        string                 `json:"name"`
	Type        NodeType               `json:"type"`
	Description string                 `json:"description"`
	Config      map[string]interface{} `json:"config"`
	CreatedBy   uuid.UUID              `json:"created_by"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// WorkflowSpec is the full graph description stored as JSONB in the jobs table.
type WorkflowSpec struct {
	Nodes    []WorkflowNode         `json:"nodes"`
	Edges    []WorkflowEdge         `json:"edges"`
	Viewport map[string]interface{} `json:"viewport,omitempty"`
}

// WorkflowNode represents a single node in the visual workflow graph.
type WorkflowNode struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Position map[string]float64     `json:"position"`
	Data     map[string]interface{} `json:"data"`
}

// WorkflowEdge represents a directed connection between two nodes.
type WorkflowEdge struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
	Type   string `json:"type,omitempty"`
	Label  string `json:"label,omitempty"`
}

// OnFailure controls what happens when a job step fails.
type OnFailure string

const (
	OnFailureStop     OnFailure = "stop"
	OnFailureContinue OnFailure = "continue"
	OnFailureRetry    OnFailure = "retry"
)

// Job is the top-level schedulable unit of work.
type Job struct {
	ID            uuid.UUID    `json:"id"`
	ProjectID     uuid.UUID    `json:"project_id"`
	Name          string       `json:"name"`
	Description   string       `json:"description"`
	IsActive      bool         `json:"is_active"`
	TimeoutSecs   int          `json:"timeout_secs"`
	MaxConcurrent int          `json:"max_concurrent"`
	OnFailure     OnFailure    `json:"on_failure"`
	RetryCount    int          `json:"retry_count"`
	WorkflowSpec  WorkflowSpec `json:"workflow_spec"`
	CreatedBy     uuid.UUID    `json:"created_by"`
	CreatedAt     time.Time    `json:"created_at"`
	UpdatedAt     time.Time    `json:"updated_at"`
}

// JobStep is a materialized, ordered execution step derived from WorkflowSpec.
type JobStep struct {
	ID        uuid.UUID              `json:"id"`
	JobID     uuid.UUID              `json:"job_id"`
	NodeDefID uuid.UUID              `json:"node_def_id"`
	StepOrder int                    `json:"step_order"`
	DependsOn []uuid.UUID            `json:"depends_on"`
	Overrides map[string]interface{} `json:"overrides"`
	Label     string                 `json:"label"`
}

// NodeRepository defines data-access for NodeDefinition.
type NodeRepository interface {
	ListByProject(ctx context.Context, projectID uuid.UUID) ([]*NodeDefinition, error)
	Create(ctx context.Context, nd *NodeDefinition) error
	FindByID(ctx context.Context, id uuid.UUID) (*NodeDefinition, error)
	Update(ctx context.Context, nd *NodeDefinition) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// JobRepository defines data-access for Job and its materialized steps.
type JobRepository interface {
	ListByProject(ctx context.Context, projectID uuid.UUID) ([]*Job, error)
	Create(ctx context.Context, j *Job) error
	FindByID(ctx context.Context, id uuid.UUID) (*Job, error)
	Update(ctx context.Context, j *Job) error
	Delete(ctx context.Context, id uuid.UUID) error
	ListSteps(ctx context.Context, jobID uuid.UUID) ([]*JobStep, error)
}
