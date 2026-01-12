# Makefile for nc-agent-runner (K8s-native stateless agent container)
#
# Multi-arch build support for amd64 and arm64
# Registry push support for ghcr.io and docker.io

# Image configuration
IMAGE_NAME ?= nc-agent-runner
IMAGE_TAG ?= latest
REGISTRY ?= ghcr.io/randybias
FULL_IMAGE = $(IMAGE_NAME):$(IMAGE_TAG)
REGISTRY_IMAGE = $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG)

# Build context must be repo root for COPY paths to work
BUILD_CONTEXT = ..

# Platforms for multi-arch builds
PLATFORMS ?= linux/amd64,linux/arm64

# Kind cluster name (must match scripts/dev-setup.sh)
KIND_CLUSTER ?= nightcrier-dev

# Git version information (for tagging)
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GIT_TAG ?= $(shell git describe --tags --exact-match 2>/dev/null || echo "")
VERSION ?= $(if $(GIT_TAG),$(GIT_TAG),v0.2.0-dev)

# =============================================================================
# Build Targets
# =============================================================================

.PHONY: build
build:
	@echo "Building single-arch image: $(FULL_IMAGE)"
	docker build -t $(FULL_IMAGE) \
		-f Dockerfile \
		$(BUILD_CONTEXT)
	@echo ""
	@echo "Image built: $(FULL_IMAGE)"

.PHONY: build-clean
build-clean:
	@echo "Building single-arch image (no cache): $(FULL_IMAGE)"
	docker build --no-cache --pull -t $(FULL_IMAGE) \
		-f Dockerfile \
		$(BUILD_CONTEXT)
	@echo ""
	@echo "Image built: $(FULL_IMAGE)"

.PHONY: buildx
buildx:
	@echo "Building multi-arch image: $(FULL_IMAGE)"
	@echo "Platforms: $(PLATFORMS)"
	docker buildx build --platform $(PLATFORMS) \
		-t $(FULL_IMAGE) \
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
		echo "Run 'make dev-cluster' from project root, or 'make -C .. dev-cluster' from here"; \
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
		-v $(HOME)/.kube/config:/home/agent/.kube/config:ro \
		--network host \
		$(FULL_IMAGE)

.PHONY: test-tools
test-tools:
	@echo "Testing tool availability in container"
	docker run --rm --entrypoint /bin/bash $(FULL_IMAGE) -c " \
		echo '=== Tool Versions ===' && \
		kubectl version --client --output=json 2>/dev/null | jq -r '.clientVersion.gitVersion' || echo 'kubectl: error' && \
		helm version --short 2>/dev/null || echo 'helm: error' && \
		rg --version 2>/dev/null | head -1 || echo 'ripgrep: error' && \
		fd --version 2>/dev/null || echo 'fd: error' && \
		jq --version 2>/dev/null || echo 'jq: error' && \
		sqlite3 --version 2>/dev/null || echo 'sqlite3: error' && \
		python3 --version 2>/dev/null || echo 'python3: error' && \
		echo '' && \
		echo '=== AI CLI Tools ===' && \
		claude --version 2>/dev/null || echo 'claude: not available (install may have failed)' && \
		codex --version 2>/dev/null || echo 'codex: not available (install may have failed)' && \
		gemini --version 2>/dev/null || echo 'gemini: not available (install may have failed)' && \
		goose --version 2>/dev/null || echo 'goose: installed (version check requires X11)' && \
		echo '' && \
		echo '=== Directory Structure ===' && \
		ls -la /home/agent/ && \
		echo '' && \
		echo '=== Skills Directory ===' && \
		ls -la /home/agent/skills/ 2>/dev/null || echo 'skills: not found' && \
		echo '' && \
		echo '=== Entrypoint ===' && \
		test -x /home/agent/entrypoint.sh && echo 'entrypoint.sh: executable' || echo 'entrypoint.sh: not found or not executable' \
	"

.PHONY: test-kubectl
test-kubectl:
	@echo "Testing kubectl access"
	docker run --rm --entrypoint kubectl \
		-v $(HOME)/.kube/config:/home/agent/.kube/config:ro \
		--network host \
		$(FULL_IMAGE) version --client

.PHONY: test-entrypoint
test-entrypoint:
	@echo "Testing entrypoint script syntax"
	@shellcheck nc-agent-runner/entrypoint.sh || echo "shellcheck not available, skipping"
	@echo ""
	@echo "Testing entrypoint in container (dry run)"
	docker run --rm --entrypoint /bin/bash $(FULL_IMAGE) -c "/home/agent/entrypoint.sh --help 2>&1 || true"

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
	@command -v shellcheck >/dev/null 2>&1 || echo "Warning: shellcheck not found (optional)"
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
	@echo "nc-agent-runner Build System"
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
	@echo "  load-kind       - Load image into kind cluster 'KIND_CLUSTER=<cluster-name>'"
	@echo ""
	@echo "Testing:"
	@echo "  test-tools      - Verify all tools installed correctly"
	@echo "  test-kubectl    - Verify kubectl works"
	@echo "  test-entrypoint - Test entrypoint script"
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
	@echo "  IMAGE_NAME      - Image name (default: nc-agent-runner)"
	@echo "  IMAGE_TAG       - Image tag (default: latest)"
	@echo "  REGISTRY        - Registry URL (default: ghcr.io/randybias)"
	@echo "  VERSION         - Version tag (default: auto from git or v0.2.0-dev)"
	@echo "  PLATFORMS       - Build platforms (default: linux/amd64,linux/arm64)"
	@echo ""
	@echo "Examples:"
	@echo "  make build                              # Build for local testing"
	@echo "  make dev                                # Build and load into kind"
	@echo "  make release                            # Build multi-arch and push"
	@echo "  make buildx-push REGISTRY=docker.io/me  # Push to custom registry"
	@echo "  make tag VERSION=v1.0.0                 # Tag with specific version"

.DEFAULT_GOAL := help
