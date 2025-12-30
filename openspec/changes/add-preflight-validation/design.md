# Design: add-preflight-validation

## Architecture

### Package Structure

```
internal/preflight/
├── preflight.go         # Validator interface, Runner, standard validators
├── preflight_test.go    # Unit tests
└── README.md            # Developer guide for adding validators
```

### Core Types

```go
// Validator represents a single pre-flight check
type Validator interface {
    // Name returns a human-readable name for this validator
    Name() string

    // Validate performs the check and returns an error with remediation guidance
    Validate(ctx context.Context, cfg *config.Config) error
}

// Runner executes multiple validators in sequence
type Runner struct {
    validators []Validator
}

// Result captures validation outcomes
type Result struct {
    Success    bool
    ChecksPassed []string
    ChecksFailed []ValidationError
}

// ValidationError wraps errors with remediation context
type ValidationError struct {
    CheckName   string
    Error       error
    Remediation string  // Helpful guidance for fixing the issue
}
```

### Standard Validators

#### 1. FileExistenceValidator
Validates that required files exist and are readable.

```go
type FileExistenceValidator struct {
    name        string
    pathGetter  func(*config.Config) string
    description string  // What this file is used for
}

// Examples:
// - AgentSystemPromptValidator
// - TriageKubeconfigValidator (per cluster)
// - MigrationsPathValidator
```

#### 2. ConfigConsistencyValidator
Validates configuration value consistency (delegates to existing config.Validate()).

```go
type ConfigConsistencyValidator struct{}
```

## Integration with main.go

### Current Flow (Before)
```
main()
  ↓
Load config
  ↓
Setup logging
  ↓
Check system prompt (warn if missing) ← PROBLEM
  ↓
Initialize components (may fail)
  ↓
Run application
```

### Proposed Flow (After)
```
main()
  ↓
Load config
  ↓
Setup logging
  ↓
**Run preflight validation** ← NEW (fail fast)
  ↓
Initialize components
  ↓
Run application
```

### Implementation in main.go

```go
func run(cmd *cobra.Command, args []string) error {
    // Load configuration
    cfg, err := config.LoadWithConfigFile(configFile)
    if err != nil {
        return fmt.Errorf("failed to load configuration: %w", err)
    }

    // Setup structured logging
    setupLogging(cfg.LogLevel)

    // Run pre-flight validation (FAIL FAST on critical issues)
    slog.Info("running pre-flight validation...")
    if err := preflight.Validate(context.Background(), cfg); err != nil {
        slog.Error("pre-flight validation failed", "error", err)
        return fmt.Errorf("pre-flight validation failed:\n\n%w", err)
    }
    slog.Info("pre-flight validation passed")

    // Continue with component initialization...
}
```

## Validation Categories

### Category 1: Critical Files (MUST exist)

These files are required for core functionality:

1. **Agent System Prompt File** (`cfg.AgentSystemPromptFile`)
   - Used by: Agent execution
   - Current behavior: Warns and continues (BROKEN)
   - New behavior: Fail hard with remediation

2. **Triage Kubeconfig Files** (`cfg.Clusters[*].Triage.Kubeconfig`)
   - Used by: Agent Jobs (mounted as Secrets)
   - Current behavior: Validated in config but not checked for existence
   - New behavior: Fail hard if enabled cluster has missing kubeconfig

3. **Migration Path** (`cfg.StateStorage.MigrationsPath`)
   - Used by: Database initialization
   - Current behavior: Fails during state store init
   - New behavior: Fail during preflight (clearer error)

### Category 2: Configuration Consistency (already in config.Validate())

Keep existing validation in `config.Validate()`:
- API keys (at least one)
- Numeric ranges
- Enum values
- Storage configuration

The preflight validator will call `config.Validate()` as one check.

### Category 3: External Dependencies (OUT OF SCOPE)

These are NOT validated during preflight:
- Kubernetes cluster connectivity (bootstrap handles this)
- Object storage endpoint (deferred to first use)
- MCP server connectivity (handled by connection manager)

Rationale: Network checks add latency and may give false positives. Better to fail at component init with full context.

## Error Message Format

All validation errors follow this structure:

```
ERROR: <Check Name> failed

<Problem description>

Remediation:
  1. <Step 1>
  2. <Step 2>
  ...

Configuration reference:
  Key: <config_key>
  Value: <current_value>
  Source: <config_file|environment|default>
```

Example:

```
ERROR: Agent System Prompt File validation failed

The agent system prompt file does not exist: ./configs/triage-system-prompt.md

Remediation:
  1. Check that the file exists at the specified path
  2. Verify the path in configuration (agent_system_prompt_file)
  3. Ensure the file is readable by the nightcrier process
  4. Common location: ./configs/base-triage-prompt.md

Configuration reference:
  Key: agent_system_prompt_file
  Value: ./configs/triage-system-prompt.md
  Source: configs/config-weu.yaml
```

## Developer Guide (README.md)

The `internal/preflight/README.md` will document:

### When to Add a Validator

Add a pre-flight validator when:
1. A missing file/config causes silent failures or confusing errors
2. The requirement is **critical** for core functionality
3. The check is fast (< 100ms) and has no side effects
4. The error can be caught before any work begins

Do NOT add a validator for:
1. Network connectivity checks (defer to component init)
2. Optional features (use graceful degradation)
3. Deep validation of file contents (defer to component that uses it)

### How to Add a Validator

```go
// 1. Implement the Validator interface
type MyCustomValidator struct {
    description string
}

func (v *MyCustomValidator) Name() string {
    return "My Custom Check"
}

func (v *MyCustomValidator) Validate(ctx context.Context, cfg *config.Config) error {
    // Perform validation
    if someCondition {
        return fmt.Errorf("validation failed: %s\n\nRemediation:\n  1. Do this\n  2. Do that", v.description)
    }
    return nil
}

// 2. Register in preflight.go
func Validate(ctx context.Context, cfg *config.Config) error {
    runner := NewRunner()
    runner.Add(&ConfigConsistencyValidator{})
    runner.Add(&SystemPromptValidator{})
    runner.Add(&MyCustomValidator{description: "..."})  // ← Add here
    return runner.Run(ctx, cfg)
}
```

## Migration Strategy

### Phase 1: Create Package and Infrastructure
- Create `internal/preflight` package
- Implement Validator interface and Runner
- Add unit tests for infrastructure

### Phase 2: Migrate Existing Checks
- Move system prompt check from main.go
- Add triage kubeconfig validators
- Add migrations path validator
- Remove old scattered checks

### Phase 3: Integrate and Document
- Add preflight call to main.go
- Write developer README
- Update user-facing documentation

## Testing Strategy

### Unit Tests
- Test each validator in isolation
- Mock file system for file validators
- Test error message formatting

### Integration Tests
- Test full preflight runner with real config
- Test with missing files
- Test with invalid config values
- Verify error messages are helpful

### Manual Testing
- Delete system prompt file → verify clear error
- Misconfigure kubeconfig path → verify clear error
- Test with valid configuration → verify passes quickly
