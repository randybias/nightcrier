#!/usr/bin/env bash
set -euo pipefail

echo "=== Testing Docker Args Construction ==="
echo "Note: Not running actual Docker, just verifying args structure"

# Create test workspace
mkdir -p scratch/test-incident/output
echo '{}' > scratch/test-incident/incident.json
export WORKSPACE_DIR="$(pwd)/scratch/test-incident"
export OUTPUT_DIR="$WORKSPACE_DIR/output"

# Setup full environment as nightcrier would
export AGENT_IMAGE="nightcrier-agent:latest"
export AGENT_CLI="claude"
export AGENT_HOME="/home/agent"
export LLM_MODEL="sonnet"
export AGENT_ALLOWED_TOOLS="Read,Grep,Glob,Bash"
export OUTPUT_FILE="test.log"
export ANTHROPIC_API_KEY="test-key-123"
export KUBECONFIG_PATH="$HOME/.kube/config"
export KUBERNETES_CONTEXT="test-cluster"
export CONTAINER_TIMEOUT="600"
export CONTAINER_MEMORY="2g"
export CONTAINER_CPUS="1.5"
export CONTAINER_NETWORK="host"
export DEBUG="false"
export PROMPT="Investigate the incident"
export INCIDENT_ID="test-001"

echo ""
echo "Test 1: Verify refactored script builds without errors"
# Use bash -n to syntax check
if bash -n run-agent.sh; then
    echo "✓ Refactored run-agent.sh syntax is valid"
else
    echo "✗ FAILED: Syntax error in run-agent.sh"
    exit 1
fi

echo ""
echo "Test 2: Check Docker args sections exist in refactored script"
# Verify key sections are present
if grep -q "DOCKER_ARGS" run-agent.sh && \
   grep -q "docker.*\"\${DOCKER_ARGS\[@\]}\"" run-agent.sh; then
    echo "✓ Docker args array and execution present"
else
    echo "✗ FAILED: Docker args structure missing"
    exit 1
fi

echo ""
echo "Test 3: Verify volume mounts are preserved"
if grep -q "incident.json.*:ro" run-agent.sh && \
   grep -q ".kube/config.*:ro" run-agent.sh && \
   grep -q "output" run-agent.sh; then
    echo "✓ Critical volume mounts present"
else
    echo "✗ FAILED: Volume mounts missing or incorrect"
    exit 1
fi

echo ""
echo "Test 4: Verify environment variable passing"
if grep -q "ANTHROPIC_API_KEY" run-agent.sh && \
   grep -q "OPENAI_API_KEY" run-agent.sh && \
   grep -q "KUBERNETES_CONTEXT" run-agent.sh; then
    echo "✓ API keys and context env vars are passed"
else
    echo "✗ FAILED: Environment variable passing incomplete"
    exit 1
fi

echo ""
echo "Test 5: Verify DEBUG mode container retention"
if grep -q "DEBUG.*!= \"true\"" run-agent.sh && \
   grep -q "DOCKER_ARGS+=(\"--rm\")" run-agent.sh; then
    echo "✓ DEBUG mode controls --rm flag"
else
    echo "✗ FAILED: DEBUG mode container retention logic missing"
    exit 1
fi

echo ""
echo "Test 6: Verify container naming for session extraction"
if grep -q "nightcrier-agent-\${INCIDENT_ID}" run-agent.sh || \
   grep -q "--name.*nightcrier-agent" run-agent.sh; then
    echo "✓ Container naming for session extraction present"
else
    echo "✗ FAILED: Container naming missing"
    exit 1
fi

echo ""
echo "Test 7: Verify timeout handling"
if grep -q "CONTAINER_TIMEOUT" run-agent.sh && \
   grep -q "timeout.*docker" run-agent.sh; then
    echo "✓ Timeout wrapper present"
else
    echo "✗ FAILED: Timeout handling missing"
    exit 1
fi

echo ""
echo "Test 8: Verify agent command dispatch"
if grep -q "AGENT_RUNNER=.*runners/\${AGENT_CLI}.sh" run-agent.sh && \
   grep -q "AGENT_CMD=\$(bash.*\$AGENT_RUNNER" run-agent.sh; then
    echo "✓ Agent command dispatch logic present"
else
    echo "✗ FAILED: Agent dispatch logic missing or incorrect"
    exit 1
fi

echo ""
echo "Test 9: Verify post-run hook dispatch"
if grep -q "post.*hook" run-agent.sh && \
   grep -q "runners/\${AGENT_CLI}-post.sh" run-agent.sh; then
    echo "✓ Post-run hook dispatch present"
else
    echo "✗ FAILED: Post-run hook dispatch missing"
    exit 1
fi

echo ""
echo "Test 10: Compare critical sections with original"
# Check that the original's Docker setup hasn't been accidentally removed
OLD_SCRIPT="/Users/rbias/code/nightcrier/agent-container/run-agent.sh"
NEW_SCRIPT="run-agent.sh"

# Count occurrences of key Docker flags in both
old_volume_count=$(grep -c "\-v" "$OLD_SCRIPT" || echo "0")
new_volume_count=$(grep -c "\-v" "$NEW_SCRIPT" || echo "0")

if [[ $new_volume_count -ge $((old_volume_count - 2)) ]]; then
    echo "✓ Volume mount count preserved (old: $old_volume_count, new: $new_volume_count)"
else
    echo "✗ FAILED: Volume mount count decreased significantly"
    echo "  Old: $old_volume_count, New: $new_volume_count"
    exit 1
fi

# Cleanup
rm -rf scratch/test-incident

echo ""
echo "=== All Docker Args Tests PASSED ==="
