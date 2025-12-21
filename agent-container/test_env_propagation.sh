#!/usr/bin/env bash
set -euo pipefail

echo "=== Testing Environment Variable Propagation ==="

echo ""
echo "Test 1: Required variables are validated"
# Test that missing required vars are caught
export AGENT_CLI="claude"
export AGENT_HOME="/home/agent"
export PROMPT="Test"
export LLM_MODEL="sonnet"
# Missing OUTPUT_FILE
export ANTHROPIC_API_KEY="test-key"

set +e
result=$(bash runners/claude.sh 2>&1)
exit_code=$?
set -e

if [[ $exit_code -ne 0 ]] && [[ "$result" =~ "OUTPUT_FILE" ]]; then
    echo "✓ Missing OUTPUT_FILE is detected"
else
    echo "✗ FAILED: Should detect missing OUTPUT_FILE"
    echo "Exit code: $exit_code"
    echo "Output: $result"
    exit 1
fi

echo ""
echo "Test 2: Optional variables handled correctly"
export OUTPUT_FILE="test.log"
export AGENT_VERBOSE=""  # Empty, should be optional
export SYSTEM_PROMPT=""  # Empty, should be optional
export OUTPUT_FORMAT=""  # Empty, should be optional

CMD=$(bash runners/claude.sh 2>&1)
if [[ -n "$CMD" ]] && [[ "$CMD" =~ "claude" ]] && [[ ! "$CMD" =~ "--verbose" ]]; then
    echo "✓ Optional empty variables don't add flags"
else
    echo "✗ FAILED: Optional variables not handled correctly"
    echo "Command: $CMD"
    exit 1
fi

echo ""
echo "Test 3: Set optional variables are included"
export AGENT_VERBOSE="true"
export SYSTEM_PROMPT="Be careful"
export OUTPUT_FORMAT="json"

CMD=$(bash runners/claude.sh 2>&1)
if [[ "$CMD" =~ "--verbose" ]] && [[ "$CMD" =~ "--output-format json" ]] && [[ "$CMD" =~ "Be careful" ]]; then
    echo "✓ Set optional variables are included in command"
else
    echo "✗ FAILED: Set optional variables not included"
    echo "Command: $CMD"
    exit 1
fi

echo ""
echo "Test 4: Agent-specific API keys"
# Test Claude requires ANTHROPIC_API_KEY
export AGENT_CLI="claude"
unset ANTHROPIC_API_KEY || true
export OPENAI_API_KEY="test"  # Wrong key for Claude

set +e
result=$(bash runners/claude.sh 2>&1)
exit_code=$?
set -e

if [[ $exit_code -ne 0 ]] && [[ "$result" =~ "ANTHROPIC_API_KEY" ]]; then
    echo "✓ Claude correctly requires ANTHROPIC_API_KEY"
else
    echo "✗ FAILED: Claude should require ANTHROPIC_API_KEY"
    echo "Output: $result"
    exit 1
fi

# Test Codex requires OPENAI_API_KEY
export AGENT_CLI="codex"
unset OPENAI_API_KEY || true
export ANTHROPIC_API_KEY="test"  # Wrong key for Codex

set +e
result=$(bash runners/codex.sh 2>&1)
exit_code=$?
set -e

if [[ $exit_code -ne 0 ]] && [[ "$result" =~ "OPENAI_API_KEY" ]]; then
    echo "✓ Codex correctly requires OPENAI_API_KEY"
else
    echo "✗ FAILED: Codex should require OPENAI_API_KEY"
    echo "Output: $result"
    exit 1
fi

# Test Gemini accepts both keys
export AGENT_CLI="gemini"
unset GEMINI_API_KEY || true
unset GOOGLE_API_KEY || true

set +e
result=$(bash runners/gemini.sh 2>&1)
exit_code=$?
set -e

if [[ $exit_code -ne 0 ]] && ([[ "$result" =~ "GEMINI_API_KEY" ]] || [[ "$result" =~ "GOOGLE_API_KEY" ]]); then
    echo "✓ Gemini correctly requires GEMINI_API_KEY or GOOGLE_API_KEY"
else
    echo "✗ FAILED: Gemini should require an API key"
    echo "Output: $result"
    exit 1
fi

# Test Gemini works with GEMINI_API_KEY
export GEMINI_API_KEY="test"
CMD=$(bash runners/gemini.sh 2>&1)
if [[ "$CMD" =~ "gemini" ]]; then
    echo "✓ Gemini works with GEMINI_API_KEY"
else
    echo "✗ FAILED: Gemini should work with GEMINI_API_KEY"
    exit 1
fi

# Test Gemini works with GOOGLE_API_KEY
unset GEMINI_API_KEY
export GOOGLE_API_KEY="test"
CMD=$(bash runners/gemini.sh 2>&1)
if [[ "$CMD" =~ "gemini" ]]; then
    echo "✓ Gemini works with GOOGLE_API_KEY"
else
    echo "✗ FAILED: Gemini should work with GOOGLE_API_KEY"
    exit 1
fi

echo ""
echo "=== All Environment Variable Tests PASSED ==="
