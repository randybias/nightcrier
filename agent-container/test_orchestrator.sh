#!/usr/bin/env bash
set -euo pipefail

echo "=== Testing Orchestrator Dispatch ==="

# Create a mock Docker command that just echoes
mkdir -p scratch
cat > scratch/mock-docker << 'EOFDOCKER'
#!/usr/bin/env bash
echo "MOCK DOCKER CALLED"
echo "Args: $*"
exit 0
EOFDOCKER
chmod +x scratch/mock-docker
export PATH="$(pwd)/scratch:$PATH"

echo ""
echo "Test 1: Dispatch to claude runner"
export AGENT_IMAGE="test-image:latest"
export AGENT_CLI="claude"
export AGENT_HOME="/home/agent"
export LLM_MODEL="sonnet"
export AGENT_ALLOWED_TOOLS="Read,Grep,Bash"
export CONTAINER_TIMEOUT="600"
export WORKSPACE_DIR="$(pwd)/scratch/test-workspace"
export KUBECONFIG_PATH="$HOME/.kube/config"
export OUTPUT_DIR="$(pwd)/scratch/test-workspace/output"
export OUTPUT_FILE="test.log"
export ANTHROPIC_API_KEY="test-key"
export DEBUG="false"
export AGENT_VERBOSE="false"
export PROMPT="Test investigation"

mkdir -p "$WORKSPACE_DIR"
mkdir -p "$OUTPUT_DIR"
echo '{}' > "$WORKSPACE_DIR/incident.json"

# Extract just the command building part by sourcing and calling the function
# We can't run the full script without Docker, so let's test the dispatch logic
cd /Users/rbias/code/worktrees/feature/refactor-agent-runners/agent-container

# Source common.sh and test dispatch
source runners/common.sh

# Test that the sub-runner file exists and can be called
AGENT_RUNNER="runners/${AGENT_CLI}.sh"
if [[ -f "$AGENT_RUNNER" ]]; then
    echo "✓ Claude runner exists at $AGENT_RUNNER"
else
    echo "✗ FAILED: Claude runner not found"
    exit 1
fi

# Test command generation through runner
AGENT_CMD=$(bash "$AGENT_RUNNER")
if [[ -n "$AGENT_CMD" ]] && [[ "$AGENT_CMD" =~ "claude" ]]; then
    echo "✓ Claude runner generates command: ${AGENT_CMD:0:50}..."
else
    echo "✗ FAILED: Claude runner didn't generate valid command"
    echo "Got: $AGENT_CMD"
    exit 1
fi

echo ""
echo "Test 2: Dispatch to codex runner"
export AGENT_CLI="codex"
export OPENAI_API_KEY="test-key"
unset ANTHROPIC_API_KEY

AGENT_RUNNER="runners/${AGENT_CLI}.sh"
if [[ -f "$AGENT_RUNNER" ]]; then
    echo "✓ Codex runner exists at $AGENT_RUNNER"
else
    echo "✗ FAILED: Codex runner not found"
    exit 1
fi

AGENT_CMD=$(bash "$AGENT_RUNNER")
if [[ -n "$AGENT_CMD" ]] && [[ "$AGENT_CMD" =~ "codex" ]]; then
    echo "✓ Codex runner generates command: ${AGENT_CMD:0:50}..."
else
    echo "✗ FAILED: Codex runner didn't generate valid command"
    echo "Got: $AGENT_CMD"
    exit 1
fi

echo ""
echo "Test 3: Dispatch to gemini runner"
export AGENT_CLI="gemini"
export GEMINI_API_KEY="test-key"
unset OPENAI_API_KEY

AGENT_RUNNER="runners/${AGENT_CLI}.sh"
if [[ -f "$AGENT_RUNNER" ]]; then
    echo "✓ Gemini runner exists at $AGENT_RUNNER"
else
    echo "✗ FAILED: Gemini runner not found"
    exit 1
fi

AGENT_CMD=$(bash "$AGENT_RUNNER")
if [[ -n "$AGENT_CMD" ]] && [[ "$AGENT_CMD" =~ "gemini" ]]; then
    echo "✓ Gemini runner generates command: ${AGENT_CMD:0:50}..."
else
    echo "✗ FAILED: Gemini runner didn't generate valid command"
    echo "Got: $AGENT_CMD"
    exit 1
fi

echo ""
echo "Test 4: Invalid agent name"
export AGENT_CLI="invalid-agent"
export ANTHROPIC_API_KEY="test-key"

AGENT_RUNNER="runners/${AGENT_CLI}.sh"
if [[ ! -f "$AGENT_RUNNER" ]]; then
    echo "✓ Correctly detects missing runner for invalid agent"
else
    echo "✗ FAILED: Should not have found runner for invalid agent"
    exit 1
fi

# Cleanup
rm -rf scratch/test-workspace scratch/mock-docker

echo ""
echo "=== All Orchestrator Tests PASSED ==="
