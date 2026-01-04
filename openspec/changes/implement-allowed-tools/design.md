# Design: Implement Allowed Tools

## 1. Overview

This document details the architectural decisions for implementing tool restrictions across multiple AI agent CLIs.

## 2. Agent CLI Research

### 2.1 Claude Code

**Mechanism:** `--allowedTools` flag

```bash
claude -p "prompt" --allowedTools "Read,Write,Grep,Glob,Bash"
```

**Format:** Comma-separated tool names, case-sensitive.

**Available tools:** Read, Write, Edit, Bash, Glob, Grep, WebFetch, WebSearch, Task, TodoWrite, Skill, NotebookEdit

**Behavior when restricted:**
- Tool calls to non-allowed tools return error
- Agent is informed the tool is not available

### 2.2 Codex (OpenAI)

**Mechanism:** `--allow` flag (repeated per tool) or `--dangerously-bypass-approvals-and-sandbox`

```bash
codex exec --allow Read --allow Write --allow Bash "prompt"
```

**Note:** Current entrypoint uses `--dangerously-bypass-approvals-and-sandbox` which disables all restrictions. Need to replace with explicit `--allow` flags.

**Available tools:** Read, Write, Bash, Grep, Glob (similar to Claude)

### 2.3 Gemini CLI

**Mechanism:** Needs research.

**Current state:** Gemini CLI (`gemini`) with `--yolo` flag auto-approves all tool calls. Tool restriction mechanism not yet identified.

**Action:** Research `gemini --help`, documentation, and config options to identify restriction mechanism. There's likely some option available.

### 2.4 Goose

**Mechanism:** `--profile` with tool configuration or `GOOSE_TOOLKITS` env var

```bash
goose run --profile restricted --text "prompt"
```

Profile config (`~/.config/goose/profiles.yaml`):
```yaml
restricted:
  toolkits:
    - developer  # Provides Read, Write, Bash, etc.
```

**Alternative:** May support `--tools` flag in newer versions.

**Recommendation:** Research current Goose version, implement if supported.

## 3. Translation Layer Design

### 3.1 Function Signature

```bash
# translate_allowed_tools <tools_csv> <agent_cli>
# Outputs: Agent-specific flags to stdout
# Errors: Warnings to stderr
translate_allowed_tools() {
    local tools="$1"
    local agent="$2"

    # Handle empty/unset
    if [[ -z "$tools" ]]; then
        echo "# No tool restrictions configured" >&2
        return 0
    fi

    case "$agent" in
        claude)
            echo "--allowedTools \"$tools\""
            ;;
        codex)
            local flags=""
            IFS=',' read -ra TOOL_ARRAY <<< "$tools"
            for tool in "${TOOL_ARRAY[@]}"; do
                flags+="--allow ${tool} "
            done
            echo "$flags"
            ;;
        gemini)
            echo "# WARNING: Gemini CLI does not support tool restrictions" >&2
            echo "# Configured restrictions will NOT be enforced: $tools" >&2
            ;;
        goose)
            # Goose may need profile-based approach
            echo "# Goose tool restriction requires profile config" >&2
            ;;
    esac
}
```

### 3.2 Tool Name Mapping

Some agents use different names for equivalent tools:

| Generic | Claude | Codex | Gemini | Goose |
|---------|--------|-------|--------|-------|
| Read | Read | read | read | read |
| Write | Write | write | write | write |
| Bash | Bash | bash | bash | shell |
| Grep | Grep | grep | search | search |
| Glob | Glob | glob | find | find |

**Decision:** Use Claude-style names as the canonical format. Translate as needed per agent.

## 4. Security Considerations

### 4.1 Default Behavior

If `allowed_tools` is not configured, use agent defaults (permissive). Operators decide their own security posture.

### 4.2 Bash Restrictions

The `Bash` tool is especially sensitive. Consider:
- Allowing Bash with pattern restrictions (e.g., only `kubectl` commands)
- This is complex and agent-specific; defer to future work

### 4.3 Audit Trail

Capture allowed tools in `prompt-sent.md` metadata:
```markdown
## Metadata
- Allowed Tools: Read,Grep,Glob,Bash
- Tool Restrictions Enforced: true
```

## 5. Implementation Sequence

```
Phase 1: Data Flow
├── JobConfig.AllowedTools field
├── ALLOWED_TOOLS env var in Job spec
└── k8s_executor passes config value

Phase 2: Claude (primary agent)
├── Read ALLOWED_TOOLS in entrypoint
├── Replace hardcoded --allowedTools
└── Test with restricted tools

Phase 3: Other Agents
├── Research each agent's mechanism
├── Implement translation
└── Document any limitations
```

Note: Tool restriction verification will be handled by the release test suite, not core runtime.

## 6. Files Changed

| File | Change |
|------|--------|
| `internal/agent/k8s/job.go` | Add `AllowedTools` to JobConfig, env var in spec |
| `internal/agent/k8s_executor.go` | Pass AllowedTools from config |
| `nc-agent-runner/entrypoint.sh` | Add translation, update run_agent() |
| `configs/config.example.yaml` | Document per-agent format |
| `docs/configuration.md` | Security best practices |

## 7. Open Questions

1. Should we validate tool names against a known list?
2. How to handle agent version differences in tool support?
3. Should we support regex patterns for Bash restrictions?
