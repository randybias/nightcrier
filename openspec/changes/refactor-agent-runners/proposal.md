# Change: Refactor Agent Runners into Modular Sub-Scripts

## Why

The current `run-agent.sh` script mixes agent-agnostic orchestration logic with agent-specific implementation details (CLI arguments, authentication, session extraction, debugging). This leads to:
- Complex `case` statements that grow with each new agent
- Claude-specific post-run hooks that fail for other agents (e.g., `.claude` directory extraction fails for Codex)
- Difficult troubleshooting when agent-specific logging/debug paths differ
- Lost artifacts when agents have different session/log structures

Each AI CLI (Claude, Codex, Gemini, Goose) has fundamentally different:
- Authentication mechanisms (API key handling, login commands)
- CLI arguments and flags
- Verbose/debug modes
- Session data locations (`.claude/`, `.codex/`, etc.)
- Log file formats and extraction methods

## What Changes

- **Refactor** `run-agent.sh` into a thin orchestrator that handles:
  - Environment initialization
  - Docker container setup (common across all agents)
  - Workspace/output directory preparation
  - Pre-run validation
  - Agent-specific sub-script invocation
  - Post-run hook dispatch

- **Create** agent-specific sub-runners:
  - `runners/claude.sh` - Claude Code CLI specifics
  - `runners/codex.sh` - OpenAI Codex CLI specifics
  - `runners/gemini.sh` - Google Gemini CLI specifics
  - `runners/goose.sh` - Goose CLI specifics

- **Create** agent-specific post-run hooks:
  - `runners/claude-post.sh` - Extract `.claude/` session, JSONL commands
  - `runners/codex-post.sh` - Extract `.codex/` session, Codex-specific logs
  - `runners/gemini-post.sh` - Extract Gemini session artifacts
  - `runners/goose-post.sh` - Extract Goose session artifacts

- **Standardize** outputs: Each sub-runner produces:
  - `agent-commands-executed.log` (format may vary per agent)
  - `agent-session.tar.gz` (agent-specific session archive)
  - Standard exit codes and error handling

## Impact

- Affected specs: `agent-container`, `agent-logging`
- Affected code:
  - `agent-container/run-agent.sh` - Complete refactor
  - `agent-container/runners/` - New directory with sub-scripts
  - `agent-container/Dockerfile` - May need updates for runner scripts
