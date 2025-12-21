#!/usr/bin/env bash
set -euo pipefail

echo "=== Testing Error Handling and Edge Cases ==="

# Setup base environment
export AGENT_HOME="/home/agent"
export LLM_MODEL="sonnet"
export AGENT_ALLOWED_TOOLS="Read,Grep,Bash"
export OUTPUT_FILE="test.log"

echo ""
echo "Test 1: Empty prompt"
export AGENT_CLI="claude"
export ANTHROPIC_API_KEY="test-key"
export PROMPT=""

set +e
result=$(bash runners/claude.sh 2>&1)
exit_code=$?
set -e

# Empty prompt should be caught by validate_runner_env
if [[ $exit_code -ne 0 ]] && [[ "$result" =~ "PROMPT" ]]; then
    echo "✓ Empty prompt is detected"
else
    echo "✗ FAILED: Empty prompt should be detected"
    echo "Exit code: $exit_code, Output: $result"
    exit 1
fi

echo ""
echo "Test 2: Prompt with newlines"
export PROMPT="Line 1
Line 2
Line 3"

CMD=$(bash runners/claude.sh 2>&1)
if [[ "$CMD" =~ "Line 1" ]]; then
    echo "✓ Multi-line prompts are handled"
else
    echo "✗ FAILED: Multi-line prompts not handled"
    echo "Command: $CMD"
    exit 1
fi

echo ""
echo "Test 3: Prompt with special bash characters"
export PROMPT="Test \$VAR and \`echo bad\` and \$(whoami)"

CMD=$(bash runners/claude.sh 2>&1)
# Should be escaped properly
if [[ "$CMD" =~ "Test" ]]; then
    echo "✓ Special characters in prompt are handled"
else
    echo "✗ FAILED: Special characters not handled"
    echo "Command: $CMD"
    exit 1
fi

echo ""
echo "Test 4: Very long prompt"
export PROMPT="$(printf 'A%.0s' {1..1000})"  # 1000 A's

CMD=$(bash runners/claude.sh 2>&1)
if [[ ${#CMD} -gt 1000 ]]; then
    echo "✓ Long prompts are handled"
else
    echo "✗ FAILED: Long prompt not handled properly"
    exit 1
fi

echo ""
echo "Test 5: Model name with special characters"
export PROMPT="Test"
export LLM_MODEL="claude-3-5-sonnet-20241022"

CMD=$(bash runners/claude.sh 2>&1)
if [[ "$CMD" =~ "claude-3-5-sonnet-20241022" ]]; then
    echo "✓ Complex model names are handled"
else
    echo "✗ FAILED: Complex model name not handled"
    echo "Command: $CMD"
    exit 1
fi

echo ""
echo "Test 6: System prompt with quotes"
export LLM_MODEL="sonnet"
export SYSTEM_PROMPT="Use 'single quotes' and \"double quotes\" carefully"

CMD=$(bash runners/claude.sh 2>&1)
if [[ "$CMD" =~ "single quotes" ]] && [[ "$CMD" =~ "double quotes" ]]; then
    echo "✓ Quotes in system prompt are handled"
else
    echo "✗ FAILED: Quotes in system prompt not handled"
    echo "Command: $CMD"
    exit 1
fi

echo ""
echo "Test 7: Codex model mapping edge cases"
export AGENT_CLI="codex"
export OPENAI_API_KEY="test-key"
unset ANTHROPIC_API_KEY || true
unset SYSTEM_PROMPT || true

# Test all model mappings
for model in "opus" "gpt-5-codex" "sonnet" "gpt-5.2" "haiku" "gpt-4o" "custom-model"; do
    export LLM_MODEL="$model"
    CMD=$(bash runners/codex.sh 2>&1)
    if [[ -n "$CMD" ]] && [[ "$CMD" =~ "codex" ]]; then
        echo "✓ Model '$model' mapping works"
    else
        echo "✗ FAILED: Model '$model' not handled"
        echo "Command: $CMD"
        exit 1
    fi
done

echo ""
echo "Test 8: Missing AGENT_HOME"
export AGENT_CLI="claude"
export ANTHROPIC_API_KEY="test-key"
export PROMPT="Test"
unset AGENT_HOME || true

set +e
result=$(bash runners/claude.sh 2>&1)
exit_code=$?
set -e

if [[ $exit_code -ne 0 ]] && [[ "$result" =~ "AGENT_HOME" ]]; then
    echo "✓ Missing AGENT_HOME is detected"
else
    echo "✗ FAILED: Missing AGENT_HOME should be detected"
    echo "Output: $result"
    exit 1
fi

echo ""
echo "=== All Error Handling Tests PASSED ==="
