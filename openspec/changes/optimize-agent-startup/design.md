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
│  │ 3. Run incident_triage.sh --skip-dump │  │
│  │ 4. Bundle into PRELOADED_CONTEXT      │  │
│  └───────────────┬───────────────────────┘  │
│                  │                           │
│                  ▼                           │
│  ┌───────────────────────────────────────┐  │
│  │ Build Enhanced Prompt:                │  │
│  │  - System prompt (from file)          │  │
│  │  - Preloaded context (embedded)       │  │
│  │  - Report template instructions       │  │
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
        context+="## Incident Context\n\n"
        context+="<incident>\n$(cat "$workspace_dir/incident.json")\n</incident>\n\n"
    fi

    # Read permissions
    if [[ -f "$workspace_dir/incident_cluster_permissions.json" ]]; then
        context+="## Cluster Permissions\n\n"
        context+="<permissions>\n$(cat "$workspace_dir/incident_cluster_permissions.json")\n</permissions>\n\n"
    fi

    # Run baseline triage (with error handling)
    if command -v kubectl &>/dev/null && [[ -f ~/.claude/skills/k8s-troubleshooter/scripts/incident_triage.sh ]]; then
        local triage_output
        triage_output=$(~/.claude/skills/k8s-troubleshooter/scripts/incident_triage.sh --skip-dump 2>&1 || echo "Triage failed")
        context+="## Baseline Triage Results\n\n"
        context+="<triage>\n${triage_output}\n</triage>\n\n"
    fi

    echo "$context"
}
```

**Integration Point**: Called from `run-agent.sh` before building agent command

### 2. Enhanced Prompt Builder

**Location**: `agent-container/run-agent.sh`

**Modified Section**: Prompt construction (around line 300-350)

```bash
# Build final prompt with preloaded context
PRELOADED_CONTEXT=$(preload_incident_context "$WORKSPACE_DIR")

FINAL_PROMPT="${SYSTEM_PROMPT}

${PRELOADED_CONTEXT}

## Investigation Task

${USER_PROMPT}

## Report Structure

Your investigation report MUST follow this exact structure:

### 1. Problem Statement
[2-3 sentence description of the incident]

### 2. Summary of Findings
**Root Cause**: [concise root cause]
**Confidence Level**: [percentage with justification]

### 3. Recommended Immediate Remediation Steps
1. [prioritized action]
2. [prioritized action]
...

### 4. Supporting Evidence and Work Done
[detailed kubectl outputs, logs, diagnostic steps taken]
"
```

### 3. Report Structure Validation

**Location**: New file `agent-container/validate-report.sh`

**Purpose**: Post-execution validation that report follows template

```bash
#!/usr/bin/env bash
# validate-report.sh - Verify investigation report structure

validate_report_structure() {
    local report_file="$1"

    # Check for required sections
    local required_sections=(
        "Problem Statement"
        "Summary of Findings"
        "Recommended Immediate Remediation Steps"
        "Supporting Evidence and Work Done"
    )

    for section in "${required_sections[@]}"; do
        if ! grep -q "### .*${section}" "$report_file"; then
            echo "ERROR: Missing required section: $section" >&2
            return 1
        fi
    done

    # Check for Root Cause and Confidence Level
    if ! grep -q "**Root Cause**:" "$report_file"; then
        echo "ERROR: Missing Root Cause in Summary of Findings" >&2
        return 1
    fi

    if ! grep -q "**Confidence Level**:" "$report_file"; then
        echo "ERROR: Missing Confidence Level in Summary of Findings" >&2
        return 1
    fi

    return 0
}
```

**Integration**: Called from `run-agent.sh` after agent completes, before success/failure determination

## Trade-offs

### Context Preloading

**Pros:**
- Eliminates 3-5 agent turns (30-60s time savings)
- Reduces token usage by ~20-30%
- Provides consistent baseline diagnostics
- Agents can jump straight to deep investigation

**Cons:**
- Increases initial prompt size (may hit model context limits with very large incidents)
- Host must have kubectl access and triage script available
- Triage failures could delay agent start

**Decision**: Proceed with preloading, implement graceful degradation if triage fails

### Structured Report Template

**Pros:**
- Predictable parsing for downstream systems
- Consistent user experience
- Ensures key information (root cause, confidence) always present
- Easier to compare reports across agents

**Cons:**
- May constrain agent creativity in unusual incidents
- Requires template adherence monitoring
- Migration burden for existing report consumers

**Decision**: Implement strict structure with optional "Additional Notes" section for flexibility

## Performance Expectations

Based on current test runs:

| Metric | Current | Target | Improvement |
|--------|---------|--------|-------------|
| Time to first investigation turn | 30-45s | 5-10s | 70-80% |
| Total investigation time | 90-120s | 60-80s | 30-40% |
| Token usage (input) | ~8,000 | ~5,500 | 30% |
| Report consistency | ~40% | 100% | 150% |

## Rollout Strategy

1. **Phase 1**: Implement context preloading (can be deployed independently)
2. **Phase 2**: Add report structure validation (warn-only mode)
3. **Phase 3**: Enforce report structure (error on non-compliance)
4. **Phase 4**: Deprecate old report format after 2-week migration period

## Testing Strategy

1. **Unit tests**: Validate context bundling functions
2. **Integration tests**: Run agents with preloaded context against test incidents
3. **Comparison tests**: Run same incidents with/without preloading, compare:
   - Time to completion
   - Token usage
   - Investigation quality (confidence levels, root cause accuracy)
4. **Structure validation**: Automated checks that all reports pass validation

## Open Questions

1. **Context size limits**: What's the maximum safe context size before hitting model limits?
   - **Answer**: Monitor in testing, implement truncation at 10,000 tokens

2. **Triage failure handling**: Should agent proceed without triage or fail fast?
   - **Answer**: Proceed with warning, mark context as incomplete

3. **Report version migration**: How to handle old report format during transition?
   - **Answer**: Version header in reports, parser supports both for 2 weeks
