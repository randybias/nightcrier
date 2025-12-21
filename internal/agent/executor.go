package agent

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
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
	AdditionalPrompt string // Optional additional context for the agent
	Debug            bool   // Enable debug output in run-agent.sh
	Verbose          bool   // Enable verbose agent output (shows thinking/tool usage)
	Kubeconfig       string // Path to kubeconfig file for cluster access
}

// Executor runs the agent script in a workspace directory.
type Executor struct {
	config ExecutorConfig
	tuning *config.TuningConfig
}

// LogPaths contains the paths to captured agent log files
type LogPaths struct {
	Stdout   string // Path to stdout log file
	Stderr   string // Path to stderr log file
	Combined string // Path to combined log file with timestamps
}

// LogCapture manages capturing agent stdout/stderr to log files
type LogCapture struct {
	stdoutFile   *os.File
	stderrFile   *os.File
	combinedFile *os.File
	logPaths     LogPaths
	mu           sync.Mutex // Protects writes to combined log
}

// NewLogCapture creates a new LogCapture instance and sets up log files.
// It creates the logs directory in the workspace and opens the log files for writing.
// If debug is false, returns nil (no logging in production mode).
// The caller is responsible for calling Close() to clean up resources.
func NewLogCapture(workspacePath string, debug bool) (*LogCapture, error) {
	if !debug {
		return nil, nil
	}

	logsDir := filepath.Join(workspacePath, "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create logs directory: %w", err)
	}

	lc := &LogCapture{
		logPaths: LogPaths{
			Stdout:   filepath.Join(logsDir, "agent-stdout.log"),
			Stderr:   filepath.Join(logsDir, "agent-stderr.log"),
			Combined: filepath.Join(logsDir, "agent-full.log"),
		},
	}

	// Open stdout log file
	stdoutFile, err := os.Create(lc.logPaths.Stdout)
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout log file: %w", err)
	}
	lc.stdoutFile = stdoutFile

	// Open stderr log file
	stderrFile, err := os.Create(lc.logPaths.Stderr)
	if err != nil {
		stdoutFile.Close()
		return nil, fmt.Errorf("failed to create stderr log file: %w", err)
	}
	lc.stderrFile = stderrFile

	// Open combined log file
	combinedFile, err := os.Create(lc.logPaths.Combined)
	if err != nil {
		stdoutFile.Close()
		stderrFile.Close()
		return nil, fmt.Errorf("failed to create combined log file: %w", err)
	}
	lc.combinedFile = combinedFile

	return lc, nil
}

// Close closes all log files. It should be called with defer after creating LogCapture.
func (lc *LogCapture) Close() error {
	var errs []error

	if lc.stdoutFile != nil {
		if err := lc.stdoutFile.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close stdout log: %w", err))
		}
	}

	if lc.stderrFile != nil {
		if err := lc.stderrFile.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close stderr log: %w", err))
		}
	}

	if lc.combinedFile != nil {
		if err := lc.combinedFile.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close combined log: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing log files: %v", errs)
	}

	return nil
}

// GetLogPaths returns the paths to the log files
func (lc *LogCapture) GetLogPaths() LogPaths {
	return lc.logPaths
}

// writeToStdout writes data to stdout log and combined log with STDOUT prefix
func (lc *LogCapture) writeToStdout(data []byte) error {
	if _, err := lc.stdoutFile.Write(data); err != nil {
		return err
	}
	return lc.writeToCombined("STDOUT", data)
}

// writeToStderr writes data to stderr log and combined log with STDERR prefix
func (lc *LogCapture) writeToStderr(data []byte) error {
	if _, err := lc.stderrFile.Write(data); err != nil {
		return err
	}
	return lc.writeToCombined("STDERR", data)
}

// writeToCombined writes data to combined log with timestamp and stream indicator
func (lc *LogCapture) writeToCombined(stream string, data []byte) error {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	timestamp := time.Now().Format(time.RFC3339)
	scanner := bufio.NewScanner(bufio.NewReader(bytes.NewReader(data)))
	for scanner.Scan() {
		line := scanner.Text()
		if _, err := fmt.Fprintf(lc.combinedFile, "%s [%s] %s\n", timestamp, stream, line); err != nil {
			return err
		}
	}
	return scanner.Err()
}

// logWriter is an io.Writer adapter that writes to LogCapture
type logWriter struct {
	logCapture *LogCapture
	isStderr   bool
}

// Write implements io.Writer interface for use with io.TeeReader
func (lw *logWriter) Write(p []byte) (n int, err error) {
	if lw.isStderr {
		if err := lw.logCapture.writeToStderr(p); err != nil {
			return 0, err
		}
	} else {
		if err := lw.logCapture.writeToStdout(p); err != nil {
			return 0, err
		}
	}
	return len(p), nil
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
// It returns the exit code, log file paths, and any error encountered.
func (e *Executor) Execute(ctx context.Context, workspacePath string, incidentID string) (int, LogPaths, error) {
	// Use the configured additional prompt (may be empty)
	return e.ExecuteWithPrompt(ctx, workspacePath, incidentID, e.config.AdditionalPrompt)
}

// ExecuteWithPrompt runs the agent with a custom prompt
func (e *Executor) ExecuteWithPrompt(ctx context.Context, workspacePath string, incidentID string, prompt string) (int, LogPaths, error) {
	slog.Info("executing agent",
		"script", e.config.ScriptPath,
		"workspace", workspacePath,
		"incident_id", incidentID,
		"agent_cli", e.config.AgentCLI,
		"model", e.config.Model,
		"timeout", e.config.Timeout)

	// Capture the combined prompt to prompt-sent.md before execution
	if err := e.capturePrompt(workspacePath, incidentID, prompt); err != nil {
		slog.Warn("failed to capture prompt for audit", "error", err)
		// Continue execution - prompt capture failure is not fatal
	}

	// Create log capture to persist agent output to files (DEBUG mode only)
	logCapture, err := NewLogCapture(workspacePath, e.config.Debug)
	if err != nil {
		return -1, LogPaths{}, fmt.Errorf("failed to create log capture: %w", err)
	}
	defer func() {
		if logCapture != nil {
			logCapture.Close()
		}
	}()

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

	// Add kubeconfig if specified (Phase 2: multi-cluster support)
	if e.config.Kubeconfig != "" {
		args = append(args, "--kubeconfig", e.config.Kubeconfig)
	}

	// Build the combined prompt: system prompt content + additional prompt (if set)
	// The system prompt drives the investigation; additional prompt provides optional context
	systemPromptContent, err := e.readSystemPromptFile()
	if err != nil {
		return -1, LogPaths{}, fmt.Errorf("failed to read system prompt file: %w", err)
	}

	// Combine prompts: system prompt is primary, additional prompt appended if present
	combinedPrompt := systemPromptContent
	if prompt != "" {
		if combinedPrompt != "" {
			combinedPrompt += "\n\n" + prompt
		} else {
			combinedPrompt = prompt
		}
	}

	if combinedPrompt == "" {
		return -1, LogPaths{}, fmt.Errorf("no prompt available: system prompt file is empty and no additional prompt provided")
	}

	args = append(args, combinedPrompt)

	// Create context with timeout using configured buffer from TuningConfig
	timeoutWithBuffer := e.config.Timeout + e.tuning.Agent.TimeoutBufferSeconds
	execCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutWithBuffer)*time.Second)
	defer cancel()

	// Build bash command - add -x flag in debug mode to trace command execution
	bashArgs := []string{e.config.ScriptPath}
	if e.config.Debug {
		bashArgs = []string{"-x", e.config.ScriptPath}
	}
	cmd := exec.CommandContext(execCtx, "bash", append(bashArgs, args...)...)

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

	// Enable debug output in run-agent.sh when running in debug mode
	if e.config.Debug {
		cmd.Env = append(cmd.Env, "DEBUG=true")
	}

	// Enable verbose agent output (shows thinking and tool usage)
	if e.config.Verbose {
		cmd.Env = append(cmd.Env, "AGENT_VERBOSE=true")
	}

	// Add kubeconfig path for cluster access (Phase 2: multi-cluster support)
	if e.config.Kubeconfig != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("KUBECONFIG=%s", e.config.Kubeconfig))
	}

	// Capture stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return -1, LogPaths{}, fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return -1, LogPaths{}, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return -1, LogPaths{}, fmt.Errorf("failed to start script: %w", err)
	}

	// Use TeeReader to capture output to log files while still reading for slog
	// This allows both file persistence and real-time visibility
	// If logCapture is nil (non-DEBUG mode), TeeReader writes go to io.Discard
	var stdoutDest, stderrDest io.Writer
	if logCapture != nil {
		stdoutDest = &logWriter{logCapture: logCapture, isStderr: false}
		stderrDest = &logWriter{logCapture: logCapture, isStderr: true}
	} else {
		stdoutDest = io.Discard
		stderrDest = io.Discard
	}
	stdoutTee := io.TeeReader(stdout, stdoutDest)
	stderrTee := io.TeeReader(stderr, stderrDest)

	// Log output as it comes in using configured buffer sizes from TuningConfig
	// The slog output provides real-time visibility while TeeReader writes to files
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		buf := make([]byte, e.tuning.IO.StdoutBufferSize)
		for {
			n, err := stdoutTee.Read(buf)
			if n > 0 {
				slog.Info("agent stdout", "output", string(buf[:n]))
			}
			if err != nil {
				break
			}
		}
	}()

	go func() {
		defer wg.Done()
		buf := make([]byte, e.tuning.IO.StderrBufferSize)
		for {
			n, err := stderrTee.Read(buf)
			if n > 0 {
				slog.Warn("agent stderr", "output", string(buf[:n]))
			}
			if err != nil {
				break
			}
		}
	}()

	// Wait for output goroutines to finish reading
	wg.Wait()

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
			return -1, LogPaths{}, fmt.Errorf("failed to wait for script: %w", err)
		}
	}

	slog.Info("agent script completed", "exit_code", exitCode)
	if logCapture != nil {
		return exitCode, logCapture.GetLogPaths(), nil
	}
	return exitCode, LogPaths{}, nil
}

// capturePrompt writes the combined system + additional prompt to prompt-sent.md
// for auditability and debugging. This is called before subprocess launch.
func (e *Executor) capturePrompt(workspacePath string, incidentID string, additionalPrompt string) error {
	// Read system prompt file content
	systemPromptContent, err := e.readSystemPromptFile()
	if err != nil {
		return fmt.Errorf("failed to read system prompt: %w", err)
	}

	// Generate the prompt-sent.md content
	content := e.generatePromptSentContent(incidentID, systemPromptContent, additionalPrompt)

	// Write to workspace
	promptPath := filepath.Join(workspacePath, "prompt-sent.md")
	if err := os.WriteFile(promptPath, []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to write prompt-sent.md: %w", err)
	}

	slog.Debug("captured prompt to prompt-sent.md", "path", promptPath)
	return nil
}

// readSystemPromptFile reads the system prompt file content.
// Returns empty string if no system prompt file is configured.
func (e *Executor) readSystemPromptFile() (string, error) {
	if e.config.SystemPromptFile == "" {
		return "", nil
	}

	content, err := os.ReadFile(e.config.SystemPromptFile)
	if err != nil {
		return "", fmt.Errorf("failed to read system prompt file %s: %w", e.config.SystemPromptFile, err)
	}

	return string(content), nil
}

// generatePromptSentContent creates the markdown content for prompt-sent.md
func (e *Executor) generatePromptSentContent(incidentID string, systemPrompt string, additionalPrompt string) string {
	timestamp := time.Now().UTC().Format(time.RFC3339)

	// Extract cluster name from kubeconfig path if available
	clusterName := "unknown"
	if e.config.Kubeconfig != "" {
		// Try to extract cluster name from kubeconfig filename
		// e.g., "./kubeconfigs/prod-us-east-1-readonly.yaml" -> "prod-us-east-1"
		base := filepath.Base(e.config.Kubeconfig)
		// Remove extension and -readonly suffix
		clusterName = base
		if ext := filepath.Ext(base); ext != "" {
			clusterName = base[:len(base)-len(ext)]
		}
		// Remove common suffixes
		for _, suffix := range []string{"-readonly", "-admin", "-kubeconfig"} {
			if len(clusterName) > len(suffix) && clusterName[len(clusterName)-len(suffix):] == suffix {
				clusterName = clusterName[:len(clusterName)-len(suffix)]
			}
		}
	}

	// Build the prompt-sent.md content
	var content string
	content = "# Prompt Sent to Agent\n\n"
	content += "## Metadata\n"
	content += fmt.Sprintf("- Timestamp: %s\n", timestamp)
	content += fmt.Sprintf("- Incident ID: %s\n", incidentID)
	content += fmt.Sprintf("- Cluster: %s\n", clusterName)
	content += fmt.Sprintf("- Agent CLI: %s\n", e.config.AgentCLI)
	content += fmt.Sprintf("- Model: %s\n", e.config.Model)
	content += "\n"

	content += "## System Prompt\n\n"
	if systemPrompt != "" {
		content += systemPrompt
		if systemPrompt[len(systemPrompt)-1] != '\n' {
			content += "\n"
		}
	} else {
		content += "*No system prompt configured*\n"
	}
	content += "\n"

	content += "## Additional Prompt\n\n"
	if additionalPrompt != "" {
		content += additionalPrompt
		if additionalPrompt[len(additionalPrompt)-1] != '\n' {
			content += "\n"
		}
	} else {
		content += "*None provided*\n"
	}

	return content
}
