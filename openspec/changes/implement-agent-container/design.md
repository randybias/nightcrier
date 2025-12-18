# Design: Agent Container

## Context

The event runner needs to invoke AI agents to perform read-only triage of Kubernetes incidents. Rather than requiring each deployment to install and configure multiple AI CLIs, we build a Docker container with all necessary tools pre-installed.

### Background

- Multiple AI CLI tools exist: Claude Code, OpenAI Codex, Google Gemini, Block Goose
- Each has different invocation patterns, authentication methods, and capabilities
- The k8s-troubleshooter skill from k8s4agents provides Kubernetes diagnostic guidance
- Containerization provides isolation and reproducibility

### Constraints

- Must work on both ARM64 (Mac M1/M2) and x86_64 architectures
- Must support headless/non-interactive execution
- Must enforce workspace isolation for security
- Skills must be available to agents without manual setup

## Goals / Non-Goals

### Goals

- Build a Docker container with multiple AI CLI tools
- Support Claude (default), Codex, and Gemini agents
- Include k8s-troubleshooter skill from GitHub
- Provide a unified `run-agent.sh` wrapper script
- Enforce workspace isolation (no access to source code)
- Capture agent output to timestamped log files

### Non-Goals

- Interactive agent sessions (headless only)
- Goose support (blocked by X11 dependency)
- Agent result parsing or interpretation
- Skill updates during runtime

## Decisions

### Decision 1: Debian Base Image

**What:** Use `debian:bookworm-slim` as the base image.

**Why:**
- Smaller than Ubuntu
- Wide package availability
- Good ARM64 support
- Stable and well-maintained

**Alternatives Considered:**
- Alpine: Rejected due to musl libc compatibility issues with some tools
- Ubuntu: Rejected as unnecessarily large

### Decision 2: Multi-Agent CLI Support

**What:** Install Claude Code, OpenAI Codex, and Google Gemini CLIs. Select via `-a/--agent` flag.

**Why:**
- Flexibility to use different AI backends
- Each agent has different strengths
- Allows comparison and fallback options

**Implementation:**
```bash
./run-agent.sh -a claude "prompt"   # Default
./run-agent.sh -a codex "prompt"
./run-agent.sh -a gemini "prompt"
```

**Agent-Specific Invocations:**
- Claude: `claude -p "prompt" --model sonnet --allowedTools ...`
- Codex: `codex exec --skip-git-repo-check "prompt"` (requires login step)
- Gemini: `gemini -p "prompt"`

### Decision 3: Skills Built Into Container

**What:** Clone k8s4agents from GitHub during Docker build and install skills at `/skills/`.

**Why:**
- Skills are always available without manual setup
- Version is pinned at build time
- No need for volume mounts to provide skills

**Implementation:**
```dockerfile
RUN git clone --depth 1 https://github.com/randybias/k8s4agents.git /tmp/k8s4agents \
    && cp -r /tmp/k8s4agents/skills/* /skills/ \
    && rm -rf /tmp/k8s4agents
```

### Decision 4: Required Workspace Flag

**What:** The `-w/--workspace` flag is required. No default to current directory.

**Why:**
- Prevents accidentally mounting source code into container
- Forces explicit isolation of incident data
- Security: container only sees incident-specific files

**Implementation:**
```bash
if [[ -z "$WORKSPACE_DIR" ]]; then
    echo "Error: Workspace directory is required (-w flag)"
    exit 1
fi
```

### Decision 5: Output Capture with Tee

**What:** Pipe agent output through `tee` to capture to timestamped log file while displaying.

**Why:**
- Preserves full agent output for debugging and audit
- Allows real-time viewing of agent progress
- Standardized naming: `triage_<agent>_<timestamp>.log`

**Implementation:**
```bash
cmd+=" 2>&1 | tee /output/${OUTPUT_FILE}"
```

### Decision 6: Codex Login Workaround

**What:** Codex requires explicit login with API key before execution.

**Why:**
- Codex doesn't auto-read `OPENAI_API_KEY` environment variable
- Must use `codex login --with-api-key` with key via stdin

**Implementation:**
```bash
echo -n "${OPENAI_API_KEY}" > /tmp/.codex-key
codex login --with-api-key < /tmp/.codex-key
rm -f /tmp/.codex-key
codex exec --skip-git-repo-check "prompt"
```

### Decision 7: Goose Disabled

**What:** Block Goose CLI is not included (commented out).

**Why:**
- Even the "CLI" binary requires X11 libraries (`libxcb.so.1`)
- Block hasn't provided a true headless build
- Would require installing X11 libs, adding significant image size

**Future:** Re-enable when Block provides headless binary.

## Architecture

```
Host:
  ./incidents/inc-123/          <-- Incident workspace (REQUIRED)
    event.json                  <-- Input data
    context/                    <-- Additional context files
  ./incidents/inc-123/output/   <-- Agent output
    triage_claude_YYYYMMDD_HHMMSS.log

Container:
  /workspace/                   <-- Mounted from incident workspace
  /output/                      <-- Mounted from workspace/output
  /skills/                      <-- Built-in from k8s4agents
    k8s-troubleshooter/
      SKILL.md
      references/
      scripts/
  /root/.kube/config            <-- Mounted read-only from host
  /root/.claude/commands/       <-- Slash command for skill
    k8s-troubleshooter.md
```

## Component Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                      run-agent.sh                            │
│                                                              │
│  Parse flags → Validate → Build docker args → Execute       │
│                                                              │
│  Flags: -a agent, -w workspace, -m model, -t tools, etc.   │
└────────────────────────────┬────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────┐
│                   Docker Container                           │
│                   k8s-triage-agent:latest                   │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  AI CLI Tools                                         │  │
│  │  - claude (Anthropic)                                │  │
│  │  - codex (OpenAI)                                    │  │
│  │  - gemini (Google)                                   │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Kubernetes Tools                                     │  │
│  │  - kubectl 1.31                                      │  │
│  │  - helm 3.x                                          │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Skills                                               │  │
│  │  - /skills/k8s-troubleshooter/                       │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

## Risks / Trade-offs

### Risk: Large Image Size (2.8GB)

**Impact:** Slower pulls, more storage required

**Mitigation:**
- Use slim base image
- Clean apt cache after installs
- Consider multi-stage build for production

**Trade-off:** Accepted for convenience of having all tools available

### Risk: Skills Version Drift

**Impact:** Container may have outdated skills vs. GitHub

**Mitigation:**
- Rebuild container periodically
- Tag containers with build date
- Consider build args for skill version

### Risk: API Key Exposure in Process List

**Impact:** Keys visible via `ps` command

**Mitigation:**
- Keys passed via environment variables (not command line)
- Container is ephemeral
- Codex key written to temp file, deleted immediately

## Configuration Reference

See `agent-container/README.md` for full configuration documentation.

Key environment variables:
- `ANTHROPIC_API_KEY` - Claude authentication
- `OPENAI_API_KEY` - Codex authentication
- `GEMINI_API_KEY` - Gemini authentication
- `CLAUDE_MODEL` - Model selection (default: sonnet)
- `CONTAINER_TIMEOUT` - Execution timeout (default: 600s)
- `CONTAINER_MEMORY` - Memory limit (default: 2g)
