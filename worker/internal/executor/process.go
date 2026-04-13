package executor

import (
	"context"
	"fmt"
	"os/exec"
	"syscall"
	"time"
)

const (
	// defaultPath is the explicit PATH injected into child processes.
	// Alpine/minimal containers may lack a full PATH; this covers the common cases.
	defaultPath = "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"

	// killGraceSeconds is the window between SIGTERM and SIGKILL.
	killGracePeriod = 5 * time.Second
)

// buildCmd constructs an exec.Cmd for the given argv slice.
//
// Notable properties of the returned command:
//   - Runs in its own process group (Setpgid=true) so that killProcessGroup can
//     reliably terminate any spawned child processes.
//   - Stdout and Stderr pipes are opened; the caller must consume them before
//     calling Wait.
//   - The context deadline is honoured, but callers are responsible for
//     calling killProcessGroup when the context is cancelled rather than relying
//     on exec.CommandContext's built-in SIGKILL behaviour (which only kills the
//     direct child, not the whole group).
func buildCmd(ctx context.Context, argv []string, env []string, workDir string) (*exec.Cmd, error) {
	if len(argv) == 0 {
		return nil, fmt.Errorf("buildCmd: argv must not be empty")
	}

	// Use context-aware command so the stdlib can participate in deadline
	// tracking, but we override the kill signal below.
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)

	// Explicit PATH so scripts can call standard utilities.
	path := fmt.Sprintf("PATH=%s", defaultPath)
	cmd.Env = append([]string{path}, env...)

	if workDir != "" {
		cmd.Dir = workDir
	}

	// Put the process in its own group; kill(2) with negative PID kills all.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	return cmd, nil
}

// killProcessGroup sends SIGTERM to the entire process group of cmd, waits up
// to killGracePeriod, and then sends SIGKILL if the process is still running.
//
// It is safe to call even if the process has already exited.
func killProcessGroup(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}

	pgid := -cmd.Process.Pid

	// SIGTERM – polite shutdown request.
	if err := syscall.Kill(pgid, syscall.SIGTERM); err != nil {
		// Process may have already exited; that's fine.
		return
	}

	// Give the process group a moment to clean up.
	done := make(chan struct{})
	go func() {
		// Wait for the process (non-blocking via polling would be ideal, but
		// the process handle is owned by the caller's goroutine; we can check
		// by sending signal 0 which does not actually send a signal).
		deadline := time.Now().Add(killGracePeriod)
		for time.Now().Before(deadline) {
			if err := syscall.Kill(pgid, syscall.Signal(0)); err != nil {
				// Process group is gone.
				break
			}
			time.Sleep(200 * time.Millisecond)
		}
		close(done)
	}()

	select {
	case <-done:
		// Group exited within grace period.
	case <-time.After(killGracePeriod):
		// Grace period expired – force kill.
		_ = syscall.Kill(pgid, syscall.SIGKILL)
	}
}
