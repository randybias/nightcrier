# Nightcrier Controller Docker Image
# Multi-stage build for minimal runtime image with debugging tools
#
# Runtime Dependencies:
# - ca-certificates: HTTPS connections (MCP servers, cloud storage)
# - sqlite3: Database inspection/debugging (nightcrier uses pure Go sqlite)
# - curl, jq: API debugging and health checks
# - procps: Process monitoring (ps, top)

# =============================================================================
# Stage 1: Build the Go binary
# =============================================================================
FROM golang:1.25-bookworm AS builder

WORKDIR /build

# Copy dependency files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build arguments for version injection
ARG VERSION=dev
ARG BUILD_TIME
ARG GIT_COMMIT

# Build static binary (CGO_ENABLED=0 for pure Go, including modernc.org/sqlite)
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-s -w -X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -X main.GitCommit=${GIT_COMMIT}" \
    -o nightcrier ./cmd/nightcrier

# =============================================================================
# Stage 2: Runtime image with debugging tools
# =============================================================================
FROM debian:bookworm-slim

# Install runtime and debugging tools
RUN apt-get update && apt-get install -y --no-install-recommends \
    # Core - required for HTTPS
    ca-certificates \
    # Debugging tools
    curl \
    jq \
    sqlite3 \
    procps \
    # Network debugging
    netcat-openbsd \
    dnsutils \
    && rm -rf /var/lib/apt/lists/* \
    && apt-get clean

# Create non-root user for security
RUN groupadd -r nightcrier && useradd -r -g nightcrier nightcrier

# Copy binary from builder
COPY --from=builder /build/nightcrier /usr/local/bin/nightcrier

# Copy database migrations
COPY --from=builder /build/migrations /migrations

# Create directories for runtime data
RUN mkdir -p /data /configs && chown -R nightcrier:nightcrier /data /configs

# Switch to non-root user
USER nightcrier

# Default working directory
WORKDIR /data

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD nightcrier --version || exit 1

# Labels for image metadata
LABEL maintainer="nightcrier" \
      description="Nightcrier Kubernetes Incident Triage Controller" \
      org.opencontainers.image.source="https://github.com/randybias/nightcrier"

# Default entrypoint
ENTRYPOINT ["nightcrier"]

# Default command (show help if no args provided)
CMD ["--help"]
