# Proposal: Refactor to K8s-Native Stateless Agent Execution

## Problem Statement

The current agent execution model uses Docker with host volume mounts, creating several issues:

1. **Host coupling**: Volume mounts tie execution to the host filesystem, preventing remote execution
2. **Split orchestration**: Logic is split between Go (config) and Bash (execution), making the system harder to maintain
3. **State dependency**: Post-run extraction via `docker cp` conflicts with `--rm` flag and relies on container persistence
4. **Production deployment**: Docker-based execution doesn't align with production Kubernetes deployments

## Proposed Solution

Refactor the agent execution model to run **exclusively on Kubernetes**, eliminating Docker and achieving fully stateless containers:

- **Inputs**: Delivered via K8s ConfigMaps and Secrets (idiomatic K8s pattern)
- **Outputs**: Uploaded to Object Store by the container before exit (presigned PUT URLs)
- **Containers**: Fully stateless - no volume mounts to host, no host coupling
- **Docker**: Replaced by `kind` for local development (Docker scripts kept for reference only)

### Key Design Decisions

1. **K8s-native input delivery**: Use ConfigMaps for incident data and Secrets for kubeconfig/API keys. No Object Store for inputs - K8s already solves this problem.

2. **Object Store for outputs only**: Container uploads reports, logs, and session archives to Object Store before exit using presigned PUT URLs passed as environment variables.

3. **Unified mount points + entrypoint symlinks**: Single Job spec works for all agent types. The container entrypoint creates agent-specific symlinks for skills.

4. **Go orchestrator creates K8s resources via client-go**: ConfigMaps, Secrets, and Jobs created programmatically - no shell scripts, no `kubectl apply`.

5. **Container naming**: Use `nc-agent-runner` or `nightcrier-agent-runner` for clarity (not generic `nightcrier-agent`).

6. **In-container command extraction**: Post-run extraction of "commands executed" happens inside the container before upload, not via `docker cp`.

## Scope

### In Scope

- New K8s executor using client-go (`internal/agent/k8s_executor.go`)
- Unified container entrypoint (`nc-agent-runner`) with:
  - Agent-specific skill path setup
  - Output upload logic
  - Command extraction per agent type
- ConfigMap/Secret generation for incident data
- Presigned URL generation for output upload
- Local development setup with `kind`
- Container image build scripts and optional GitHub Action for version updates
- Basic real-time logging during agent execution
- Removal of Docker-based execution path (keep for reference)

### Out of Scope

- Advanced log streaming infrastructure
- Multi-cluster Nightcrier deployment (agent kubeconfig provisioning is external)
- Changes to the AI agent CLIs themselves

## Agent Skills Research

Each agent has different skill loading mechanisms:

| Agent | Skills Location | Notes |
|-------|-----------------|-------|
| Claude | `~/.claude/skills/` | Native skill support |
| Codex | `~/.codex/skills/` | Uses `--enable skills` flag |
| Gemini | N/A (uses context files) | Uses `GEMINI.md` at project root |
| Goose | Multiple locations with priority | `~/.claude/skills/`, `~/.config/agents/skills/`, `~/.config/goose/skills/`, `./.goose/skills/`, `./.agents/skills/` |

The entrypoint must set up appropriate symlinks for each agent type.

## Command Extraction Requirements

Extracting "what commands were run" is critical for validation. Each agent stores session data differently:

| Agent | Session Format | Location |
|-------|---------------|----------|
| Claude | JSONL | `~/.claude/projects/*/` |
| Codex | JSONL | `~/.codex/sessions/` |
| Gemini | JSON | `~/.gemini/tmp/*/chats/session-*.json` |
| Goose | SQLite | `~/.config/goose/sessions.db` |

The container entrypoint must extract commands before uploading to Object Store. This replaces the current `*-post.sh` scripts that use `docker cp`.

## Impact

### Capabilities Affected

- **agent-container**: MODIFIED - Remove Docker/shell orchestration, add K8s-native entrypoint
- **k8s-executor**: ADDED - New capability for K8s Job orchestration

### Breaking Changes

- `run-agent.sh` and `runners/*.sh` kept for reference but not executed
- Docker-based local development no longer supported (use `kind` instead)
- Container image renamed to `nc-agent-runner`

### Dependencies

- Existing Object Store integration (cloud-storage spec)
- K8s client-go library
- `sqlite3` available in container image (for Goose session extraction)
- `jq` available in container image (for JSON parsing)

## Container Image Maintenance

The container image requires regular updates to agent CLI versions:
- Claude Code CLI
- OpenAI Codex CLI
- Google Gemini CLI
- Goose CLI
- kubectl

Initial approach: Convenience scripts for manual updates. Future: GitHub Action for automated weekly builds.

## Supersedes

This proposal supersedes `add-k8s-runtime` which proposed a dual Docker/K8s approach with PVCs. The new direction is K8s-only with Object Store for outputs (no PVCs, no parallel Docker operation).

## Success Criteria

1. Agent execution works identically on `kind` (local) and production K8s clusters
2. Zero host volume mounts - container is fully stateless
3. Reports and session archives reliably uploaded to Object Store
4. All agent types (Claude, Codex, Gemini, Goose) work with the new executor
5. Commands executed by each agent are extracted and available in Object Store
6. No Docker daemon required for any Nightcrier operation
7. Basic real-time logging visible during agent execution
