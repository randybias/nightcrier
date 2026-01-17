# ============================================================================
# Stage 1: Build nats-cli with proper Go version
# ============================================================================
FROM golang:1.24-bookworm AS nats-builder

ARG NATS_CLI_VERSION=v0.3.0

# Build nats-cli from source
RUN go install github.com/nats-io/natscli/nats@${NATS_CLI_VERSION}

# The binary will be in /go/bin/nats


# ============================================================================
# Stage 2: Main agent container
# ============================================================================
FROM debian:bookworm-slim

# Install base dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    # Core utilities
    ca-certificates \
    curl \
    wget \
    gnupg \
    unzip \
    bzip2 \
    tar \
    # Version control
    git \
    # JSON/YAML processing
    jq \
    yq \
    # Database tools
    sqlite3 \
    # Search tools (ripgrep, fd, fzf for agent use)
    ripgrep \
    fd-find \
    fzf \
    # File viewing
    tree \
    bat \
    less \
    # Process monitoring
    htop \
    procps \
    # Network debugging
    dnsutils \
    netcat-openbsd \
    iputils-ping \
    iproute2 \
    # Python for scripting
    python3 \
    python3-pip \
    python3-venv \
    # Build tools (removed golang - not needed in final image)
    make \
    && rm -rf /var/lib/apt/lists/*

# Create symlinks for tools with different names in Debian
RUN ln -s /usr/bin/fdfind /usr/bin/fd && \
    ln -s /usr/bin/batcat /usr/bin/bat

# Install Node.js 20 LTS (required for AI CLI tools)
RUN curl -fsSL https://deb.nodesource.com/setup_20.x | bash - \
    && apt-get install -y nodejs \
    && rm -rf /var/lib/apt/lists/*

# Install kubectl 1.31+
RUN curl -fsSL https://pkgs.k8s.io/core:/stable:/v1.31/deb/Release.key | gpg --dearmor -o /etc/apt/keyrings/kubernetes-apt-keyring.gpg \
    && echo 'deb [signed-by=/etc/apt/keyrings/kubernetes-apt-keyring.gpg] https://pkgs.k8s.io/core:/stable:/v1.31/deb/ /' > /etc/apt/sources.list.d/kubernetes.list \
    && apt-get update \
    && apt-get install -y kubectl \
    && rm -rf /var/lib/apt/lists/*

# Install Helm 3.x
RUN curl -fsSL https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash

# Copy nats-cli binary from builder stage
COPY --from=nats-builder /go/bin/nats /usr/local/bin/nats
RUN chmod +x /usr/local/bin/nats && nats --version

# Create npm global directory with proper permissions
ENV NPM_CONFIG_PREFIX=/home/npm-global
ENV PATH="/home/npm-global/bin:${PATH}"
RUN mkdir -p /home/npm-global && chmod 777 /home/npm-global

# Install AI CLI tools
# Claude Code CLI (Anthropic)
RUN npm install -g @anthropic-ai/claude-code@latest || echo "Warning: Claude Code CLI not available, install manually if needed"

# OpenAI Codex CLI
RUN npm install -g @openai/codex@latest || echo "Warning: Codex CLI not available, install manually if needed"

# Google Gemini CLI
RUN npm install -g @google/gemini-cli@latest || echo "Warning: Gemini CLI not available, install manually if needed"

# Goose CLI - install using official install script to /usr/local/bin
# Use CONFIGURE=false to skip interactive configuration
# Note: GOOSE_DISABLE_KEYRING=1 will be set in entrypoint.sh to use filesystem storage
# instead of system keyring (which requires DBus/X11 not available in containers)
RUN curl -fsSL https://github.com/block/goose/releases/download/stable/download_cli.sh | CONFIGURE=false bash \
    && if [ -f "$HOME/.local/bin/goose" ]; then \
        mv "$HOME/.local/bin/goose" /usr/local/bin/goose && chmod +x /usr/local/bin/goose; \
       fi

# Create non-root user for security
ARG USER_NAME=agent
ARG USER_UID=1000
ARG USER_GID=1000

RUN groupadd --gid ${USER_GID} ${USER_NAME} \
    && useradd --uid ${USER_UID} --gid ${USER_GID} -m ${USER_NAME}

# Create output, logs, and skills directories
# Skills will be cloned at runtime from GitHub (k8s4agents repo)
RUN mkdir -p /home/${USER_NAME}/output /home/${USER_NAME}/logs /home/${USER_NAME}/skills \
    && chown -R ${USER_NAME}:${USER_NAME} /home/${USER_NAME}/output /home/${USER_NAME}/logs /home/${USER_NAME}/skills

# Create Codex managed config for container environment
# This sets safe defaults that work in containerized/headless environments
RUN mkdir -p /etc/codex && \
    cat > /etc/codex/managed_config.toml <<'EOF'
# Managed configuration for Codex in containerized environments
# These settings override user config and CLI flags

[sandbox]
# Bypass sandbox since we're already in an isolated container
mode = "danger-full-access"

[approval]
# Never ask for approval in headless/container environments
policy = "never"

[network]
# Allow network access for kubectl and other tools
access = true
EOF

# Copy entrypoint script, NATS helper, and hooks
COPY --chown=${USER_NAME}:${USER_NAME} nc-agent-runner/entrypoint.sh /home/${USER_NAME}/entrypoint.sh
COPY --chown=${USER_NAME}:${USER_NAME} nc-agent-runner/scripts/nats-publish.sh /home/${USER_NAME}/scripts/nats-publish.sh
COPY --chown=${USER_NAME}:${USER_NAME} nc-agent-runner/hooks/ /home/${USER_NAME}/hooks/
RUN chmod +x /home/${USER_NAME}/entrypoint.sh /home/${USER_NAME}/scripts/nats-publish.sh /home/${USER_NAME}/hooks/*.sh

# Switch to non-root user
USER ${USER_NAME}

# Working directory
WORKDIR /home/${USER_NAME}

# Set entrypoint
ENTRYPOINT ["/home/agent/entrypoint.sh"]

# Labels
LABEL maintainer="nightcrier"
LABEL description="AI agent runner container with kubectl, helm, and multiple AI CLIs (Claude, Codex, Goose, Gemini) for Kubernetes incident investigation"
LABEL version="1.0.0-nats"
