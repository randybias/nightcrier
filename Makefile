.PHONY: build clean help

# Version information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME ?= $(shell date -u '+%Y-%m-%d_%H:%M:%S_UTC')
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Build variables
BINARY_NAME = nightcrier
BUILD_DIR = bin
MAIN_PATH = ./cmd/nightcrier

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

help: ## Show this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'
