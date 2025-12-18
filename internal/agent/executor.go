package agent

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
)

// Executor runs the agent script in a workspace directory.
type Executor struct {
	scriptPath string
}

// NewExecutor creates a new Executor with the given script path.
func NewExecutor(scriptPath string) *Executor {
	return &Executor{
		scriptPath: scriptPath,
	}
}

// Execute runs the agent script with the given incident ID in the workspace directory.
// It returns the exit code and any error encountered.
func (e *Executor) Execute(ctx context.Context, workspacePath string, incidentID string) (int, error) {
	slog.Info("executing agent script",
		"script", e.scriptPath,
		"workspace", workspacePath,
		"incident_id", incidentID)

	cmd := exec.CommandContext(ctx, e.scriptPath)
	cmd.Dir = workspacePath
	cmd.Env = append(os.Environ(), fmt.Sprintf("INCIDENT_ID=%s", incidentID))

	// Capture stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return -1, fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return -1, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return -1, fmt.Errorf("failed to start script: %w", err)
	}

	// Log output as it comes in
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := stdout.Read(buf)
			if n > 0 {
				slog.Info("agent stdout", "output", string(buf[:n]))
			}
			if err != nil {
				break
			}
		}
	}()

	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := stderr.Read(buf)
			if n > 0 {
				slog.Warn("agent stderr", "output", string(buf[:n]))
			}
			if err != nil {
				break
			}
		}
	}()

	// Wait for the command to complete
	err = cmd.Wait()

	// Get exit code
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
			slog.Info("agent script exited with non-zero code",
				"exit_code", exitCode,
				"error", err)
		} else {
			return -1, fmt.Errorf("failed to wait for script: %w", err)
		}
	}

	slog.Info("agent script completed", "exit_code", exitCode)
	return exitCode, nil
}
