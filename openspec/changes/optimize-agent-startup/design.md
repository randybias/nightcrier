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
    local context=""

    # Read incident.json
    if [[ -f "$workspace_dir/incident.json" ]]; then
        context+="<incident>\n$(cat "$workspace_dir/incident.json")\n</incident>\n\n"
    fi

    # Read permissions
    if [[ -f "$workspace_dir/incident_cluster_permissions.json" ]]; then
        context+="<permissions>\n$(cat "$workspace_dir/incident_cluster_permissions.json")\n</permissions>\n\n"
    fi

    # Run skill's baseline triage (with error handling)
    local triage_script="$SKILLS_DIR/k8s-troubleshooter/scripts/incident_triage.sh"
    if [[ -x "$triage_script" ]]; then
        local triage_output
        if triage_output=$(timeout 30 "$triage_script" --skip-dump 2>&1); then
            context+="<triage>\n${triage_output}\n</triage>\n\n"
        else
            log_warning "Triage script failed or timed out, proceeding without baseline triage"
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
- Truncation priority: triage output first (incident.json and permissions never truncated)

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
| Preloaded context | Domain-specific data (incident, permissions, triage) |
| Report format | Defined by skill (see k8s4agents `add-standardized-report-format`) |
| Runner | Orchestrates preloading and prompt assembly |

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
