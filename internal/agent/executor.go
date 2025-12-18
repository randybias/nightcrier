package agent

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// ExecutorConfig holds configuration for the agent executor
type ExecutorConfig struct {
	ScriptPath       string
	SystemPromptFile string
	AllowedTools     string
	Model            string
	Timeout          int // seconds
}

// Executor runs the agent script in a workspace directory.
type Executor struct {
	config ExecutorConfig
}

// NewExecutor creates a new Executor with the given script path.
// The script path is converted to an absolute path to ensure it works
// regardless of the working directory when Execute is called.
func NewExecutor(scriptPath string) *Executor {
	return NewExecutorWithConfig(ExecutorConfig{
		ScriptPath:   scriptPath,
		AllowedTools: "Read,Write,Grep,Glob,Bash,Skill",
		Model:        "sonnet",
		Timeout:      300,
	})
}

// NewExecutorWithConfig creates an Executor with full configuration
func NewExecutorWithConfig(config ExecutorConfig) *Executor {
	absPath, err := filepath.Abs(config.ScriptPath)
	if err != nil {
		slog.Warn("failed to get absolute path for script, using as-is",
			"script", config.ScriptPath,
			"error", err)
		absPath = config.ScriptPath
	}
	config.ScriptPath = absPath

	if config.SystemPromptFile != "" {
		absPrompt, err := filepath.Abs(config.SystemPromptFile)
		if err == nil {
			config.SystemPromptFile = absPrompt
		}
	}

	return &Executor{config: config}
}

// Execute runs the agent script with the given incident ID in the workspace directory.
// It returns the exit code and any error encountered.
func (e *Executor) Execute(ctx context.Context, workspacePath string, incidentID string) (int, error) {
	// Build the prompt
	prompt := "Production incident detected. Fault event details are in event.json. " +
		"Perform immediate triage and root cause analysis. " +
		"Write findings to output/investigation.md"

	return e.ExecuteWithPrompt(ctx, workspacePath, incidentID, prompt)
}

// ExecuteWithPrompt runs the agent with a custom prompt
func (e *Executor) ExecuteWithPrompt(ctx context.Context, workspacePath string, incidentID string, prompt string) (int, error) {
	slog.Info("executing agent",
		"script", e.config.ScriptPath,
		"workspace", workspacePath,
		"incident_id", incidentID,
		"model", e.config.Model,
		"timeout", e.config.Timeout)

	// Build command args for run-agent.sh
	args := []string{
		"--workspace", workspacePath,
		"--model", e.config.Model,
		"--allowed-tools", e.config.AllowedTools,
		"--timeout", fmt.Sprintf("%d", e.config.Timeout),
	}

	if e.config.SystemPromptFile != "" {
		args = append(args, "--system-prompt-file", e.config.SystemPromptFile)
	}

	// Add the prompt as the final argument
	args = append(args, prompt)

	// Create context with timeout
	execCtx, cancel := context.WithTimeout(ctx, time.Duration(e.config.Timeout+60)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(execCtx, "bash", append([]string{e.config.ScriptPath}, args...)...)
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
