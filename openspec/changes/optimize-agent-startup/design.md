# Design: Optimize Agent Startup

## Architecture Overview

This change introduces a **context preloading layer** between nightcrier's agent invocation and the agent execution, bundling diagnostic context into the initial prompt.

```
┌─────────────────┐
│   Nightcrier    │
│  (Go process)   │
└────────┬────────┘
         │ Invokes run-agent.sh with WORKSPACE_DIR
         ▼
┌─────────────────────────────────────────────┐
│          run-agent.sh (Host)                │
│  ┌───────────────────────────────────────┐  │
│  │ 1. Read incident.json                 │  │
│  │ 2. Read incident_cluster_permissions  │  │
│  │ 3. Run skill triage script            │  │
│  │ 4. Bundle into PRELOADED_CONTEXT      │  │
│  └───────────────┬───────────────────────┘  │
│                  │                           │
│                  ▼                           │
│  ┌───────────────────────────────────────┐  │
│  │ Build Enhanced Prompt:                │  │
│  │  - System prompt (generic, from file) │  │
│  │  - Preloaded context (domain-specific)│  │
│  │  - User prompt                        │  │
│  └───────────────┬───────────────────────┘  │
└──────────────────┼───────────────────────────┘
                   │ Pass to container
                   ▼
         ┌──────────────────┐
         │ Agent Container  │
         │  (Docker)        │
         │                  │
         │ Agent starts     │
         │ with full        │
         │ context already  │
         │ loaded           │
         └──────────────────┘
```

## Component Design

### 1. Context Preloading Module

**Location**: `agent-container/runners/common.sh`

**New Function**: `preload_incident_context()`

```bash
preload_incident_context() {
    local workspace_dir="$1"
    local skills_cache_dir="$2"
    local disable_triage="${3:-false}"
    local context=""

    # Read incident.json
    if [[ -f "$workspace_dir/incident.json" ]]; then
        context+="<incident>\n$(cat "$workspace_dir/incident.json")\n</incident>\n\n"
    fi

    # Read permissions
    if [[ -f "$workspace_dir/incident_cluster_permissions.json" ]]; then
        context+="<permissions>\n$(cat "$workspace_dir/incident_cluster_permissions.json")\n</permissions>\n\n"
    fi

    # Run K8s skill's baseline triage (with error handling)
    if [[ "$disable_triage" != "true" ]]; then
        local triage_script="${skills_cache_dir}/k8s-troubleshooter/scripts/incident_triage.sh"
        if [[ -x "$triage_script" ]]; then
            local triage_output
            if triage_output=$(timeout 30 "$triage_script" --skip-dump 2>&1); then
                context+="<initial_triage_report>\n${triage_output}\n</initial_triage_report>\n\n"
            else
                log_warning "K8s triage script failed, agent will run triage itself"
            fi
        else
            log_debug "K8s triage script not found at: $triage_script"
            log_debug "Agent will run triage via skill"
        fi
    fi

    echo "$context"
}
```

**Integration Point**: Called from `run-agent.sh` before building agent command

### 2. Enhanced Prompt Builder

**Location**: `agent-container/run-agent.sh`

**Modified Section**: Prompt construction

```bash
# Build final prompt with preloaded context
PRELOADED_CONTEXT=$(preload_incident_context "$WORKSPACE_DIR")

# Combine: system prompt + preloaded context + user prompt
# System prompt is generic (from configs/triage-system-prompt.md)
# Preloaded context provides domain-specific data
# User prompt is the investigation request
```

### 3. Context Size Management

**Token Estimation**: 4 characters ≈ 1 token

**Limits**:
- Warning threshold: 8,000 tokens
- Truncation threshold: 10,000 tokens
- Truncation priority: initial_triage_report output first (incident.json and permissions never truncated)

## Trade-offs

### Context Preloading

**Pros:**
- Eliminates 3-5 agent turns (30-60s time savings)
- Reduces token usage by ~20-30%
- Provides consistent baseline diagnostics
- Agents can jump straight to deep investigation

**Cons:**
- Increases initial prompt size (may hit model context limits with very large incidents)
- Host must have skill triage script available
- Triage failures could delay agent start (mitigated by timeout and graceful degradation)

**Decision**: Proceed with preloading, implement graceful degradation if triage fails

## Performance Expectations

Based on current test runs:

| Metric | Current | Target | Improvement |
|--------|---------|--------|-------------|
| Time to first investigation turn | 30-45s | 5-10s | 70-80% |
| Total investigation time | 90-120s | 60-80s | 30-40% |
| Token usage (input) | ~8,000 | ~5,500 | 30% |

## Separation of Concerns

This design maintains clean separation:

| Component | Responsibility |
|-----------|----------------|
| System prompt | Generic IT triage guidance (see `generalize-triage-system-prompt`) |
| Preloaded context | K8s-specific data (incident, permissions, initial_triage_report) |
| Report format | Defined by skill (see k8s4agents `add-standardized-report-format`) |
| Runner | Orchestrates preloading and prompt assembly |
| Skills cache | K8s-troubleshooter skill stored in `./agent-home/skills/` |

## Testing Strategy

1. **Unit tests**: Validate context bundling functions
2. **Integration tests**: Run agents with preloaded context against test incidents
3. **Comparison tests**: Run same incidents with/without preloading, compare:
   - Time to completion
   - Token usage
   - Investigation quality

## Open Questions

1. **Context size limits**: What's the maximum safe context size before hitting model limits?
   - **Answer**: Monitor in testing, implement truncation at 10,000 tokens

2. **Triage failure handling**: Should agent proceed without triage or fail fast?
   - **Answer**: Proceed with warning, mark context as incomplete

## Implementation Status

### Phase 1: Configuration and Foundation ✓ Complete

**Skills Configuration** (Task 1.1):
- Added `SkillsConfig` struct to `internal/config/config.go`
- Fields: `cache_dir` (default: "./agent-home/skills"), `disable_triage_preload` (default: false)
- Environment variables: `SKILLS_CACHE_DIR`, `SKILLS_DISABLE_TRIAGE_PRELOAD`
- Updated example configs: `configs/config.example.yaml`, `configs/config-multicluster.yaml`

**Automatic Skills Caching** (Task 1.2):
- Created `internal/skills/cache.go` with `EnsureSkillsCached()` function
- Automatically clones k8s4agents from GitHub on first run if triage script not found
- Integrated into `cmd/nightcrier/main.go` startup sequence (called before main loop)
- Non-fatal: Logs warning and continues if git clone fails
- Location: `./agent-home/skills/k8s-troubleshooter/` (relative to nightcrier working directory)

**Context Preloading Function** (Task 1.3-1.4):
- Implemented `preload_incident_context()` in `agent-container/runners/common.sh`
- Reads incident.json → wraps in `<incident>` tags
- Reads incident_cluster_permissions.json → wraps in `<permissions>` tags
- Executes K8s triage script: `${skills_cache_dir}/k8s-troubleshooter/scripts/incident_triage.sh --skip-dump`
- Timeout: 30 seconds
- Wraps triage output in `<initial_triage_report>` tags
- Graceful degradation: Logs warning if triage fails, continues without it

**Runner Integration** (Task 1.5):
- Modified `agent-container/run-agent.sh` to read `SKILLS_CACHE_DIR` from environment
- Calls `preload_incident_context()` before building agent command
- Stores result in `PRELOADED_CONTEXT` variable
- Exports to agent-specific runners

**Context Size Monitoring** (Task 1.6):
- Implemented `monitor_context_size()` function in `runners/common.sh`
- Token estimation: 4 characters ≈ 1 token
- Warning threshold: 8,000 tokens
- Truncation strategy documented: Truncate `<initial_triage_report>` first if needed

### Phase 2: Agent Runner Updates ✓ Complete

All four agent runners updated to inject preloaded context:

**Claude Runner** (Task 2.1):
- Modified `agent-container/runners/claude.sh`
- Injects via `--append-system-prompt` flag (after system prompt file)
- Proper escaping with `escape_single_quotes()` function

**Codex Runner** (Task 2.2):
- Modified `agent-container/runners/codex.sh`
- Prepends context to user prompt with proper escaping

**Gemini Runner** (Task 2.3):
- Modified `agent-container/runners/gemini.sh`
- Prepends context to user prompt with proper escaping

**Goose Runner** (Task 2.4):
- Modified `agent-container/runners/goose.sh`
- Prepends context to user prompt with proper escaping

### Phase 3: Bug Fixes and Refinements ✓ Complete

**Skill Path Mismatch** (Task 3.1):
- Fixed path in `common.sh` to include nested `skills/` directory
- Corrected: `k8s-troubleshooter/scripts/incident_triage.sh` → `k8s4agents/skills/k8s-troubleshooter/scripts/incident_triage.sh`
- Issue: k8s4agents repo contains multiple skills, not just k8s-troubleshooter

**Environment Variable Wiring** (Task 3.2):
- Added `SkillsCacheDir` and `DisableTriagePreload` fields to `ExecutorConfig` struct
- Modified `internal/agent/executor.go` to pass `SKILLS_DIR` and `DISABLE_TRIAGE_PRELOAD` to run-agent.sh
- Wired config values through `cmd/nightcrier/main.go`

**Directory Naming Clarification** (Task 3.3):
- Changed `K8sSkillName` constant from "k8s-troubleshooter" to "k8s4agents" in `internal/skills/cache.go`
- Updated all paths to use `k8s4agents` directory name to match repo name
- Prevents confusion between repo (k8s4agents) and individual skill (k8s-troubleshooter)

**Audit Trail Accuracy** (Task 3.4):
- **Problem**: `prompt-sent.md` was captured by Go before bash did preloading
- **Solution**: Added `append_preloaded_context_to_audit()` function to `common.sh`
- Called from `run-agent.sh` after preloading completes but before execution
- Appends full preloaded context to audit file for debugging and compliance

**Context Tag Clarity** (Task 3.5):
- Renamed `<permissions>` tag to `<kubernetes_cluster_access_permissions>`
- Makes it clear what the permissions are for and how they relate to Kubernetes
- Updated in `agent-container/runners/common.sh`

**System Prompt for Skill Compliance** (Task 3.6):
- **Problem**: Agent was generating traditional narrative reports instead of following skill's 7-section template
- **Root Cause**: System prompt said "use your skill" but didn't tell agent WHERE it was or to READ it
- **Solution**: Updated `configs/triage-system-prompt.md` to:
  - Remove `incident.json` file reference (data is preloaded)
  - Add explicit instruction to READ skill file first: `cat ~/.claude/skills/k8s4agents/skills/k8s-troubleshooter/SKILL.md`
  - List all 7 mandatory report sections with "CRITICAL" and "MUST" language
  - Reference specific skill elements (FACT-n/INF-n labels, H1/H2/H3 hypothesis ranking, emoji status indicators)

### Phase 4: Testing and Validation

**Initial Testing** (Task 4.1): ✓ Complete
- Tested with CrashLoopBackOff incident
- Verified preloading works (visible in docker command logs with `--append-system-prompt`)
- Confirmed context includes all three sections: incident, kubernetes_cluster_access_permissions, initial_triage_report
- Validated audit trail accuracy after prompt-sent.md fixes
- User confirmed: "it's working"

**Remaining Tests** (Task 4.2-4.4): Pending
- Comparison testing (with/without preloading)
- Multi-agent testing (Claude, Codex, Gemini, Goose)
- Performance benchmarking

### Phase 5: Documentation - In Progress

**gitignore** (Task 5.1): ✓ Complete
- Added `/agent-home/` to `.gitignore`

**Documentation Updates** (Task 5.2-5.3): Pending
- agent-container README update needed
- Main README or setup docs update needed

## Key Design Decisions Made During Implementation

### 1. Automatic Skill Caching
**Decision**: Implement automatic caching in Go startup sequence rather than requiring manual setup.

**Rationale**: Production systems should not require manual operations. Automatic caching ensures skills are always available and eliminates setup steps.

**Implementation**:
- Check if triage script exists at startup
- Clone k8s4agents from GitHub if missing
- Non-fatal: Log warning and continue if clone fails
- Location: `./agent-home/skills/` (relative to nightcrier root, gitignored)

### 2. Configuration Priority
**Decision**: Config file defaults with environment variable overrides.

**Pattern**: Code default → Config file → Environment variable → CLI flag

**Rationale**: Production systems need declarative configuration files as the primary source of truth, with environment variables available for deployment-specific overrides.

### 3. Single Triage Script Path
**Decision**: Use single, well-defined path for triage script rather than searching multiple locations.

**Path**: `${skills_cache_dir}/k8s-troubleshooter/scripts/incident_triage.sh`

**Rationale**: Avoids potential conflicts from discovering multiple triage scripts, provides clear expectations for skill structure.

### 4. XML-Style Tag Naming
**Decision**: Use `<initial_triage_report>` instead of `<triage>` for preloaded triage output.

**Rationale**: Clarifies that this is the initial baseline triage, not the full triage process the agent performs. Avoids confusion with the overall triage workflow.

### 5. Audit Trail Completeness (Phase 3 Decision)
**Decision**: Append preloaded context to `prompt-sent.md` after preloading completes.

**Problem**: Go's `capturePrompt()` was called before bash preloading, so audit file didn't show actual context sent to agent.

**Solution**: Added bash function to append preloaded context to existing `prompt-sent.md` after preloading but before execution.

**Rationale**: Accurate audit trails are essential for debugging and compliance. Without seeing the actual preloaded context, we couldn't diagnose why agents weren't following skill templates.

### 6. Descriptive Context Tags (Phase 3 Decision)
**Decision**: Use `<kubernetes_cluster_access_permissions>` instead of generic `<permissions>`.

**Rationale**: Generic tag names are ambiguous. Agents need clear context about what permissions are for and how they relate to the investigation domain (Kubernetes cluster access).

### 7. Explicit Skill Reading Instructions (Phase 3 Decision)
**Decision**: System prompt must explicitly tell agent to READ skill file, not just "use your skill".

**Problem**: Agents generated traditional narrative reports instead of following skill's mandatory 7-section template, even though skill was mounted and accessible.

**Root Cause**: System prompt said "use your mounted skill" but didn't tell agent WHERE it was (`~/.claude/skills/k8s4agents/skills/k8s-troubleshooter/SKILL.md`) or to READ it first.

**Solution**: Updated system prompt with:
- "Required First Step" section with bash command to cat skill file
- Explicit listing of all 7 mandatory report sections
- "CRITICAL" and "MUST" language to emphasize requirement
- References to specific skill format elements (FACT-n/INF-n labels, H1/H2/H3 hypothesis ranking, emoji status indicators)

**Rationale**: Skills are powerful but only if agents actually read them. Vague guidance ("use your skill") is insufficient; explicit, actionable instructions are required.
