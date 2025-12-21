# Design: Modular Agent Runner Architecture

## Context

The current `run-agent.sh` is a 773-line monolithic script that tries to handle four different AI CLIs (Claude, Codex, Gemini, Goose) in a single file. This has become problematic as:

1. **Different authentication patterns**: Claude uses `ANTHROPIC_API_KEY` directly; Codex requires explicit `codex login --with-api-key`; Gemini uses `GEMINI_API_KEY` or `GOOGLE_API_KEY`
2. **Different CLI invocations**: Each has unique flags, model mappings, and execution patterns
3. **Different session structures**: Claude stores sessions in `.claude/projects/*.jsonl`; Codex uses `.codex/`; others have their own patterns
4. **Post-run extraction hardcoded for Claude**: The `post_run_extract_claude_session` function assumes Claude, causing failures for Codex

## Goals / Non-Goals

**Goals:**
- Clean separation of agent-agnostic orchestration from agent-specific logic
- Each agent's peculiarities isolated in its own files
- Standardized interface for post-run artifact extraction
- No loss of debugging/logging capabilities during refactor
- Easy addition of new agents in the future

**Non-Goals:**
- Changing the Docker container architecture
- Modifying the nightcrier Go code that invokes run-agent.sh
- Adding new agent support (just refactoring existing)

## Decisions

### Decision: Sub-Runner Architecture

Each agent gets two files in `agent-container/runners/`:
- `{agent}.sh` - Builds the CLI command and handles agent-specific setup
- `{agent}-post.sh` - Extracts session artifacts after execution

**Why**: This keeps agent logic isolated while allowing shared orchestration. Alternative considered: single file per agent with both pre and post logic, rejected because it mixes concerns and makes the main script harder to reason about.

### Decision: Standardized Environment Contract

Sub-runners receive standardized environment variables:
```bash
# From main orchestrator (read-only for sub-runners)
AGENT_CLI          # Agent name (claude, codex, gemini, goose)
AGENT_HOME         # /home/agent in container
PROMPT             # The investigation prompt
LLM_MODEL          # Model to use
AGENT_VERBOSE      # true/false
SYSTEM_PROMPT_FILE # Path to system prompt if provided
OUTPUT_FILE        # Target output filename
WORKSPACE_DIR      # Host workspace directory
INCIDENT_ID        # Incident identifier

# API keys (agent-specific)
ANTHROPIC_API_KEY  # For Claude
OPENAI_API_KEY     # For Codex
GEMINI_API_KEY     # For Gemini
GOOGLE_API_KEY     # For Gemini (alternate)
```

**Why**: Explicit contract makes sub-runners predictable and testable.

### Decision: Sub-Runner Returns Command String

Each `{agent}.sh` outputs the command string to stdout. The main orchestrator captures it and executes via Docker.

```bash
# In run-agent.sh
AGENT_CMD=$(source "runners/${AGENT_CLI}.sh")
docker "${DOCKER_ARGS[@]}" "$AGENT_CMD"
```

**Why**: Keeps execution in the main script for consistent error handling and logging.

### Decision: Post-Run Hooks Are Agent-Agnostic

The main script calls a dispatcher that routes to agent-specific post hooks:

```bash
# In run-agent.sh
run_post_hooks() {
    local post_script="runners/${AGENT_CLI}-post.sh"
    if [[ -f "$post_script" ]]; then
        source "$post_script"
    fi
}
```

**Why**: Graceful fallback if an agent doesn't have post-run requirements.

### Decision: Standardized Output Artifacts

All agents produce (when DEBUG mode enabled):
- `logs/agent-session.tar.gz` - Archive of agent's session directory
- `logs/agent-commands-executed.log` - Extracted commands in standard format

Format for `agent-commands-executed.log`:
```
# Agent Commands Executed
# Agent: {agent_cli}
# Generated: {timestamp}
# Incident: {incident_id}
#
$ command1 # description if available
$ command2
```

**Why**: Downstream tooling (storage upload, HTML report) expects consistent paths.

## Directory Structure After Refactor

```
agent-container/
├── run-agent.sh              # Thin orchestrator (~300 lines)
├── runners/
│   ├── common.sh             # Shared functions (validation, logging)
│   ├── claude.sh             # Claude CLI command builder
│   ├── claude-post.sh        # Claude session extraction
│   ├── codex.sh              # Codex CLI command builder
│   ├── codex-post.sh         # Codex session extraction
│   ├── gemini.sh             # Gemini CLI command builder
│   ├── gemini-post.sh        # Gemini session extraction
│   ├── goose.sh              # Goose CLI command builder
│   └── goose-post.sh         # Goose session extraction
├── Dockerfile
├── Makefile
└── README.md
```

## Risks / Trade-offs

| Risk | Mitigation |
|------|------------|
| Breaking existing Claude investigations | Test Claude path first; keep old script as backup |
| Codex JSONL format differs from Claude | Both use JSONL; adapt jq query for Codex event structure |
| Gemini uses JSON not JSONL | Parse `logs.json` differently; may need different jq approach |
| Goose uses SQLite not files | Use `sqlite3` CLI for command extraction; more complex |
| Shell script complexity | Keep each sub-runner under 100 lines; thorough comments |

## Migration Plan

1. Create `runners/` directory with new scripts
2. Test each sub-runner in isolation
3. Refactor `run-agent.sh` to use sub-runners
4. Verify Claude path works identically to before
5. Fix Codex post-run extraction
6. Stub Gemini/Goose post-run (minimal logging)
7. Remove legacy code from main script

## Agent Session Storage Details (Research Findings)

### 1. Codex Session Storage

**Location**: `~/.codex/sessions/` (JSONL files)

Codex stores sessions as JSONL files in `$CODEX_HOME/sessions/` (typically `~/.codex/sessions/`). Each session file contains:
- A metadata header line with session info (ID, source, timestamp, model provider)
- Subsequent lines representing session events

Additional locations:
- `~/.codex/config.toml` - Configuration
- `~/.codex/log/codex-tui.log` - TUI logs
- `~/.codex/AGENTS.md` - Global instructions

**Extraction approach**: Similar to Claude - extract `~/.codex/` directory and parse JSONL for commands.

### 2. Gemini CLI Session Storage

**Location**: `~/.gemini/tmp/<hash>/chats/session-*.json`

Gemini CLI automatically saves sessions in hashed folders under `~/.gemini/tmp/`. Each session includes:
- Complete conversation history in `chats/session-<timestamp>-<id>.json`
- Messages array with user/gemini interactions
- Tool use and bash command execution logs
- Sessions are project-specific (context switches when changing directories)

**Extraction approach**: Extract `~/.gemini/` directory. Parse `session-*.json` files (JSON format, not JSONL) for bash commands.

**Context file integration**: Gemini CLI uses `GEMINI.md` files for persistent context:
- Hierarchical loading: `~/.gemini/GEMINI.md` (global), project root, subdirectories
- Automatically loaded as system prompt
- Used to provide k8s-troubleshooter skill reference without native skills system

### 3. Goose Session Storage

**Location**: `~/.config/goose/sessions.db` (SQLite database)

As of Goose v1.10.0, sessions are stored in a SQLite database rather than individual .jsonl files. Configuration lives in `~/.config/goose/config.yaml`.

**Extraction approach**: Extract `~/.config/goose/` directory. Command extraction would require SQLite queries rather than JSONL parsing - more complex than other agents.

**Provider consideration**: Goose works with any LLM and supports multi-model configuration. The session storage format is provider-agnostic (same SQLite structure regardless of which LLM backend is used), so no provider-specific post-hooks are needed.

## Implementation Details

### Bug Fixes Applied

1. **INCIDENT_ID unbound variable**: Changed to `${INCIDENT_ID:-}` pattern for bash strict mode compatibility
2. **Docker ENTRYPOINT understanding**: Removed redundant `bash -c` since image already has `ENTRYPOINT ["/bin/bash", "-c"]`
3. **AGENT_IMAGE default**: Set to `nightcrier-agent:latest` (was unset, causing Docker errors)
4. **Debug command building**: Changed to single-line output to avoid bash -c parsing issues with multi-line strings

### Testing Results

**Claude**:
- ✓ Basic execution working
- ✓ DEBUG mode session extraction: `.claude/projects/*.jsonl`
- ✓ Command extraction from JSONL working

**Codex**:
- ✓ Login flow with `codex login --with-api-key` working
- ✓ Model mapping: `opus→gpt-5-codex`, `sonnet→gpt-5.2`, `haiku→gpt-4o`
- ✓ DEBUG mode session extraction: `.codex/sessions/*.jsonl`
- ✓ Command extraction from JSONL working

**Gemini**:
- ✓ GEMINI.md context file created in Dockerfile
- ✓ Gemini loads context and knows about k8s-troubleshooter skill
- ✓ DEBUG mode session extraction: `.gemini/tmp/*/chats/session-*.json`
- ✓ Session file path fixed from `logs.json` to `session-*.json`
- ✓ Variable scope fixed (removed invalid `local` outside function)

**Test Suite Created**:
- 6 isolated test scripts (45+ test cases total)
- `test_common.sh`: Utility functions
- `test_claude_commands.sh`: Command generation
- `test_orchestrator.sh`: Dispatch mechanism
- `test_env_propagation.sh`: Environment variables
- `test_error_handling.sh`: Edge cases
- `test_docker_args.sh`: Docker arguments validation

## Sources

- [OpenAI Codex CLI Memory Deep Dive](https://mer.vin/2025/12/openai-codex-cli-memory-deep-dive/)
- [Codex CLI Configuration](https://github.com/openai/codex/blob/main/docs/config.md)
- [Gemini CLI Session Management](https://developers.googleblog.com/pick-up-exactly-where-you-left-off-with-session-management-in-gemini-cli/)
- [Gemini CLI GitHub](https://github.com/google-gemini/gemini-cli)
- [Practical Gemini CLI: Instruction Following — System Prompts and Context](https://medium.com/google-cloud/practical-gemini-cli-instruction-following-system-prompts-and-context-d3c26bed51b6)
- [Gemini CLI Configuration](https://github.com/google-gemini/gemini-cli/blob/main/docs/cli/configuration.md)
- [Goose CLI Commands](https://block.github.io/goose/docs/guides/goose-cli-commands/)
- [Goose Provider Configuration](https://deepwiki.com/block/goose/2.2-provider-configuration)
