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

# Create workspace, skills, and output directories
RUN mkdir -p /workspace /skills /output && chmod 777 /workspace /skills /output

# Clone k8s4agents skills from GitHub
# These provide Kubernetes troubleshooting capabilities for the AI agents
RUN git clone --depth 1 https://github.com/randybias/k8s4agents.git /tmp/k8s4agents \
    && cp -r /tmp/k8s4agents/skills/* /skills/ \
    && chmod -R 755 /skills \
    && rm -rf /tmp/k8s4agents

# Setup Claude Code user skills directory
# Skills are available at /skills/k8s-troubleshooter/SKILL.md
RUN mkdir -p /root/.claude/commands \
    && echo "Read /skills/k8s-troubleshooter/SKILL.md and follow its instructions for Kubernetes troubleshooting." > /root/.claude/commands/k8s-troubleshooter.md

# Set working directory
WORKDIR /workspace

# No default entrypoint - run-agent.sh will call the appropriate CLI
ENTRYPOINT ["/bin/bash", "-c"]
CMD ["echo 'Use run-agent.sh to invoke this container'"]

# Labels
LABEL maintainer="kubernetes-mcp-alerts-event-runner"
LABEL description="Triage agent container with kubectl, helm, and multiple AI CLIs (Claude, Codex, Goose, Gemini) for Kubernetes incident investigation"
