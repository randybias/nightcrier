package agent

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/rbias/nightcrier/internal/config"
)

// ExecutorConfig holds configuration for the agent executor
type ExecutorConfig struct {
	ScriptPath       string
	SystemPromptFile string
	AllowedTools     string
	Model            string
	Timeout          int    // seconds
	AgentCLI         string // claude, codex, goose, gemini
	AgentImage       string // Docker image for agent container
	Prompt           string // Prompt to send to the agent
}

// Executor runs the agent script in a workspace directory.
type Executor struct {
	config ExecutorConfig
	tuning *config.TuningConfig
}

// NewExecutorWithConfig creates an Executor with full configuration.
// All configuration values must be provided explicitly - no defaults are applied.
// The tuning parameter provides runtime tuning parameters like timeout buffers and I/O buffer sizes.
func NewExecutorWithConfig(config ExecutorConfig, tuning *config.TuningConfig) *Executor {
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

	return &Executor{
		config: config,
		tuning: tuning,
	}
}

// Execute runs the agent script with the given incident ID in the workspace directory.
// It returns the exit code and any error encountered.
func (e *Executor) Execute(ctx context.Context, workspacePath string, incidentID string) (int, error) {
	// Use the configured prompt
	return e.ExecuteWithPrompt(ctx, workspacePath, incidentID, e.config.Prompt)
}

// ExecuteWithPrompt runs the agent with a custom prompt
func (e *Executor) ExecuteWithPrompt(ctx context.Context, workspacePath string, incidentID string, prompt string) (int, error) {
	slog.Info("executing agent",
		"script", e.config.ScriptPath,
		"workspace", workspacePath,
		"incident_id", incidentID,
		"agent_cli", e.config.AgentCLI,
		"model", e.config.Model,
		"timeout", e.config.Timeout)

	// Build command args for run-agent.sh
	args := []string{
		"--workspace", workspacePath,
		"--model", e.config.Model,
		"--allowed-tools", e.config.AllowedTools,
		"--timeout", fmt.Sprintf("%d", e.config.Timeout),
	}

	// Add agent CLI selection if specified
	if e.config.AgentCLI != "" {
		args = append(args, "--agent", e.config.AgentCLI)
	}

	if e.config.SystemPromptFile != "" {
		args = append(args, "--system-prompt-file", e.config.SystemPromptFile)
	}

	// Add the prompt as the final argument
	args = append(args, prompt)

	// Create context with timeout using configured buffer from TuningConfig
	timeoutWithBuffer := e.config.Timeout + e.tuning.Agent.TimeoutBufferSeconds
	execCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutWithBuffer)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(execCtx, "bash", append([]string{e.config.ScriptPath}, args...)...)

	// Set all configuration as environment variables for the script using generic agent-agnostic names
	// This eliminates the need for hardcoded defaults in the script
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("INCIDENT_ID=%s", incidentID),
		fmt.Sprintf("AGENT_CLI=%s", e.config.AgentCLI),
		fmt.Sprintf("AGENT_IMAGE=%s", e.config.AgentImage),
		fmt.Sprintf("LLM_MODEL=%s", e.config.Model),
		fmt.Sprintf("AGENT_ALLOWED_TOOLS=%s", e.config.AllowedTools),
		fmt.Sprintf("CONTAINER_TIMEOUT=%d", e.config.Timeout),
		fmt.Sprintf("OUTPUT_FORMAT=%s", "text"),
		fmt.Sprintf("CONTAINER_NETWORK=%s", "host"),
	)

	// Add optional config values if set
	if e.config.SystemPromptFile != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("SYSTEM_PROMPT_FILE=%s", e.config.SystemPromptFile))
	}

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

	// Log output as it comes in using configured buffer sizes from TuningConfig
	go func() {
		buf := make([]byte, e.tuning.IO.StdoutBufferSize)
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
		buf := make([]byte, e.tuning.IO.StderrBufferSize)
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
