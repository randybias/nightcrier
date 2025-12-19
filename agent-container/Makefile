# Makefile for nightcrier-agent container

IMAGE_NAME ?= nightcrier-agent
IMAGE_TAG ?= latest
FULL_IMAGE = $(IMAGE_NAME):$(IMAGE_TAG)

# Build the container image
.PHONY: build
build:
	docker build -t $(FULL_IMAGE) .

# Build without cache
.PHONY: build-clean
build-clean:
	docker build --no-cache --pull -t $(FULL_IMAGE) .

# Test workspace (isolated from source code)
TEST_WORKSPACE ?= $(CURDIR)/scratch/test-incident

# Create test workspace if it doesn't exist
.PHONY: test-workspace
test-workspace:
	@mkdir -p $(TEST_WORKSPACE)
	@echo '{"alert": "test", "namespace": "default", "pod": "test-pod"}' > $(TEST_WORKSPACE)/event.json

# Run a test prompt with Claude (requires ANTHROPIC_API_KEY)
.PHONY: test-claude
test-claude: test-workspace
	@if [ -z "$(ANTHROPIC_API_KEY)" ]; then \
		echo "Error: ANTHROPIC_API_KEY not set"; \
		exit 1; \
	fi
	./run-agent.sh -d -a claude -w $(TEST_WORKSPACE) -t "Read,Grep,Bash" "Say hello, list workspace files, and confirm kubectl access"

# Run a test prompt with Codex (requires OPENAI_API_KEY)
.PHONY: test-codex
test-codex: test-workspace
	@if [ -z "$(OPENAI_API_KEY)" ]; then \
		echo "Error: OPENAI_API_KEY not set"; \
		exit 1; \
	fi
	./run-agent.sh -d -a codex -w $(TEST_WORKSPACE) "Say hello"

# Run a test prompt with Gemini (requires GEMINI_API_KEY)
.PHONY: test-gemini
test-gemini: test-workspace
	@if [ -z "$(GEMINI_API_KEY)" ]; then \
		echo "Error: GEMINI_API_KEY not set"; \
		exit 1; \
	fi
	./run-agent.sh -d -a gemini -w $(TEST_WORKSPACE) "Say hello"

# Run interactive shell in container for debugging
.PHONY: shell
shell:
	docker run --rm -it --entrypoint /bin/bash \
		-v $(HOME)/.kube/config:/root/.kube/config:ro \
		--network host \
		$(FULL_IMAGE)

# Check all CLI tools work in container
.PHONY: test-tools
test-tools:
	docker run --rm --entrypoint /bin/bash $(FULL_IMAGE) -c " \
		echo '=== Tool Versions ===' && \
		kubectl version --client --output=json | jq -r '.clientVersion.gitVersion' && \
		helm version --short && \
		rg --version | head -1 && \
		fd --version && \
		python3 --version && \
		gh --version | head -1 && \
		echo '' && \
		echo '=== AI CLI Tools ===' && \
		claude --version 2>/dev/null || echo 'claude: installed' && \
		codex --version 2>/dev/null || echo 'codex: installed' && \
		gemini --version 2>/dev/null || echo 'gemini: installed' && \
		echo '(goose requires X11 libs - skipped)' && \
		echo '' && \
		echo '=== All tools present ===' \
	"

# Check kubectl works in container
.PHONY: test-kubectl
test-kubectl:
	docker run --rm --entrypoint kubectl \
		-v $(HOME)/.kube/config:/root/.kube/config:ro \
		--network host \
		$(FULL_IMAGE) version --client

# Show image info
.PHONY: info
info:
	docker images $(IMAGE_NAME)
	@echo ""
	docker inspect $(FULL_IMAGE) --format '{{.Config.Labels}}' 2>/dev/null || echo "Image not built yet"

# Clean up image
.PHONY: clean
clean:
	docker rmi $(FULL_IMAGE) 2>/dev/null || true

# Show help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build        - Build the container image"
	@echo "  build-clean  - Build without cache"
	@echo "  test-claude  - Run test with Claude (requires ANTHROPIC_API_KEY)"
	@echo "  test-codex   - Run test with Codex (requires OPENAI_API_KEY)"
	@echo "  test-gemini  - Run test with Gemini (requires GEMINI_API_KEY)"
	@echo "  test-tools   - Verify all tools are installed in container"
	@echo "  test-kubectl - Verify kubectl works in container"
	@echo "  shell        - Run interactive shell in container"
	@echo "  info         - Show image information"
	@echo "  clean        - Remove the image"
	@echo ""
	@echo "Variables:"
	@echo "  IMAGE_NAME   - Image name (default: nightcrier-agent)"
	@echo "  IMAGE_TAG    - Image tag (default: latest)"
	@echo ""
	@echo "Example:"
	@echo "  make build"
	@echo "  ANTHROPIC_API_KEY=xxx make test-claude"
