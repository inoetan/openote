package executor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/inoetan/openote/worker/internal/logstream"
)

// Executor loads job definitions from PostgreSQL, runs their steps as OS
// processes, streams log output via logstream.Writer, and records final
// state back to PostgreSQL.
type Executor struct {
	db    *pgxpool.Pool
	redis *redis.Client
}

// New returns a ready-to-use Executor.
func New(db *pgxpool.Pool, rdb *redis.Client) *Executor {
	return &Executor{db: db, redis: rdb}
}

// ExecutionRequest carries the identifiers extracted from a Redis Stream message.
type ExecutionRequest struct {
	ExecutionID uuid.UUID
	JobID       uuid.UUID
	ProjectID   uuid.UUID
	WorkerID    string
}

// jobRow holds the columns we need from the jobs table.
type jobRow struct {
	id          uuid.UUID
	onFailure   string
	timeoutSecs int
}

// stepRow holds the columns we need from job_steps joined with node_definitions.
type stepRow struct {
	stepID      uuid.UUID // job_steps.id
	nodeDefID   uuid.UUID
	stepOrder   int
	nodeType    string                 // node_definitions.type
	nodeConfig  map[string]interface{} // node_definitions.config
	overrides   map[string]interface{} // job_steps.overrides (merged on top of nodeConfig)
	label       string
}

// stepExecID is the UUID of the step_executions row.
type stepExecID = uuid.UUID

// Execute is the main entry point called by the consumer for each job dispatch.
//
// Flow:
//  1. Load job + steps from DB.
//  2. Mark execution as "running".
//  3. For each step in step_order:
//     a. Insert / update step_execution to "running".
//     b. Run the step process.
//     c. Record outcome.
//     d. Abort remaining steps if job.on_failure == "stop" and step failed.
//  4. Mark execution as "success" or "failed" with timestamps.
func (e *Executor) Execute(ctx context.Context, req ExecutionRequest) error {
	// --- 1. Load job ---
	job, err := e.loadJob(ctx, req.JobID)
	if err != nil {
		return fmt.Errorf("load job %s: %w", req.JobID, err)
	}
	if job == nil {
		return fmt.Errorf("job %s not found", req.JobID)
	}

	steps, err := e.loadSteps(ctx, req.JobID)
	if err != nil {
		return fmt.Errorf("load steps for job %s: %w", req.JobID, err)
	}

	// --- 2. Mark execution as running ---
	startedAt := time.Now().UTC()
	if err := e.updateExecution(ctx, req.ExecutionID, map[string]interface{}{
		"status":     "running",
		"started_at": startedAt,
		"worker_id":  req.WorkerID,
	}); err != nil {
		return fmt.Errorf("mark execution running: %w", err)
	}

	// shared sequence counter for all log lines within this execution
	var seq atomic.Int64

	finalStatus := "success"
	var firstStepErr error

	// --- 3. Execute each step in order ---
	for _, step := range steps {
		seID, err := e.upsertStepExecution(ctx, req.ExecutionID, step.stepID, "running")
		if err != nil {
			log.Printf("executor: upsert step_execution for step %s: %v", step.stepID, err)
			finalStatus = "failed"
			break
		}

		// Merge node config with per-step overrides (overrides win).
		mergedConfig := mergeConfig(step.nodeConfig, step.overrides)

		exitCode, stepErr := e.runStep(ctx, seID, req.ExecutionID, step.nodeType, mergedConfig, &seq)

		finishedAt := time.Now().UTC()
		stepStatus := "success"
		if stepErr != nil || exitCode != 0 {
			stepStatus = "failed"
			if firstStepErr == nil {
				firstStepErr = stepErr
			}
		}

		if err := e.finalizeStepExecution(ctx, seID, stepStatus, exitCode, finishedAt); err != nil {
			log.Printf("executor: finalize step_execution %s: %v", seID, err)
		}

		if stepStatus == "failed" {
			finalStatus = "failed"
			if job.onFailure == "stop" {
				// Skip remaining steps.
				for _, remaining := range steps {
					if remaining.stepOrder > step.stepOrder {
						skippedID, err := e.upsertStepExecution(ctx, req.ExecutionID, remaining.stepID, "skipped")
						if err != nil {
							log.Printf("executor: upsert skipped step_execution for step %s: %v", remaining.stepID, err)
							continue
						}
						_ = e.finalizeStepExecution(ctx, skippedID, "skipped", -1, finishedAt)
					}
				}
				break
			}
		}
	}

	// --- 4. Finalize execution ---
	finishedAt := time.Now().UTC()
	durationMs := finishedAt.Sub(startedAt).Milliseconds()
	if err := e.updateExecution(ctx, req.ExecutionID, map[string]interface{}{
		"status":      finalStatus,
		"finished_at": finishedAt,
		"duration_ms": durationMs,
	}); err != nil {
		log.Printf("executor: update final execution status: %v", err)
	}

	if firstStepErr != nil {
		return fmt.Errorf("execution %s failed: %w", req.ExecutionID, firstStepErr)
	}
	return nil
}

// runStep builds and runs the OS command for a single step, streaming its
// output via logstream.Writer.  It returns the process exit code and any
// non-exit-code error (e.g. a setup failure).
func (e *Executor) runStep(
	ctx context.Context,
	seID stepExecID,
	executionID uuid.UUID,
	nodeType string,
	config map[string]interface{},
	seq *atomic.Int64,
) (exitCode int, err error) {
	// Build per-step context with optional timeout.
	timeoutSecs := configInt(config, "timeout_secs", 3600)
	stepCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSecs)*time.Second)
	defer cancel()

	argv, tempFile, buildErr := buildArgv(nodeType, config)
	if buildErr != nil {
		return -1, fmt.Errorf("build argv for %s: %w", nodeType, buildErr)
	}
	if tempFile != "" {
		defer func() { _ = os.Remove(tempFile) }()
	}

	envVars := buildEnvVars(config)
	workDir, _ := config["working_dir"].(string)

	cmd, err := buildCmd(stepCtx, argv, envVars, workDir)
	if err != nil {
		return -1, fmt.Errorf("buildCmd: %w", err)
	}

	// Attach stdout/stderr pipes.
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return -1, fmt.Errorf("stdout pipe: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return -1, fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return -1, fmt.Errorf("start process: %w", err)
	}

	// Stream stdout and stderr concurrently.
	stdoutWriter := logstream.New(e.db, e.redis, executionID, seID, "stdout", seq)
	stderrWriter := logstream.New(e.db, e.redis, executionID, seID, "stderr", seq)

	streamErrCh := make(chan error, 2)
	go func() {
		streamErrCh <- e.copyStream(stdoutWriter, stdoutPipe)
	}()
	go func() {
		streamErrCh <- e.copyStream(stderrWriter, stderrPipe)
	}()

	// Wait for streams to drain.
	for i := 0; i < 2; i++ {
		if sErr := <-streamErrCh; sErr != nil {
			log.Printf("executor: stream copy error: %v", sErr)
		}
	}

	// Flush any partial last line (no trailing newline).
	stdoutWriter.Flush()
	stderrWriter.Flush()

	// Ensure process has exited.
	waitErr := cmd.Wait()

	// If context was cancelled, kill the whole process group.
	if stepCtx.Err() != nil {
		killProcessGroup(cmd)
	}

	if waitErr != nil {
		// Extract exit code from *exec.ExitError (non-zero exit).
		var exitErr *exec.ExitError
		if errors.As(waitErr, &exitErr) {
			return exitErr.ExitCode(), nil // non-zero exit recorded as step failure
		}
		// Context cancelled / timeout counts as failure.
		if stepCtx.Err() != nil {
			return -1, fmt.Errorf("step timed out or cancelled: %w", stepCtx.Err())
		}
		return -1, fmt.Errorf("wait: %w", waitErr)
	}

	return 0, nil
}

// copyStream copies reader into writer, satisfying the io.Writer contract.
func (e *Executor) copyStream(w io.Writer, r io.Reader) error {
	_, err := io.Copy(w, r)
	return err
}

// ---------------------------------------------------------------------------
// Argument construction helpers
// ---------------------------------------------------------------------------

// buildArgv constructs the argv slice and, for shell_script, writes the script
// to a temp file and returns its path in tempFile (caller must remove it).
func buildArgv(nodeType string, config map[string]interface{}) (argv []string, tempFile string, err error) {
	switch nodeType {
	case "shell_script":
		content, _ := config["script_content"].(string)
		interpreter, _ := config["interpreter"].(string)
		if interpreter == "" {
			interpreter = "bash"
		}
		// Resolve interpreter to full path if it isn't already absolute.
		if !filepath.IsAbs(interpreter) {
			interpreter = findInterpreter(interpreter)
		}

		f, err := os.CreateTemp("", "openote-script-*.sh")
		if err != nil {
			return nil, "", fmt.Errorf("create temp script: %w", err)
		}
		if _, err := f.WriteString(content); err != nil {
			_ = f.Close()
			_ = os.Remove(f.Name())
			return nil, "", fmt.Errorf("write script content: %w", err)
		}
		if err := f.Chmod(0700); err != nil {
			_ = f.Close()
			_ = os.Remove(f.Name())
			return nil, "", fmt.Errorf("chmod script: %w", err)
		}
		if err := f.Close(); err != nil {
			_ = os.Remove(f.Name())
			return nil, "", fmt.Errorf("close temp script: %w", err)
		}
		return []string{interpreter, f.Name()}, f.Name(), nil

	case "shell_command":
		command, _ := config["command"].(string)
		if command == "" {
			return nil, "", fmt.Errorf("shell_command: missing 'command' in config")
		}
		parts := splitCommand(command)
		if len(parts) == 0 {
			return nil, "", fmt.Errorf("shell_command: empty command after splitting")
		}
		return parts, "", nil

	default:
		return nil, "", fmt.Errorf("unsupported node type: %q", nodeType)
	}
}

// splitCommand splits a shell command string into argv tokens. It respects
// single-quoted strings but is intentionally simple – it does NOT perform full
// POSIX word-splitting or variable expansion; that is the interpreter's job.
func splitCommand(command string) []string {
	var parts []string
	var current strings.Builder
	inQuote := false
	quoteChar := byte(0)

	for i := 0; i < len(command); i++ {
		c := command[i]
		switch {
		case inQuote:
			if c == quoteChar {
				inQuote = false
			} else {
				current.WriteByte(c)
			}
		case c == '\'' || c == '"':
			inQuote = true
			quoteChar = c
		case c == ' ' || c == '\t' || c == '\n':
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
		default:
			current.WriteByte(c)
		}
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	return parts
}

// buildEnvVars converts config["env_vars"] (expected map[string]interface{}) to
// a slice of "KEY=VALUE" strings.
func buildEnvVars(config map[string]interface{}) []string {
	raw, ok := config["env_vars"]
	if !ok {
		return nil
	}
	m, ok := raw.(map[string]interface{})
	if !ok {
		return nil
	}
	out := make([]string, 0, len(m))
	for k, v := range m {
		out = append(out, fmt.Sprintf("%s=%v", k, v))
	}
	return out
}

// findInterpreter looks up an interpreter name in the default PATH.  If not
// found it returns the name unmodified (exec will fail with a clear error).
func findInterpreter(name string) string {
	for _, dir := range strings.Split(defaultPath, ":") {
		p := filepath.Join(dir, name)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return name
}

// ---------------------------------------------------------------------------
// Database helpers
// ---------------------------------------------------------------------------

func (e *Executor) loadJob(ctx context.Context, jobID uuid.UUID) (*jobRow, error) {
	row := e.db.QueryRow(ctx,
		`SELECT id, on_failure, timeout_secs FROM jobs WHERE id = $1`,
		jobID,
	)
	j := &jobRow{}
	if err := row.Scan(&j.id, &j.onFailure, &j.timeoutSecs); err != nil {
		return nil, fmt.Errorf("scan job: %w", err)
	}
	return j, nil
}

func (e *Executor) loadSteps(ctx context.Context, jobID uuid.UUID) ([]*stepRow, error) {
	rows, err := e.db.Query(ctx,
		`SELECT js.id, js.node_def_id, js.step_order, js.overrides, js.label,
		        nd.type, nd.config
		 FROM job_steps js
		 JOIN node_definitions nd ON nd.id = js.node_def_id
		 WHERE js.job_id = $1
		 ORDER BY js.step_order`,
		jobID,
	)
	if err != nil {
		return nil, fmt.Errorf("query job_steps: %w", err)
	}
	defer rows.Close()

	var result []*stepRow
	for rows.Next() {
		s := &stepRow{}
		var overridesJSON, nodeConfigJSON []byte
		if err := rows.Scan(
			&s.stepID, &s.nodeDefID, &s.stepOrder,
			&overridesJSON, &s.label,
			&s.nodeType, &nodeConfigJSON,
		); err != nil {
			return nil, fmt.Errorf("scan step row: %w", err)
		}
		if len(overridesJSON) > 0 {
			if err := json.Unmarshal(overridesJSON, &s.overrides); err != nil {
				return nil, fmt.Errorf("unmarshal overrides: %w", err)
			}
		}
		if len(nodeConfigJSON) > 0 {
			if err := json.Unmarshal(nodeConfigJSON, &s.nodeConfig); err != nil {
				return nil, fmt.Errorf("unmarshal node config: %w", err)
			}
		}
		result = append(result, s)
	}
	return result, rows.Err()
}

// updateExecution applies an arbitrary set of column=value pairs to executions.
// Only known columns are accepted to avoid SQL injection.
func (e *Executor) updateExecution(ctx context.Context, execID uuid.UUID, fields map[string]interface{}) error {
	allowed := map[string]bool{
		"status": true, "started_at": true, "finished_at": true,
		"worker_id": true, "duration_ms": true,
	}
	setClauses := make([]string, 0, len(fields))
	args := make([]interface{}, 0, len(fields)+1)
	i := 1
	for col, val := range fields {
		if !allowed[col] {
			continue
		}
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", col, i))
		args = append(args, val)
		i++
	}
	if len(setClauses) == 0 {
		return nil
	}
	args = append(args, execID)
	query := fmt.Sprintf(
		"UPDATE executions SET %s WHERE id = $%d",
		strings.Join(setClauses, ", "), i,
	)
	_, err := e.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update executions: %w", err)
	}
	return nil
}

// upsertStepExecution ensures a step_executions row exists and sets its status.
// Returns the step_execution UUID.
func (e *Executor) upsertStepExecution(
	ctx context.Context,
	execID uuid.UUID,
	stepID uuid.UUID,
	status string,
) (uuid.UUID, error) {
	seID := uuid.New()
	now := time.Now().UTC()
	_, err := e.db.Exec(ctx,
		`INSERT INTO step_executions (id, execution_id, step_id, status, started_at)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (execution_id, step_id) DO UPDATE
		   SET status = EXCLUDED.status, started_at = EXCLUDED.started_at`,
		seID, execID, stepID, status, now,
	)
	if err != nil {
		return uuid.Nil, fmt.Errorf("upsert step_execution: %w", err)
	}
	// On conflict the existing UUID is preserved; retrieve it.
	var actualID uuid.UUID
	err = e.db.QueryRow(ctx,
		`SELECT id FROM step_executions WHERE execution_id = $1 AND step_id = $2`,
		execID, stepID,
	).Scan(&actualID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("select step_execution id: %w", err)
	}
	return actualID, nil
}

// finalizeStepExecution writes exit_code, status, and finished_at to a step_executions row.
func (e *Executor) finalizeStepExecution(
	ctx context.Context,
	seID uuid.UUID,
	status string,
	exitCode int,
	finishedAt time.Time,
) error {
	_, err := e.db.Exec(ctx,
		`UPDATE step_executions SET status = $1, exit_code = $2, finished_at = $3 WHERE id = $4`,
		status, exitCode, finishedAt, seID,
	)
	if err != nil {
		return fmt.Errorf("update step_execution: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Utility helpers
// ---------------------------------------------------------------------------

// mergeConfig returns a new map with nodeConfig as the base and overrides on top.
func mergeConfig(base, overrides map[string]interface{}) map[string]interface{} {
	merged := make(map[string]interface{}, len(base)+len(overrides))
	for k, v := range base {
		merged[k] = v
	}
	for k, v := range overrides {
		merged[k] = v
	}
	return merged
}

// configInt extracts an integer from a config map, returning defaultVal on
// missing key or type mismatch.
func configInt(config map[string]interface{}, key string, defaultVal int) int {
	v, ok := config[key]
	if !ok {
		return defaultVal
	}
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	case json.Number:
		i, err := n.Int64()
		if err == nil {
			return int(i)
		}
	}
	return defaultVal
}

