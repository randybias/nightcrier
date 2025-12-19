# Triage Agent container for Kubernetes incident investigation
# Built from scratch with multi-arch support (works on Mac M1/M2 and Linux)
# Supports multiple AI CLI tools: Claude, Codex, Goose, Gemini

FROM debian:bookworm-slim

# Install base dependencies and common agent tools
# Grouped by purpose for clarity
RUN apt-get update && apt-get install -y --no-install-recommends \
    # Core utilities
    ca-certificates \
    curl \
    wget \
    gnupg \
    unzip \
    bzip2 \
    # Version control
    git \
    # Editors
    vim \
    neovim \
    # JSON/YAML processing
    jq \
    yq \
    # Search tools (Claude Code uses ripgrep heavily)
    ripgrep \
    fd-find \
    fzf \
    # File viewing/navigation
    tree \
    bat \
    less \
    # Process monitoring
    htop \
    procps \
    # Network debugging (for k8s network troubleshooting)
    dnsutils \
    netcat-openbsd \
    iputils-ping \
    iproute2 \
    # Python (for scripting)
    python3 \
    python3-pip \
    python3-venv \
    # Build tools (sometimes needed)
    make \
    && rm -rf /var/lib/apt/lists/*

# Create symlinks for tools with different names in Debian
RUN ln -s /usr/bin/fdfind /usr/bin/fd && \
    ln -s /usr/bin/batcat /usr/bin/bat

# Install Node.js 20 LTS
RUN curl -fsSL https://deb.nodesource.com/setup_20.x | bash - \
    && apt-get install -y nodejs \
    && rm -rf /var/lib/apt/lists/*

# Install kubectl
RUN curl -fsSL https://pkgs.k8s.io/core:/stable:/v1.31/deb/Release.key | gpg --dearmor -o /etc/apt/keyrings/kubernetes-apt-keyring.gpg \
    && echo 'deb [signed-by=/etc/apt/keyrings/kubernetes-apt-keyring.gpg] https://pkgs.k8s.io/core:/stable:/v1.31/deb/ /' > /etc/apt/sources.list.d/kubernetes.list \
    && apt-get update \
    && apt-get install -y kubectl \
    && rm -rf /var/lib/apt/lists/*

# Install Helm (for k8s-troubleshooter helm_release_debug.sh)
RUN curl -fsSL https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash

# Install GitHub CLI (useful for PR/issue integration)
RUN curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg | dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg \
    && chmod go+r /usr/share/keyrings/githubcli-archive-keyring.gpg \
    && echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" > /etc/apt/sources.list.d/github-cli.list \
    && apt-get update \
    && apt-get install -y gh \
    && rm -rf /var/lib/apt/lists/*

# Create npm global directory with proper permissions
ENV NPM_CONFIG_PREFIX=/home/npm-global
ENV PATH="/home/npm-global/bin:${PATH}"
RUN mkdir -p /home/npm-global && chmod 777 /home/npm-global

# Install AI CLI tools
# Claude Code CLI (Anthropic)
RUN npm install -g @anthropic-ai/claude-code

# OpenAI Codex CLI
RUN npm install -g @openai/codex

# Google Gemini CLI
RUN npm install -g @google/gemini-cli

# Block Goose CLI - skipped (still requires libxcb.so.1 X11 libs even in "CLI" mode)
# Install command when they fix headless:
# RUN curl -fsSL https://github.com/block/goose/releases/download/stable/download_cli.sh | bash

# Create non-root user for security (matches typical container best practices)
# Using UID 1000 which is common default for first non-root user
ARG USER_NAME=agent
ARG USER_UID=1000
ARG USER_GID=1000

RUN groupadd --gid ${USER_GID} ${USER_NAME} \
    && useradd --uid ${USER_UID} --gid ${USER_GID} -m ${USER_NAME}

# Create output directory in agent home (workspace IS the home directory)
RUN mkdir -p /home/${USER_NAME}/output \
    && chown -R ${USER_NAME}:${USER_NAME} /home/${USER_NAME}/output

# Clone k8s4agents skills from GitHub to user's skills directories
# Install to both ~/.claude/skills/ and ~/.codex/skills/ for CLI compatibility
RUN mkdir -p /home/${USER_NAME}/.claude/skills /home/${USER_NAME}/.claude/commands \
             /home/${USER_NAME}/.codex/skills /home/${USER_NAME}/.kube \
    && git clone --depth 1 https://github.com/randybias/k8s4agents.git /tmp/k8s4agents \
    && cp -r /tmp/k8s4agents/skills/* /home/${USER_NAME}/.claude/skills/ \
    && cp -r /tmp/k8s4agents/skills/* /home/${USER_NAME}/.codex/skills/ \
    && chmod -R 755 /home/${USER_NAME}/.claude/skills /home/${USER_NAME}/.codex/skills \
    && rm -rf /tmp/k8s4agents \
    && chown -R ${USER_NAME}:${USER_NAME} /home/${USER_NAME}/.claude /home/${USER_NAME}/.codex /home/${USER_NAME}/.kube

# Setup Claude Code slash command for k8s-troubleshooter
RUN echo "Read ~/.claude/skills/k8s-troubleshooter/SKILL.md and follow its instructions for Kubernetes troubleshooting." \
    > /home/${USER_NAME}/.claude/commands/k8s-troubleshooter.md \
    && chown ${USER_NAME}:${USER_NAME} /home/${USER_NAME}/.claude/commands/k8s-troubleshooter.md

# Create AGENTS.md for Codex and CLAUDE.md for Claude Code
# Both files provide the same instructions - agent home IS the workspace
RUN cat > /home/${USER_NAME}/AGENTS.md << 'EOF'
# Kubernetes Incident Triage Agent

You are a Kubernetes incident triage agent investigating a production incident.

## Files

- `event.json` - The Kubernetes event that triggered this investigation
- `output/investigation.md` - Write your findings here

## Instructions

1. Read `event.json` to understand the incident
2. Use kubectl to investigate (read-only: get, describe, logs)
3. Use the k8s-troubleshooter skill if needed
4. Write findings to `output/investigation.md`

## Output Format

Write a markdown report with:
- Executive Summary (1-2 sentences)
- Root Cause Analysis
- Evidence (relevant kubectl outputs)
- Recommendations

## Safety

- Read-only kubectl commands only
- Do not modify cluster state
EOF

RUN cp /home/${USER_NAME}/AGENTS.md /home/${USER_NAME}/CLAUDE.md \
    && chown ${USER_NAME}:${USER_NAME} /home/${USER_NAME}/AGENTS.md /home/${USER_NAME}/CLAUDE.md

# Switch to non-root user
USER ${USER_NAME}

# Working directory IS the agent home - everything in one place
WORKDIR /home/${USER_NAME}

# No default entrypoint - run-agent.sh will call the appropriate CLI
ENTRYPOINT ["/bin/bash", "-c"]
CMD ["echo 'Use run-agent.sh to invoke this container'"]

# Labels
LABEL maintainer="kubernetes-mcp-alerts-event-runner"
LABEL description="Triage agent container with kubectl, helm, and multiple AI CLIs (Claude, Codex, Goose, Gemini) for Kubernetes incident investigation"
