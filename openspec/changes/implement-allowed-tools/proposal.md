# Proposal: Implement Allowed Tools

## Summary

Wire up the existing `agent.allowed_tools` config field to restrict agent tool usage, with agent-specific translation for each supported CLI (Claude, Codex, Gemini, Goose). Add a test mode to verify security boundaries are enforced.

## Motivation

The `AllowedTools` config field exists but is only displayed in the startup banner - it's never passed to agents. This is a security gap: operators expect tool restrictions to be enforced but they currently aren't.

**Current state:**
- Config field exists: `internal/config/config.go:29-30`
- Example configs show intended use: `allowed_tools: "Read,Write,Grep,Glob,Bash,Skill"`
- entrypoint.sh hardcodes: `--allowedTools "Read,Grep,Glob,Bash,Write"` (Claude only)
- Other agents have no tool restrictions at all

**Security concern:** Without tool restrictions, agents could:
- Execute arbitrary code via unrestricted Bash
- Modify files they shouldn't touch
- Make network calls (if WebFetch is available)

## Agent-Specific Mechanisms

Each agent CLI has different tool restriction mechanisms:

| Agent | Mechanism | Format | Notes |
|-------|-----------|--------|-------|
| Claude | `--allowedTools` flag | Comma-separated | `Read,Write,Grep,Glob,Bash` |
| Codex | `--allow` flag (per tool) | Multiple flags | `--allow Read --allow Write` |
| Gemini | Not supported | N/A | Gemini CLI has no tool restriction mechanism |
| Goose | `--tools` or config | Tool list | May require profile config |

### Translation Layer

Since operators configure tools in a generic format, we need a translation layer:

```
Config: allowed_tools: "Read,Write,Grep,Glob,Bash"
           ↓
    Translation Layer
           ↓
Claude:  --allowedTools "Read,Write,Grep,Glob,Bash"
Codex:   --allow Read --allow Write --allow Grep --allow Glob --allow Bash
Gemini:  (log warning: tool restrictions not supported)
Goose:   --tools "read,write,grep,glob,bash" (lowercase)
```

## Design

### Data Flow

```
Config (agent.allowed_tools)
    ↓
K8sExecutor.loadIncidentData()
    ↓
Job environment variable: ALLOWED_TOOLS
    ↓
entrypoint.sh translate_allowed_tools()
    ↓
Agent-specific flags in run_agent()
```

### New Environment Variable

Add `ALLOWED_TOOLS` to Job spec, passed from config.

### Entrypoint Changes

Add `translate_allowed_tools()` function that outputs agent-specific flags:

```bash
translate_allowed_tools() {
    local tools="$1"
    local agent="$2"

    case "$agent" in
        claude)
            echo "--allowedTools \"$tools\""
            ;;
        codex)
            # Convert to --allow flags
            IFS=',' read -ra TOOL_ARRAY <<< "$tools"
            for tool in "${TOOL_ARRAY[@]}"; do
                echo -n "--allow $tool "
            done
            ;;
        gemini)
            echo "# Warning: Gemini does not support tool restrictions" >&2
            ;;
        goose)
            # Goose uses lowercase tool names
            echo "--tools \"${tools,,}\""
            ;;
    esac
}
```

## Test Mode for Security Verification

Add a `--dry-run-tools` flag or test harness mode that:

1. Starts agent with restricted tools
2. Attempts to use a forbidden tool (e.g., WebFetch if not in allowed list)
3. Verifies the tool call is rejected
4. Reports pass/fail

This allows operators to verify their tool restrictions are working before production use.

### Test Scenarios

| Test | Expected Behavior |
|------|-------------------|
| Bash allowed, WebFetch forbidden | Bash commands work, web fetches rejected |
| Read allowed, Write forbidden | Can read files, cannot create/modify |
| All tools allowed | No restrictions |
| Empty allowed_tools | Agent-default behavior (warn operator) |

## File Changes

| File | Change |
|------|--------|
| `internal/agent/k8s/job.go` | Add `ALLOWED_TOOLS` env var to Job spec |
| `internal/agent/k8s_executor.go` | Pass `AllowedTools` from config to Job |
| `nc-agent-runner/entrypoint.sh` | Add `translate_allowed_tools()`, update `run_agent()` |
| `configs/config.example.yaml` | Document tool restriction format per agent |

## Scope

- Pass `AllowedTools` from config to agent container
- Translate to agent-specific flags
- Research and implement for all supported agents (Claude, Codex, Gemini, Goose)

## Out of Scope

- Per-tool granular permissions (e.g., "Bash only for kubectl")
- Runtime tool restriction changes
- Tool restriction for custom MCP servers
- Test mode for verification (handled by release test suite)

## Design Decisions

1. **Default behavior:** If `allowed_tools` is empty, use agent defaults (permissive). Operators decide their own security posture.

2. **Gemini handling:** Needs further research - there's likely some mechanism. Task added to investigate before implementation.

3. **Test mode:** Tool restriction verification will be part of the release test suite, not core runtime. Out of scope for this proposal.
