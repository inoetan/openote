package job

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ---- NodeDefinition repository ----

type pgNodeRepository struct {
	pool *pgxpool.Pool
}

// NewNodeRepository returns a PostgreSQL-backed NodeRepository.
func NewNodeRepository(pool *pgxpool.Pool) NodeRepository {
	return &pgNodeRepository{pool: pool}
}

const nodeColumns = `id, project_id, name, type, description, config, created_by, created_at, updated_at`

func scanNodeDef(row pgx.Row) (*NodeDefinition, error) {
	nd := &NodeDefinition{}
	var configJSON []byte
	err := row.Scan(
		&nd.ID, &nd.ProjectID, &nd.Name, &nd.Type,
		&nd.Description, &configJSON, &nd.CreatedBy,
		&nd.CreatedAt, &nd.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("scan node_definition: %w", err)
	}
	if len(configJSON) > 0 {
		if err := json.Unmarshal(configJSON, &nd.Config); err != nil {
			return nil, fmt.Errorf("unmarshal node config: %w", err)
		}
	}
	return nd, nil
}

func (r *pgNodeRepository) ListByProject(ctx context.Context, projectID uuid.UUID) ([]*NodeDefinition, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+nodeColumns+` FROM node_definitions WHERE project_id = $1 ORDER BY created_at DESC`,
		projectID,
	)
	if err != nil {
		return nil, fmt.Errorf("query node_definitions: %w", err)
	}
	defer rows.Close()

	var result []*NodeDefinition
	for rows.Next() {
		nd := &NodeDefinition{}
		var configJSON []byte
		if err := rows.Scan(
			&nd.ID, &nd.ProjectID, &nd.Name, &nd.Type,
			&nd.Description, &configJSON, &nd.CreatedBy,
			&nd.CreatedAt, &nd.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan node row: %w", err)
		}
		if len(configJSON) > 0 {
			if err := json.Unmarshal(configJSON, &nd.Config); err != nil {
				return nil, fmt.Errorf("unmarshal node config: %w", err)
			}
		}
		result = append(result, nd)
	}
	return result, rows.Err()
}

func (r *pgNodeRepository) Create(ctx context.Context, nd *NodeDefinition) error {
	if nd.ID == uuid.Nil {
		nd.ID = uuid.New()
	}
	now := time.Now().UTC()
	nd.CreatedAt = now
	nd.UpdatedAt = now

	configJSON, err := json.Marshal(nd.Config)
	if err != nil {
		return fmt.Errorf("marshal node config: %w", err)
	}

	_, err = r.pool.Exec(ctx,
		`INSERT INTO node_definitions (id, project_id, name, type, description, config, created_by, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		nd.ID, nd.ProjectID, nd.Name, nd.Type, nd.Description,
		configJSON, nd.CreatedBy, nd.CreatedAt, nd.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert node_definition: %w", err)
	}
	return nil
}

func (r *pgNodeRepository) FindByID(ctx context.Context, id uuid.UUID) (*NodeDefinition, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+nodeColumns+` FROM node_definitions WHERE id = $1 LIMIT 1`,
		id,
	)
	return scanNodeDef(row)
}

func (r *pgNodeRepository) Update(ctx context.Context, nd *NodeDefinition) error {
	nd.UpdatedAt = time.Now().UTC()
	configJSON, err := json.Marshal(nd.Config)
	if err != nil {
		return fmt.Errorf("marshal node config: %w", err)
	}
	_, err = r.pool.Exec(ctx,
		`UPDATE node_definitions SET name=$1, type=$2, description=$3, config=$4, updated_at=$5 WHERE id=$6`,
		nd.Name, nd.Type, nd.Description, configJSON, nd.UpdatedAt, nd.ID,
	)
	if err != nil {
		return fmt.Errorf("update node_definition: %w", err)
	}
	return nil
}

func (r *pgNodeRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM node_definitions WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete node_definition: %w", err)
	}
	return nil
}

// ---- Job repository ----

type pgJobRepository struct {
	pool *pgxpool.Pool
}

// NewJobRepository returns a PostgreSQL-backed JobRepository.
func NewJobRepository(pool *pgxpool.Pool) JobRepository {
	return &pgJobRepository{pool: pool}
}

const jobColumns = `id, project_id, name, description, is_active, timeout_secs, max_concurrent, on_failure, retry_count, workflow_spec, created_by, created_at, updated_at`

func scanJob(row pgx.Row) (*Job, error) {
	j := &Job{}
	var specJSON []byte
	err := row.Scan(
		&j.ID, &j.ProjectID, &j.Name, &j.Description,
		&j.IsActive, &j.TimeoutSecs, &j.MaxConcurrent,
		&j.OnFailure, &j.RetryCount, &specJSON,
		&j.CreatedBy, &j.CreatedAt, &j.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("scan job: %w", err)
	}
	if len(specJSON) > 0 {
		if err := json.Unmarshal(specJSON, &j.WorkflowSpec); err != nil {
			return nil, fmt.Errorf("unmarshal workflow_spec: %w", err)
		}
	}
	return j, nil
}

func (r *pgJobRepository) ListByProject(ctx context.Context, projectID uuid.UUID) ([]*Job, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+jobColumns+` FROM jobs WHERE project_id = $1 ORDER BY created_at DESC`,
		projectID,
	)
	if err != nil {
		return nil, fmt.Errorf("query jobs: %w", err)
	}
	defer rows.Close()

	var result []*Job
	for rows.Next() {
		j := &Job{}
		var specJSON []byte
		if err := rows.Scan(
			&j.ID, &j.ProjectID, &j.Name, &j.Description,
			&j.IsActive, &j.TimeoutSecs, &j.MaxConcurrent,
			&j.OnFailure, &j.RetryCount, &specJSON,
			&j.CreatedBy, &j.CreatedAt, &j.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan job row: %w", err)
		}
		if len(specJSON) > 0 {
			if err := json.Unmarshal(specJSON, &j.WorkflowSpec); err != nil {
				return nil, fmt.Errorf("unmarshal workflow_spec: %w", err)
			}
		}
		result = append(result, j)
	}
	return result, rows.Err()
}

func (r *pgJobRepository) Create(ctx context.Context, j *Job) error {
	if j.ID == uuid.Nil {
		j.ID = uuid.New()
	}
	now := time.Now().UTC()
	j.CreatedAt = now
	j.UpdatedAt = now

	specJSON, err := json.Marshal(j.WorkflowSpec)
	if err != nil {
		return fmt.Errorf("marshal workflow_spec: %w", err)
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	_, err = tx.Exec(ctx,
		`INSERT INTO jobs (id, project_id, name, description, is_active, timeout_secs, max_concurrent, on_failure, retry_count, workflow_spec, created_by, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		j.ID, j.ProjectID, j.Name, j.Description, j.IsActive,
		j.TimeoutSecs, j.MaxConcurrent, j.OnFailure, j.RetryCount,
		specJSON, j.CreatedBy, j.CreatedAt, j.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert job: %w", err)
	}

	if err := materializeSteps(ctx, tx, j); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *pgJobRepository) FindByID(ctx context.Context, id uuid.UUID) (*Job, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+jobColumns+` FROM jobs WHERE id = $1 LIMIT 1`,
		id,
	)
	return scanJob(row)
}

func (r *pgJobRepository) Update(ctx context.Context, j *Job) error {
	j.UpdatedAt = time.Now().UTC()
	specJSON, err := json.Marshal(j.WorkflowSpec)
	if err != nil {
		return fmt.Errorf("marshal workflow_spec: %w", err)
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	_, err = tx.Exec(ctx,
		`UPDATE jobs SET name=$1, description=$2, is_active=$3, timeout_secs=$4, max_concurrent=$5,
		  on_failure=$6, retry_count=$7, workflow_spec=$8, updated_at=$9 WHERE id=$10`,
		j.Name, j.Description, j.IsActive, j.TimeoutSecs, j.MaxConcurrent,
		j.OnFailure, j.RetryCount, specJSON, j.UpdatedAt, j.ID,
	)
	if err != nil {
		return fmt.Errorf("update job: %w", err)
	}

	// Re-materialize steps from updated spec
	if _, err := tx.Exec(ctx, `DELETE FROM job_steps WHERE job_id = $1`, j.ID); err != nil {
		return fmt.Errorf("delete old job steps: %w", err)
	}
	if err := materializeSteps(ctx, tx, j); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *pgJobRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM jobs WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete job: %w", err)
	}
	return nil
}

func (r *pgJobRepository) ListSteps(ctx context.Context, jobID uuid.UUID) ([]*JobStep, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, job_id, node_def_id, step_order, depends_on, overrides, label
		 FROM job_steps WHERE job_id = $1 ORDER BY step_order`,
		jobID,
	)
	if err != nil {
		return nil, fmt.Errorf("query job_steps: %w", err)
	}
	defer rows.Close()

	var result []*JobStep
	for rows.Next() {
		s := &JobStep{}
		var overridesJSON []byte
		var dependsOn []uuid.UUID
		if err := rows.Scan(
			&s.ID, &s.JobID, &s.NodeDefID, &s.StepOrder,
			&dependsOn, &overridesJSON, &s.Label,
		); err != nil {
			return nil, fmt.Errorf("scan job_step row: %w", err)
		}
		s.DependsOn = dependsOn
		if len(overridesJSON) > 0 {
			if err := json.Unmarshal(overridesJSON, &s.Overrides); err != nil {
				return nil, fmt.Errorf("unmarshal overrides: %w", err)
			}
		}
		result = append(result, s)
	}
	return result, rows.Err()
}

// txQuerier is a minimal interface satisfied by both pgx.Tx and pgxpool.Pool.
type txQuerier interface {
	Exec(ctx context.Context, sql string, args ...interface{}) (interface{ RowsAffected() int64 }, error)
}

// materializeSteps derives ordered JobSteps from the WorkflowSpec and inserts them.
// Steps are ordered by WorkflowNode.Position["y"] ascending.
// DependsOn is populated from WorkflowEdge.Source mapped to the node's UUID.
func materializeSteps(ctx context.Context, tx pgx.Tx, j *Job) error {
	spec := j.WorkflowSpec

	// Build a map from node string ID → NodeDefID (from Data["node_def_id"])
	// and sort nodes by position Y.
	type nodeEntry struct {
		stringID  string
		nodeDefID uuid.UUID
		posY      float64
		label     string
		overrides map[string]interface{}
	}

	entries := make([]nodeEntry, 0, len(spec.Nodes))
	for _, n := range spec.Nodes {
		var nodeDefID uuid.UUID
		if v, ok := n.Data["node_def_id"]; ok {
			switch s := v.(type) {
			case string:
				nodeDefID, _ = uuid.Parse(s)
			}
		}
		label, _ := n.Data["label"].(string)
		overrides, _ := n.Data["overrides"].(map[string]interface{})
		posY := n.Position["y"]
		entries = append(entries, nodeEntry{
			stringID:  n.ID,
			nodeDefID: nodeDefID,
			posY:      posY,
			label:     label,
			overrides: overrides,
		})
	}

	// Sort by Y position ascending (top-to-bottom order).
	sort.Slice(entries, func(i, k int) bool {
		return entries[i].posY < entries[k].posY
	})

	// Assign step order and build stringID → step map
	stepMap := make(map[string]int, len(entries)) // stringID → step index
	stepIDs := make([]uuid.UUID, len(entries))
	for i := range entries {
		stepIDs[i] = uuid.New()
		stepMap[entries[i].stringID] = i
	}

	// Build depends_on per node from edges (source → target).
	// For each node, collect all source nodes that point to it.
	dependsOnMap := make(map[string][]uuid.UUID, len(entries))
	for _, e := range spec.Edges {
		targetIdx, ok := stepMap[e.Target]
		if !ok {
			continue
		}
		sourceIdx, ok := stepMap[e.Source]
		if !ok {
			continue
		}
		_ = targetIdx
		// The target depends on the source.
		dependsOnMap[e.Target] = append(dependsOnMap[e.Target], stepIDs[sourceIdx])
	}

	for i, entry := range entries {
		overridesJSON, err := json.Marshal(entry.overrides)
		if err != nil {
			return fmt.Errorf("marshal overrides: %w", err)
		}

		dependsOn := dependsOnMap[entry.stringID]
		if dependsOn == nil {
			dependsOn = []uuid.UUID{}
		}

		_, err = tx.Exec(ctx,
			`INSERT INTO job_steps (id, job_id, node_def_id, step_order, depends_on, overrides, label)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			stepIDs[i], j.ID, entry.nodeDefID, i+1, dependsOn, overridesJSON, entry.label,
		)
		if err != nil {
			return fmt.Errorf("insert job_step: %w", err)
		}
	}

	return nil
}
