# Nightcrier Agent Container

Docker container for running AI agents to investigate Kubernetes incidents.

## Supported AI CLI Tools

| Agent | Status | Notes |
|-------|--------|-------|
| **Claude** (default) | Working | Anthropic's Claude Code CLI, uses sonnet model by default |
| **Codex** | Working | OpenAI's Codex CLI, requires API key with Codex model access |
| **Gemini** | Working | Google's Gemini CLI |
| **Goose** | Disabled | Block's agent requires X11 libs even in CLI mode |

## Quick Start

```bash
# Build the image
make build

# Create an incident workspace (container is sandboxed to this directory)
mkdir -p ./incidents/test-incident
echo '{"incidentId":"test-001","cluster":"test","namespace":"default","resource":"test-pod","faultType":"CrashLoopBackOff","severity":"critical","context":{},"timestamp":"2025-01-01T00:00:00Z"}' > ./incidents/test-incident/incident.json

# Run an investigation with Claude (default)
export ANTHROPIC_API_KEY="your-key"
./run-agent.sh -w ./incidents/test-incident "Investigate the incident in incident.json"

# Run with Codex
export OPENAI_API_KEY="your-key"
./run-agent.sh -a codex -w ./incidents/test-incident "Analyze the issue"

# Run with Gemini
export GEMINI_API_KEY="your-key"
./run-agent.sh -a gemini -w ./incidents/test-incident "Check cluster health"
```

## Architecture

### Container Structure

```
Host:
  ./incidents/inc-123/          <-- Incident workspace (REQUIRED)
    incident.json               <-- Input data with incident context
    context/                    <-- Additional context files
  ./incidents/inc-123/output/   <-- Agent output
    triage_<agent>_YYYYMMDD_HHMMSS.log

Container (/home/agent):
  /home/agent/                  <-- Agent home AND workspace
    incident.json               <-- Mounted read-only
    output/                     <-- Mounted read-write
    .kube/config                <-- Mounted read-only
    .claude/skills/             <-- Skills (if provided)
    .codex/skills/              <-- Skills (if provided)
```

### Modular Runner Architecture

The agent container uses a modular architecture with agent-specific sub-runners:

```
agent-container/
├── run-agent.sh              # Main orchestrator
│   ├── Environment setup
│   ├── Docker container configuration
│   ├── Dispatches to agent-specific sub-runners
│   └── Post-run artifact extraction
│
└── runners/                  # Agent-specific sub-runners
    ├── common.sh             # Shared utilities (logging, validation, archiving)
    │
    ├── claude.sh             # Claude command builder
    ├── claude-post.sh        # Claude session extraction
    │
    ├── codex.sh              # Codex command builder
    ├── codex-post.sh         # Codex session extraction
    │
    ├── gemini.sh             # Gemini command builder
    └── gemini-post.sh        # Gemini session extraction
```

### How It Works

1. **Main orchestrator** (`run-agent.sh`):
   - Parses arguments and validates configuration
   - Sets up Docker environment and volume mounts
   - Exports standardized environment variables
   - Dispatches to agent-specific command builder
   - Executes Docker container with agent command
   - Dispatches to agent-specific post-run hooks

2. **Agent command builders** (`runners/<agent>.sh`):
   - Source `common.sh` for shared utilities
   - Build agent-specific CLI command string
   - Output command to stdout for orchestrator

3. **Post-run hooks** (`runners/<agent>-post.sh`):
   - Extract session data from container (DEBUG mode only)
   - Parse session files for executed commands
   - Create standardized artifacts:
     - `logs/agent-session.tar.gz` - Session archive
     - `logs/agent-commands-executed.log` - Extracted commands

### Adding a New Agent

To add support for a new AI agent:

1. Create `runners/<agent>.sh`:
   ```bash
   #!/usr/bin/env bash
   # Source common.sh
   SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
   source "$SCRIPT_DIR/common.sh"

   # Validate environment
   validate_runner_env || exit 1

   # Build command
   build_<agent>_command() {
       local cmd=""
       # ... build agent-specific command
       echo "$cmd"
   }

   build_<agent>_command
   ```

2. Create `runners/<agent>-post.sh`:
   ```bash
   #!/usr/bin/env bash
   # Source common.sh
   SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
   source "$SCRIPT_DIR/common.sh"

   # Only run in DEBUG mode
   [[ "$DEBUG" != "true" ]] && exit 0

   # Extract session data
   docker cp "$CONTAINER_NAME:/home/agent/.<agent>" "$WORKSPACE_DIR/<agent>-session-src"

   # Create standardized artifacts
   extract_commands_from_jsonl "$SESSION_FILE" "$WORKSPACE_DIR/logs/agent-commands-executed.log"
   create_archive "$WORKSPACE_DIR/<agent>-session-src" "$WORKSPACE_DIR/logs/agent-session.tar.gz"

   exit 0
   ```

3. Make scripts executable:
   ```bash
   chmod +x runners/<agent>.sh runners/<agent>-post.sh
   ```

4. Test:
   ```bash
   ./run-agent.sh -a <agent> -w ./test-workspace "Test prompt"
   ```

The main orchestrator will automatically discover and use the new sub-runners.

## Environment Variables

### Required (set by nightcrier Go application)

| Variable | Description |
|----------|-------------|
| `AGENT_IMAGE` | Docker image name |
| `AGENT_CLI` | AI CLI to use: claude, codex, gemini, goose |
| `LLM_MODEL` | Model to use (agent-agnostic) |
| `AGENT_ALLOWED_TOOLS` | Comma-separated tool list |
| `CONTAINER_TIMEOUT` | Timeout in seconds |
| `WORKSPACE_DIR` | Host incident directory |
| `KUBECONFIG_PATH` | Path to kubeconfig file |

### Optional

| Variable | Description |
|----------|-------------|
| `ANTHROPIC_API_KEY` | API key for Claude |
| `OPENAI_API_KEY` | API key for Codex |
| `GEMINI_API_KEY` | API key for Gemini |
| `GOOGLE_API_KEY` | Alternate API key for Gemini |
| `OUTPUT_FORMAT` | Output format (agent-specific) |
| `SYSTEM_PROMPT` | Additional system prompt text |
| `SYSTEM_PROMPT_FILE` | Path to system prompt file |
| `AGENT_VERBOSE` | Enable verbose output (true/false) |
| `AGENT_MAX_TURNS` | Maximum conversation turns |
| `CONTAINER_MEMORY` | Memory limit (e.g., "2g") |
| `CONTAINER_CPUS` | CPU limit (e.g., "1.5") |
| `CONTAINER_NETWORK` | Network mode (default: host) |
| `SKILLS_DIR` | Directory containing custom skills |
| `DEBUG` | Enable debug mode (true/false) |
| `INCIDENT_ID` | Incident identifier for container naming |

## DEBUG Mode

When `DEBUG=true`:
- Container is NOT removed after execution (--rm flag omitted)
- Agent session data is extracted from container
- Session archive created at `logs/agent-session.tar.gz`
- Commands log created at `logs/agent-commands-executed.log`
- Verbose debug logging throughout execution

When `DEBUG=false`:
- Container is automatically removed after execution
- No session extraction performed
- Minimal logging

## Agent-Specific Details

### Claude

- **Command builder**: `runners/claude.sh`
- **Session location**: `~/.claude/projects/*.jsonl`
- **Post-run**: Extracts session JSONL and parses Bash tool calls
- **Supports**: model selection, output format, allowed tools, system prompts, verbose mode, max turns

### Codex

- **Command builder**: `runners/codex.sh`
- **Session location**: `~/.codex/sessions/*.jsonl`
- **Post-run**: Extracts session JSONL and parses commands
- **Special**: Requires `codex login --with-api-key` before execution
- **Flags**: `--skip-git-repo-check`, `--enable skills`, `--dangerously-bypass-approvals-and-sandbox`
- **Model mapping**:
  - `opus` → `gpt-5-codex`
  - `sonnet` → `gpt-5.2`
  - `haiku` → `gpt-4o`

### Gemini

- **Command builder**: `runners/gemini.sh`
- **Session location**: `~/.gemini/tmp/<hash>/logs.json`
- **Post-run**: Extracts logs.json (JSON format, not JSONL) and parses commands
- **Supports**: model selection
- **API keys**: Accepts both `GEMINI_API_KEY` and `GOOGLE_API_KEY`

## Testing

Run syntax checks:
```bash
for script in runners/*.sh run-agent.sh; do
    bash -n "$script" && echo "✓ $script" || echo "✗ $script"
done
```

Test command generation:
```bash
export AGENT_CLI="claude"
export AGENT_HOME="/home/agent"
export PROMPT="Test investigation"
export LLM_MODEL="sonnet"
export OUTPUT_FILE="test.log"
export ANTHROPIC_API_KEY="test-key"
bash runners/claude.sh
```

## Troubleshooting

### "Agent runner not found" error

The agent specified doesn't have a runner script. Check `runners/<agent>.sh` exists.

### "Required environment variable not set" error

Missing required environment variables. Ensure nightcrier Go app is setting all required vars.

### Session extraction fails

- Verify DEBUG mode is enabled (`DEBUG=true`)
- Check INCIDENT_ID is set (needed for container naming)
- Ensure container hasn't been removed (`--rm` flag)
- Check agent actually created session data in expected location

### Command extraction produces empty file

- Session format may differ from expected
- jq query may need adjustment for agent's session structure
- Check logs for jq parsing errors
