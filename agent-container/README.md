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

```
Host:
  ./incidents/inc-123/          <-- Incident workspace (REQUIRED)
    incident.json               <-- Input data with incident context
    context/                    <-- Additional context files
  ./incidents/inc-123/output/   <-- Agent output
    triage_claude_YYYYMMDD_HHMMSS.log

Container (/workspace):
  /workspace/                   <-- Mounted from incident workspace
  /output/                      <-- Mounted from workspace/output
  /skills/                      <-- Built-in skills from k8s4agents
    k8s-troubleshooter/
  /root/.kube/config            <-- Mounted read-only from host
```

**Security**: The workspace directory is the ONLY host directory the container can access (plus kubeconfig read-only). Do NOT point workspace at source code directories.

## Skill-Driven Investigation

The agent uses a minimal system prompt (`configs/triage-system-prompt.md`) that delegates investigation methodology to the **k8s-troubleshooter** skill. This approach:

1. **Minimal System Prompt**: ~20 lines describing workspace files and output format
2. **Skill-Driven Methodology**: The k8s-troubleshooter skill contains the actual investigation playbook
3. **Structured Workflows**: Skill provides `incident_triage.sh` for systematic root cause analysis

### Investigation Flow

```
1. Agent reads incident.json for fault context
2. Agent reads incident_cluster_permissions.json for access constraints
3. Agent invokes k8s-troubleshooter skill (via /k8s-troubleshooter or incident_triage.sh)
4. Skill guides systematic investigation based on fault type
5. Agent writes findings to output/investigation.md
```

### Built-in Skills

The container includes the [k8s-troubleshooter](https://github.com/randybias/k8s4agents) skill for Kubernetes debugging. Claude can access it via the `/k8s-troubleshooter` slash command or by reading `/skills/k8s-troubleshooter/SKILL.md`.

## Configuration

### Required Parameters

| Parameter | Description |
|-----------|-------------|
| `-w, --workspace DIR` | Incident workspace directory (REQUIRED) |
| Prompt | The investigation prompt (positional argument) |

### Agent Selection

| Flag | Environment | Description |
|------|-------------|-------------|
| `-a, --agent` | `AGENT_CLI` | AI CLI: claude, codex, gemini (default: claude) |

### API Keys

| Variable | Used By | Notes |
|----------|---------|-------|
| `ANTHROPIC_API_KEY` | Claude | Required for Claude |
| `OPENAI_API_KEY` | Codex | Must have Codex model access |
| `GEMINI_API_KEY` | Gemini | Or use `GOOGLE_API_KEY` |

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `AGENT_IMAGE` | `nightcrier-agent:latest` | Docker image to use |
| `WORKSPACE_DIR` | (required) | Host workspace directory |
| `OUTPUT_DIR` | `workspace/output` | Output directory for logs |
| `KUBECONFIG_PATH` | `~/.kube/config` | Path to kubeconfig |
| `KUBERNETES_CONTEXT` | | Kubernetes context to use |
| `CLAUDE_MODEL` | `sonnet` | Claude model (opus, sonnet, haiku) |
| `CLAUDE_OUTPUT_FORMAT` | `text` | Output format |
| `CLAUDE_ALLOWED_TOOLS` | `Read,Grep,Glob,Bash` | Allowed tools |
| `CLAUDE_SYSTEM_PROMPT` | | System prompt text |
| `CLAUDE_VERBOSE` | `false` | Enable verbose output |
| `CLAUDE_MAX_TURNS` | | Max conversation turns |
| `CONTAINER_TIMEOUT` | `600` | Timeout in seconds |
| `CONTAINER_MEMORY` | `2g` | Memory limit |
| `CONTAINER_NETWORK` | `host` | Network mode |

### Command-Line Flags

```
./run-agent.sh --help
```

## Examples

### Basic Investigation with Claude

```bash
./run-agent.sh -w ./incidents/inc-123 "Analyze the pod failure in incident.json and suggest fixes"
```

### Use Different AI Agent

```bash
# Codex
./run-agent.sh -a codex -w ./incidents/inc-123 "Analyze the CrashLoopBackOff issue"

# Gemini
./run-agent.sh -a gemini -w ./incidents/inc-123 "Check why pods are pending"
```

### Read-Only Kubectl

```bash
./run-agent.sh -w ./incidents/inc-123 \
  -t "Read,Grep,Glob,Bash" \
  -s "Only use kubectl get, describe, and logs. Never modify cluster state." \
  "Why is pod my-app in CrashLoopBackOff?"
```

### With Custom Output Directory

```bash
./run-agent.sh -w ./incidents/inc-123 --output-dir ./reports "Investigate incident.json"
```

### Debug Mode

```bash
./run-agent.sh -d -w ./incidents/inc-123 "Your prompt here"
```

### Claude with Opus Model

```bash
./run-agent.sh -m opus -w ./incidents/inc-123 "Deep analysis of the cluster issue"
```

## Output

All agent output is captured and saved to:
- Default: `<workspace>/output/triage_<agent>_<timestamp>.log`
- Custom: Use `--output-dir` and `--output-file` flags

## Included Tools

### Kubernetes
- kubectl 1.31
- helm 3.x

### Search & Navigation
- ripgrep (rg)
- fd
- fzf
- tree

### JSON/YAML
- jq
- yq

### Network Debugging
- dnsutils (dig, nslookup)
- netcat
- ping
- iproute2

### Development
- git
- python3
- make
- GitHub CLI

### Editors
- vim
- neovim

## Integration with Event Runner

The event runner calls this script like:

```bash
./run-agent.sh \
  -a claude \
  -m sonnet \
  -w "$INCIDENT_WORKSPACE" \
  -t "Read,Grep,Glob,Bash,Skill" \
  -s "Investigate the Kubernetes incident. Read incident.json for context." \
  --output-dir "$REPORTS_DIR" \
  "Analyze the incident and provide a root cause analysis"
```

## Makefile Targets

```bash
make build         # Build the container image
make build-clean   # Build without cache
make test-claude   # Test with Claude (needs ANTHROPIC_API_KEY)
make test-codex    # Test with Codex (needs OPENAI_API_KEY)
make test-gemini   # Test with Gemini (needs GEMINI_API_KEY)
make test-tools    # Verify all tools are installed
make test-kubectl  # Verify kubectl works
make test-workspace # Create isolated test workspace
make shell         # Interactive shell in container
make info          # Show image information
make clean         # Remove the image
```

## Troubleshooting

### Codex Authentication

Codex requires explicit login with the API key. The run-agent.sh script handles this automatically, but your OpenAI API key must have access to Codex models (`gpt-5.1-codex-max`). If you see "Quota exceeded", your API key may not have Codex access.

### Goose Disabled

Block's Goose CLI currently requires X11 libraries (`libxcb.so.1`) even in "CLI mode", making it incompatible with headless containers. This will be re-enabled when Block provides a true headless build.

### Workspace Required

The `-w` flag is required to prevent accidentally mounting source code or sensitive directories into the container. Always use an isolated incident directory.
