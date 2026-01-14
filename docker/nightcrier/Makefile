# Makefile for nightcrier controller Docker image
#
# Multi-arch build support for amd64 and arm64
# Registry push support for any registry (ghcr.io, docker.io, gcr.io, ECR, etc.)
#
# Examples:
#   make build                                    # Build for local testing
#   make buildx-push                              # Push to default registry (ghcr.io)
#   make buildx-push REGISTRY=docker.io/myuser   # Push to DockerHub
#   make buildx-push REGISTRY=gcr.io/my-project  # Push to GCR
#   make release                                  # Full release workflow

# =============================================================================
# Configuration
# =============================================================================

# Image configuration
IMAGE_NAME ?= nightcrier
IMAGE_TAG ?= latest
REGISTRY ?= ghcr.io/randybias
FULL_IMAGE = $(IMAGE_NAME):$(IMAGE_TAG)
REGISTRY_IMAGE = $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG)

# Build context must be repo root for COPY paths to work
BUILD_CONTEXT = ../..

# Platforms for multi-arch builds
PLATFORMS ?= linux/amd64,linux/arm64

# Kind cluster name (must match scripts/dev-setup.sh)
KIND_CLUSTER ?= nightcrier-dev

# Git version information (for tagging)
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GIT_TAG ?= $(shell git describe --tags --exact-match 2>/dev/null || echo "")
VERSION ?= $(if $(GIT_TAG),$(GIT_TAG),$(shell git describe --tags --always 2>/dev/null || echo "dev"))
BUILD_TIME ?= $(shell date -u '+%Y-%m-%d_%H:%M:%S_UTC')

# Build args for version injection
BUILD_ARGS = --build-arg VERSION=$(VERSION) \
             --build-arg BUILD_TIME=$(BUILD_TIME) \
             --build-arg GIT_COMMIT=$(GIT_COMMIT)

# =============================================================================
# Build Targets
# =============================================================================

.PHONY: build
build:
	@echo "Building single-arch image: $(FULL_IMAGE)"
	@echo "Version: $(VERSION)"
	docker build -t $(FULL_IMAGE) \
		$(BUILD_ARGS) \
		-f Dockerfile \
		$(BUILD_CONTEXT)
	@echo ""
	@echo "Image built: $(FULL_IMAGE)"

.PHONY: build-clean
build-clean:
	@echo "Building single-arch image (no cache): $(FULL_IMAGE)"
	docker build --no-cache --pull -t $(FULL_IMAGE) \
		$(BUILD_ARGS) \
		-f Dockerfile \
		$(BUILD_CONTEXT)
	@echo ""
	@echo "Image built: $(FULL_IMAGE)"

.PHONY: buildx
buildx:
	@echo "Building multi-arch image: $(FULL_IMAGE)"
	@echo "Platforms: $(PLATFORMS)"
	@echo "Version: $(VERSION)"
	docker buildx build --platform $(PLATFORMS) \
		-t $(FULL_IMAGE) \
		$(BUILD_ARGS) \
		-f Dockerfile \
		$(BUILD_CONTEXT) \
		--load
	@echo ""
	@echo "Multi-arch image built: $(FULL_IMAGE)"

.PHONY: buildx-push
buildx-push:
	@echo "Building and pushing multi-arch image to registry"
	@echo "Image: $(REGISTRY_IMAGE)"
	@echo "Platforms: $(PLATFORMS)"
	@echo "Version: $(VERSION)"
	docker buildx build --platform $(PLATFORMS) \
		-t $(REGISTRY_IMAGE) \
		-t $(REGISTRY)/$(IMAGE_NAME):$(VERSION) \
		-t $(REGISTRY)/$(IMAGE_NAME):$(GIT_COMMIT) \
		$(BUILD_ARGS) \
		-f Dockerfile \
		$(BUILD_CONTEXT) \
		--push
	@echo ""
	@echo "Images pushed:"
	@echo "  - $(REGISTRY_IMAGE)"
	@echo "  - $(REGISTRY)/$(IMAGE_NAME):$(VERSION)"
	@echo "  - $(REGISTRY)/$(IMAGE_NAME):$(GIT_COMMIT)"

# =============================================================================
# Registry Operations
# =============================================================================

.PHONY: tag
tag:
	@echo "Tagging image for registry"
	docker tag $(FULL_IMAGE) $(REGISTRY_IMAGE)
	docker tag $(FULL_IMAGE) $(REGISTRY)/$(IMAGE_NAME):$(VERSION)
	docker tag $(FULL_IMAGE) $(REGISTRY)/$(IMAGE_NAME):$(GIT_COMMIT)
	@echo ""
	@echo "Tagged as:"
	@echo "  - $(REGISTRY_IMAGE)"
	@echo "  - $(REGISTRY)/$(IMAGE_NAME):$(VERSION)"
	@echo "  - $(REGISTRY)/$(IMAGE_NAME):$(GIT_COMMIT)"

.PHONY: push
push: tag
	@echo "Pushing images to registry"
	docker push $(REGISTRY_IMAGE)
	docker push $(REGISTRY)/$(IMAGE_NAME):$(VERSION)
	docker push $(REGISTRY)/$(IMAGE_NAME):$(GIT_COMMIT)
	@echo ""
	@echo "Images pushed successfully"

.PHONY: pull
pull:
	@echo "Pulling image from registry: $(REGISTRY_IMAGE)"
	docker pull $(REGISTRY_IMAGE)

# =============================================================================
# Kind Operations
# =============================================================================

.PHONY: load-kind
load-kind:
	@echo "Loading image into kind cluster"
	@if ! kind get clusters 2>/dev/null | grep -q "^$(KIND_CLUSTER)$$"; then \
		echo "Error: kind cluster '$(KIND_CLUSTER)' not found"; \
		echo "Run 'make dev-cluster' from project root first"; \
		exit 1; \
	fi
	kind load docker-image $(FULL_IMAGE) --name $(KIND_CLUSTER)
	@echo ""
	@echo "Image loaded into kind cluster $(KIND_CLUSTER)"

# =============================================================================
# Testing and Validation
# =============================================================================

.PHONY: shell
shell:
	@echo "Running interactive shell in container"
	docker run --rm -it --entrypoint /bin/bash \
		-v $(PWD)/../../configs:/configs:ro \
		$(FULL_IMAGE)

.PHONY: test-version
test-version:
	@echo "Testing nightcrier version command"
	docker run --rm $(FULL_IMAGE) --version

.PHONY: test-help
test-help:
	@echo "Testing nightcrier help command"
	docker run --rm $(FULL_IMAGE) --help

.PHONY: test-tools
test-tools:
	@echo "Testing debugging tools in container"
	docker run --rm --entrypoint /bin/bash $(FULL_IMAGE) -c " \
		echo '=== Tool Versions ===' && \
		curl --version 2>/dev/null | head -1 || echo 'curl: error' && \
		jq --version 2>/dev/null || echo 'jq: error' && \
		sqlite3 --version 2>/dev/null || echo 'sqlite3: error' && \
		dig -v 2>&1 | head -1 || echo 'dig: error' && \
		nc -h 2>&1 | head -1 || echo 'nc: available' && \
		echo '' && \
		echo '=== Nightcrier ===' && \
		nightcrier --version && \
		echo '' && \
		echo '=== Migrations ===' && \
		ls -la /migrations/ && \
		echo '' && \
		echo '=== User ===' && \
		whoami && id \
	"

# =============================================================================
# Information and Cleanup
# =============================================================================

.PHONY: info
info:
	@echo "=== Image Information ==="
	@echo "Image name: $(FULL_IMAGE)"
	@echo "Registry: $(REGISTRY)"
	@echo "Version: $(VERSION)"
	@echo "Git commit: $(GIT_COMMIT)"
	@echo "Git tag: $(GIT_TAG)"
	@echo "Build time: $(BUILD_TIME)"
	@echo ""
	@echo "=== Local Images ==="
	@docker images $(IMAGE_NAME) 2>/dev/null || echo "No local images found"
	@echo ""
	@echo "=== Image Details ==="
	@docker inspect $(FULL_IMAGE) --format '{{.Config.Labels}}' 2>/dev/null || echo "Image not built yet"
	@echo ""
	@docker inspect $(FULL_IMAGE) --format 'Size: {{.Size}} bytes' 2>/dev/null || true

.PHONY: clean
clean:
	@echo "Removing local images"
	docker rmi $(FULL_IMAGE) 2>/dev/null || true
	docker rmi $(REGISTRY_IMAGE) 2>/dev/null || true
	@echo "Cleanup complete"

.PHONY: prune
prune:
	@echo "Pruning dangling build cache"
	docker builder prune -f
	@echo "Prune complete"

# =============================================================================
# Setup and Prerequisites
# =============================================================================

.PHONY: setup-buildx
setup-buildx:
	@echo "Setting up Docker buildx for multi-arch builds"
	@if ! docker buildx ls | grep -q multiarch-builder; then \
		docker buildx create --name multiarch-builder --use; \
		docker buildx inspect --bootstrap; \
		echo ""; \
		echo "Buildx builder 'multiarch-builder' created and activated"; \
	else \
		docker buildx use multiarch-builder; \
		echo "Using existing buildx builder 'multiarch-builder'"; \
	fi

.PHONY: check-prereqs
check-prereqs:
	@echo "Checking prerequisites..."
	@command -v docker >/dev/null 2>&1 || (echo "Error: docker not found" && exit 1)
	@command -v docker buildx >/dev/null 2>&1 || (echo "Error: docker buildx not available" && exit 1)
	@command -v git >/dev/null 2>&1 || (echo "Error: git not found" && exit 1)
	@echo "All required prerequisites found"

# =============================================================================
# Development Workflows
# =============================================================================

.PHONY: dev
dev: check-prereqs build load-kind
	@echo ""
	@echo "=== Development Build Complete ==="
	@echo "Image built and loaded into kind cluster"
	@echo "Ready for local testing"

.PHONY: release
release: check-prereqs setup-buildx buildx-push
	@echo ""
	@echo "=== Release Build Complete ==="
	@echo "Multi-arch image built and pushed to registry"

# =============================================================================
# Help
# =============================================================================

.PHONY: help
help:
	@echo "Nightcrier Controller Docker Build System"
	@echo ""
	@echo "Build Targets:"
	@echo "  build           - Build single-arch image for local testing"
	@echo "  build-clean     - Build single-arch without cache"
	@echo "  buildx          - Build multi-arch image (amd64, arm64)"
	@echo "  buildx-push     - Build multi-arch and push to registry"
	@echo ""
	@echo "Registry Operations:"
	@echo "  tag             - Tag local image for registry"
	@echo "  push            - Push tagged images to registry"
	@echo "  pull            - Pull image from registry"
	@echo ""
	@echo "Kind Operations:"
	@echo "  load-kind       - Load image into kind cluster"
	@echo ""
	@echo "Testing:"
	@echo "  test-version    - Verify nightcrier --version"
	@echo "  test-help       - Verify nightcrier --help"
	@echo "  test-tools      - Verify debugging tools installed"
	@echo "  shell           - Interactive shell in container"
	@echo ""
	@echo "Maintenance:"
	@echo "  info            - Show image information"
	@echo "  clean           - Remove local images"
	@echo "  prune           - Prune build cache"
	@echo ""
	@echo "Setup:"
	@echo "  setup-buildx    - Setup multi-arch builder"
	@echo "  check-prereqs   - Check required tools"
	@echo ""
	@echo "Workflows:"
	@echo "  dev             - Build and load into kind (for local dev)"
	@echo "  release         - Build multi-arch and push (for release)"
	@echo ""
	@echo "Variables:"
	@echo "  IMAGE_NAME      - Image name (default: nightcrier)"
	@echo "  IMAGE_TAG       - Image tag (default: latest)"
	@echo "  REGISTRY        - Registry URL (default: ghcr.io/randybias)"
	@echo "  VERSION         - Version tag (default: auto from git)"
	@echo "  PLATFORMS       - Build platforms (default: linux/amd64,linux/arm64)"
	@echo ""
	@echo "Registry Examples:"
	@echo "  make buildx-push                              # ghcr.io (default)"
	@echo "  make buildx-push REGISTRY=docker.io/myuser   # DockerHub"
	@echo "  make buildx-push REGISTRY=gcr.io/my-project  # Google Container Registry"
	@echo "  make buildx-push REGISTRY=123456.dkr.ecr.us-east-1.amazonaws.com  # AWS ECR"

.DEFAULT_GOAL := help
