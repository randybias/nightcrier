# Design: Configuration Structure Cleanup

## Overview

This document describes the technical approach for migrating from flat configuration keys to nested structure while removing obsolete options.

## Architecture Decision: Nested vs Flat Configuration

### Current State Analysis

The codebase has three configuration patterns:

1. **Nested (modern)** - Used by newer features:
   ```yaml
   object_storage:
     url: "..."
     signed_url_expiry: "..."
   state_storage:
     type: "sqlite"
     sqlite_path: "..."
   clusters:
     - name: "..."
       mcp:
         endpoint: "..."
   ```

2. **Flat (legacy)** - Used by agent/k8s config:
   ```yaml
   agent_cli: "claude"
   agent_model: "sonnet"
   k8s_namespace: "nightcrier"
   ```

3. **Dead code** - Defined but never consumed:
   ```yaml
   agent_script_path: "..."  # Unused
   agent_image: "..."        # Superseded by k8s_image
   skills:
     cache_dir: "..."        # Unused
   ```

### Decision: Migrate to Nested Structure

**Rationale:**
- Consistency with existing nested patterns
- Better discoverability (related options grouped together)
- Cleaner YAML structure
- Matches Viper's natural hierarchical binding

## Config Struct Changes

### Before

```go
type Config struct {
    // Agent Configuration (flat)
    AgentScriptPath       string `mapstructure:"agent_script_path"`       // REMOVE
    AgentSystemPromptFile string `mapstructure:"agent_system_prompt_file"`
    AgentAllowedTools     string `mapstructure:"agent_allowed_tools"`
    AgentModel            string `mapstructure:"agent_model"`
    AgentTimeout          int    `mapstructure:"agent_timeout"`
    AgentCLI              string `mapstructure:"agent_cli"`
    AgentImage            string `mapstructure:"agent_image"`             // REMOVE
    AgentVerbose          bool   `mapstructure:"agent_verbose"`           // REMOVE
    AdditionalAgentPrompt string `mapstructure:"additional_agent_prompt"`

    // K8s Configuration (flat)
    K8sNamespace       string `mapstructure:"k8s_namespace"`
    K8sImage           string `mapstructure:"k8s_image"`
    K8sImagePullPolicy string `mapstructure:"k8s_image_pull_policy"`
    K8sTimeout         int    `mapstructure:"k8s_timeout"`
    K8sMemoryLimit     string `mapstructure:"k8s_memory_limit"`
    K8sCPULimit        string `mapstructure:"k8s_cpu_limit"`
    K8sCleanupTTL      int    `mapstructure:"k8s_cleanup_ttl"`

    // Skills Configuration - REMOVE ENTIRELY
    Skills SkillsConfig `mapstructure:"skills"`
}
```

### After

```go
type Config struct {
    // ... other fields ...

    // Agent Configuration (nested)
    Agent AgentConfig `mapstructure:"agent"`

    // K8s Executor Configuration (nested)
    K8s K8sConfig `mapstructure:"k8s"`

    // Skills section REMOVED - not used
}

// AgentConfig holds AI agent runtime configuration.
type AgentConfig struct {
    // CLI is the AI CLI to use: claude, codex, goose, gemini
    CLI string `mapstructure:"cli"`

    // Model is the LLM model to use (model names depend on CLI)
    Model string `mapstructure:"model"`

    // Timeout is the maximum execution time in seconds
    Timeout int `mapstructure:"timeout"`

    // SystemPromptFile is the path to the system prompt file
    SystemPromptFile string `mapstructure:"system_prompt_file"`

    // AllowedTools is a comma-separated list of allowed tools
    AllowedTools string `mapstructure:"allowed_tools"`

    // AdditionalPrompt is optional cluster-specific context
    AdditionalPrompt string `mapstructure:"additional_prompt"`
}

// K8sConfig holds Kubernetes executor configuration.
type K8sConfig struct {
    // Namespace where Jobs and ConfigMaps are created
    Namespace string `mapstructure:"namespace"`

    // Image is the container image for the agent runner
    Image string `mapstructure:"image"`

    // ImagePullPolicy: Always, Never, IfNotPresent
    ImagePullPolicy string `mapstructure:"image_pull_policy"`

    // Timeout is the Job timeout in seconds
    Timeout int `mapstructure:"timeout"`

    // MemoryLimit for Job containers (e.g., "2Gi")
    MemoryLimit string `mapstructure:"memory_limit"`

    // CPULimit for Job containers (e.g., "1")
    CPULimit string `mapstructure:"cpu_limit"`

    // CleanupTTL is the TTL for Job cleanup after completion (seconds)
    CleanupTTL int `mapstructure:"cleanup_ttl"`
}
```

## Environment Variable Mapping

Environment variables remain unchanged for compatibility:

| Old Flat Key | New Nested Key | Env Var (unchanged) |
|--------------|----------------|---------------------|
| `agent_cli` | `agent.cli` | `AGENT_CLI` |
| `agent_model` | `agent.model` | `AGENT_MODEL` |
| `agent_timeout` | `agent.timeout` | `AGENT_TIMEOUT` |
| `agent_system_prompt_file` | `agent.system_prompt_file` | `AGENT_SYSTEM_PROMPT_FILE` |
| `agent_allowed_tools` | `agent.allowed_tools` | `AGENT_ALLOWED_TOOLS` |
| `additional_agent_prompt` | `agent.additional_prompt` | `ADDITIONAL_AGENT_PROMPT` |
| `k8s_namespace` | `k8s.namespace` | `K8S_NAMESPACE` |
| `k8s_image` | `k8s.image` | `K8S_IMAGE` |
| `k8s_image_pull_policy` | `k8s.image_pull_policy` | `K8S_IMAGE_PULL_POLICY` |
| `k8s_timeout` | `k8s.timeout` | `K8S_TIMEOUT` |
| `k8s_memory_limit` | `k8s.memory_limit` | `K8S_MEMORY_LIMIT` |
| `k8s_cpu_limit` | `k8s.cpu_limit` | `K8S_CPU_LIMIT` |
| `k8s_cleanup_ttl` | `k8s.cleanup_ttl` | `K8S_CLEANUP_TTL` |

**Removed (no replacement):**
- `AGENT_SCRIPT_PATH` - obsolete
- `AGENT_IMAGE` - superseded by `K8S_IMAGE`
- `AGENT_VERBOSE` - never used
- `SKILLS_CACHE_DIR` - never used
- `SKILLS_DISABLE_TRIAGE_PRELOAD` - never used

## Code Changes

### main.go Updates

```go
// Before
k8sExecCfg := agent.K8sExecutorConfig{
    Namespace:        cfg.K8sNamespace,
    Image:            cfg.K8sImage,
    ImagePullPolicy:  cfg.K8sImagePullPolicy,
    Timeout:          cfg.K8sTimeout,
    MemoryLimit:      cfg.K8sMemoryLimit,
    CPULimit:         cfg.K8sCPULimit,
    CleanupTTL:       int32(cfg.K8sCleanupTTL),
    AgentCLI:         cfg.AgentCLI,
    Model:            cfg.AgentModel,
    SystemPromptFile: cfg.AgentSystemPromptFile,
    Debug:            cfg.LogLevel == "debug",
}

// After
k8sExecCfg := agent.K8sExecutorConfig{
    Namespace:        cfg.K8s.Namespace,
    Image:            cfg.K8s.Image,
    ImagePullPolicy:  cfg.K8s.ImagePullPolicy,
    Timeout:          cfg.K8s.Timeout,
    MemoryLimit:      cfg.K8s.MemoryLimit,
    CPULimit:         cfg.K8s.CPULimit,
    CleanupTTL:       int32(cfg.K8s.CleanupTTL),
    AgentCLI:         cfg.Agent.CLI,
    Model:            cfg.Agent.Model,
    SystemPromptFile: cfg.Agent.SystemPromptFile,
    Debug:            cfg.LogLevel == "debug",
}
```

### Validation Updates

Move validation into struct methods where appropriate:

```go
func (a *AgentConfig) Validate() error {
    if a.CLI == "" {
        return fmt.Errorf("agent.cli is required")
    }
    validCLIs := map[string]bool{"claude": true, "codex": true, "goose": true, "gemini": true}
    if !validCLIs[a.CLI] {
        return fmt.Errorf("invalid agent.cli '%s'", a.CLI)
    }
    if a.Model == "" {
        return fmt.Errorf("agent.model is required")
    }
    if a.Timeout < 1 {
        return fmt.Errorf("agent.timeout must be >= 1")
    }
    return nil
}

func (k *K8sConfig) Validate() error {
    validPolicies := map[string]bool{"Always": true, "Never": true, "IfNotPresent": true}
    if k.ImagePullPolicy != "" && !validPolicies[k.ImagePullPolicy] {
        return fmt.Errorf("invalid k8s.image_pull_policy '%s'", k.ImagePullPolicy)
    }
    return nil
}

func (k *K8sConfig) ApplyDefaults() {
    if k.Namespace == "" {
        k.Namespace = "nightcrier"
    }
    if k.Image == "" {
        k.Image = "nc-agent-runner:latest"
    }
    if k.ImagePullPolicy == "" {
        k.ImagePullPolicy = "IfNotPresent"
    }
    if k.Timeout == 0 {
        k.Timeout = 600
    }
    if k.MemoryLimit == "" {
        k.MemoryLimit = "2Gi"
    }
    if k.CPULimit == "" {
        k.CPULimit = "1"
    }
    if k.CleanupTTL == 0 {
        k.CleanupTTL = 3600
    }
}
```

## Test Migration Strategy

### config_test.go Changes

All test YAML fixtures need to be updated. Example transformation:

```yaml
# Before
agent_script_path: "./agent-container/run-agent.sh"
agent_system_prompt_file: "./configs/base-triage-prompt.md"
agent_allowed_tools: "Read,Write,Grep,Glob,Bash,Skill"
agent_model: "sonnet"
agent_timeout: 300
agent_cli: "claude"
agent_image: "nightcrier-agent:latest"
k8s_namespace: "nightcrier"
k8s_image: "nc-agent-runner:latest"

# After
agent:
  cli: "claude"
  model: "sonnet"
  timeout: 300
  system_prompt_file: "./configs/base-triage-prompt.md"
  allowed_tools: "Read,Write,Grep,Glob,Bash,Skill"
k8s:
  namespace: "nightcrier"
  image: "nc-agent-runner:latest"
```

## Directory Cleanup

### agent-container/

Current state:
```
agent-container/
└── scratch/    # empty
```

Action: Remove entire directory. No code references it.

### agent-home/

Current state:
```
agent-home/
└── skills/
    └── k8s4agents/    # cloned repo, but unused
```

Action: Remove entire directory. Skills are cloned inside container at runtime.

## Rollout Plan

1. **Phase 1: Remove dead code**
   - Remove `AgentScriptPath`, `AgentImage`, `AgentVerbose` from config
   - Remove `SkillsConfig` struct and related bindings
   - Update tests that reference these

2. **Phase 2: Create new nested structs**
   - Add `AgentConfig` and `K8sConfig` structs
   - Keep old flat fields temporarily for reference

3. **Phase 3: Migrate to nested fields**
   - Update `main.go` to use nested accessors
   - Update validation to use nested fields
   - Update status display

4. **Phase 4: Remove flat fields**
   - Remove all old `Agent*` and `K8s*` flat fields
   - Update all test fixtures

5. **Phase 5: Cleanup**
   - Remove obsolete directories
   - Update all config example files
   - Final test validation
