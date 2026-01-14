.PHONY: build clean help build-agent-image dev-cluster dev-teardown dev-secrets \
        docker-build docker-push docker-release

# Version information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME ?= $(shell date -u '+%Y-%m-%d_%H:%M:%S_UTC')
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Build variables
BINARY_NAME = nightcrier
BUILD_DIR = bin
MAIN_PATH = ./cmd/nightcrier

# Agent image variables
AGENT_IMAGE_NAME = nc-agent-runner
AGENT_IMAGE_VERSION ?= $(VERSION)

# Linker flags for version injection
LDFLAGS = -X main.Version=$(VERSION) \
          -X main.BuildTime=$(BUILD_TIME) \
          -X main.GitCommit=$(GIT_COMMIT)

build: ## Build the nightcrier binary
	@mkdir -p $(BUILD_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "Built $(BUILD_DIR)/$(BINARY_NAME) [$(VERSION)]"

clean: ## Remove build artifacts
	rm -rf $(BUILD_DIR)

build-agent-image: ## Build the nc-agent-runner container image
	@./scripts/build-agent-image.sh $(AGENT_IMAGE_VERSION)

# Development cluster targets
dev-cluster: ## Create kind cluster and load agent image
	@./scripts/dev-setup.sh

dev-teardown: ## Delete kind cluster
	@./scripts/dev-teardown.sh

dev-secrets: ## Update API key secrets in kind cluster
	@echo "Updating agent-api-keys secret in nightcrier namespace..."
	@kubectl delete secret agent-api-keys -n nightcrier --ignore-not-found
	@kubectl create secret generic agent-api-keys -n nightcrier \
		--from-literal=ANTHROPIC_API_KEY="$${ANTHROPIC_API_KEY:-}" \
		--from-literal=OPENAI_API_KEY="$${OPENAI_API_KEY:-}" \
		--from-literal=GEMINI_API_KEY="$${GEMINI_API_KEY:-}"
	@echo "Secret updated. Set env vars before running: ANTHROPIC_API_KEY, OPENAI_API_KEY, GEMINI_API_KEY"

# Docker targets for nightcrier controller image
docker-build: ## Build nightcrier Docker image (single-arch)
	$(MAKE) -C docker/nightcrier build

docker-push: ## Build and push nightcrier multi-arch image to registry
	$(MAKE) -C docker/nightcrier buildx-push

docker-release: ## Full release: setup buildx, build multi-arch, push
	$(MAKE) -C docker/nightcrier release

help: ## Show this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'
