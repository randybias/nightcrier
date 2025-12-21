#!/usr/bin/env bash
set -euo pipefail

echo "=== Testing Claude Command Generation (Old vs New) ==="

# Setup test environment
export AGENT_CLI="claude"
export AGENT_HOME="/home/agent"
export AGENT_ALLOWED_TOOLS="Read,Grep,Glob,Bash"
export OUTPUT_FILE="test.log"
export ANTHROPIC_API_KEY="test-key-123"
export DEBUG="false"

echo ""
echo "Test 1: Basic command (prompt + model)"
export PROMPT="Investigate the incident"
export LLM_MODEL="sonnet"
export OUTPUT_FORMAT=""
export SYSTEM_PROMPT=""
export SYSTEM_PROMPT_FILE=""
export AGENT_VERBOSE=""
export AGENT_MAX_TURNS=""

NEW_CMD=$(bash runners/claude.sh)
EXPECTED="claude -p 'Investigate the incident' --model sonnet --allowedTools Read,Grep,Glob,Bash 2>&1 | tee /home/agent/output/test.log"

if [[ "$NEW_CMD" == "$EXPECTED" ]]; then
    echo "✓ Basic command matches"
else
    echo "✗ FAILED: Commands don't match"
    echo "Expected: $EXPECTED"
    echo "Got:      $NEW_CMD"
    exit 1
fi

echo ""
echo "Test 2: With verbose flag"
export AGENT_VERBOSE="true"
NEW_CMD=$(bash runners/claude.sh)
EXPECTED="claude -p 'Investigate the incident' --model sonnet --allowedTools Read,Grep,Glob,Bash --verbose 2>&1 | tee /home/agent/output/test.log"

if [[ "$NEW_CMD" == "$EXPECTED" ]]; then
    echo "✓ Verbose command matches"
else
    echo "✗ FAILED: Verbose command doesn't match"
    echo "Expected: $EXPECTED"
    echo "Got:      $NEW_CMD"
    exit 1
fi

echo ""
echo "Test 3: With output format"
export AGENT_VERBOSE=""
export OUTPUT_FORMAT="json"
NEW_CMD=$(bash runners/claude.sh)
EXPECTED="claude -p 'Investigate the incident' --model sonnet --output-format json --allowedTools Read,Grep,Glob,Bash 2>&1 | tee /home/agent/output/test.log"

if [[ "$NEW_CMD" == "$EXPECTED" ]]; then
    echo "✓ Output format command matches"
else
    echo "✗ FAILED: Output format command doesn't match"
    echo "Expected: $EXPECTED"
    echo "Got:      $NEW_CMD"
    exit 1
fi

echo ""
echo "Test 4: With system prompt (inline)"
export OUTPUT_FORMAT=""
export SYSTEM_PROMPT="Only use read-only kubectl commands"
NEW_CMD=$(bash runners/claude.sh)
EXPECTED="claude -p 'Investigate the incident' --model sonnet --allowedTools Read,Grep,Glob,Bash --append-system-prompt 'Only use read-only kubectl commands' 2>&1 | tee /home/agent/output/test.log"

if [[ "$NEW_CMD" == "$EXPECTED" ]]; then
    echo "✓ System prompt command matches"
else
    echo "✗ FAILED: System prompt command doesn't match"
    echo "Expected: $EXPECTED"
    echo "Got:      $NEW_CMD"
    exit 1
fi

echo ""
echo "Test 5: With max turns"
export SYSTEM_PROMPT=""
export AGENT_MAX_TURNS="50"
NEW_CMD=$(bash runners/claude.sh)
EXPECTED="claude -p 'Investigate the incident' --model sonnet --allowedTools Read,Grep,Glob,Bash --max-turns 50 2>&1 | tee /home/agent/output/test.log"

if [[ "$NEW_CMD" == "$EXPECTED" ]]; then
    echo "✓ Max turns command matches"
else
    echo "✗ FAILED: Max turns command doesn't match"
    echo "Expected: $EXPECTED"
    echo "Got:      $NEW_CMD"
    exit 1
fi

echo ""
echo "Test 6: With all options"
export OUTPUT_FORMAT="json"
export AGENT_VERBOSE="true"
export SYSTEM_PROMPT="Be thorough"
export AGENT_MAX_TURNS="100"
NEW_CMD=$(bash runners/claude.sh)
EXPECTED="claude -p 'Investigate the incident' --model sonnet --output-format json --allowedTools Read,Grep,Glob,Bash --append-system-prompt 'Be thorough' --verbose --max-turns 100 2>&1 | tee /home/agent/output/test.log"

if [[ "$NEW_CMD" == "$EXPECTED" ]]; then
    echo "✓ Full options command matches"
else
    echo "✗ FAILED: Full options command doesn't match"
    echo "Expected: $EXPECTED"
    echo "Got:      $NEW_CMD"
    exit 1
fi

echo ""
echo "Test 7: Special characters in prompt"
export OUTPUT_FORMAT=""
export AGENT_VERBOSE=""
export SYSTEM_PROMPT=""
export AGENT_MAX_TURNS=""
export PROMPT="Test with 'single quotes' and \"double quotes\""
NEW_CMD=$(bash runners/claude.sh)
# Check that single quotes are escaped
if [[ "$NEW_CMD" =~ "Test with" ]] && [[ "$NEW_CMD" =~ "quotes" ]]; then
    echo "✓ Special characters handled"
else
    echo "✗ FAILED: Special characters not handled properly"
    echo "Got: $NEW_CMD"
    exit 1
fi

echo ""
echo "=== All Claude Command Tests PASSED ==="
