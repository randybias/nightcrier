#!/usr/bin/env bash
set -euo pipefail

echo "=== Testing common.sh Functions ==="
source runners/common.sh

# Test 1: escape_single_quotes
echo "Test 1: escape_single_quotes"
input="Don't use single quotes"
expected="Don'\\''t use single quotes"
result=$(escape_single_quotes "$input")
if [[ "$result" == "$expected" ]]; then
    echo "✓ escape_single_quotes works"
else
    echo "✗ FAILED: got '$result', expected '$expected'"
    exit 1
fi

# Test 2: log functions
echo ""
echo "Test 2: log functions"
DEBUG=true
log_debug "This is a debug message" 2>&1 | grep -q "DEBUG:" && echo "✓ log_debug works" || { echo "✗ log_debug failed"; exit 1; }
log_info "This is an info message" 2>&1 | grep -q "INFO:" && echo "✓ log_info works" || { echo "✗ log_info failed"; exit 1; }
log_error "This is an error message" 2>&1 | grep -q "ERROR:" && echo "✓ log_error works" || { echo "✗ log_error failed"; exit 1; }

# Test 3: validate_runner_env with missing vars
echo ""
echo "Test 3: validate_runner_env (should fail with missing vars)"
unset AGENT_CLI AGENT_HOME PROMPT LLM_MODEL OUTPUT_FILE
set +e  # Temporarily disable exit on error
validate_runner_env 2>/dev/null
result=$?
set -e  # Re-enable exit on error
if [[ $result -eq 0 ]]; then
    echo "✗ FAILED: Should have failed with missing vars"
    exit 1
else
    echo "✓ validate_runner_env correctly detects missing vars"
fi

# Test 4: validate_runner_env with all vars
echo ""
echo "Test 4: validate_runner_env (should pass with all vars)"
export AGENT_CLI="claude"
export AGENT_HOME="/home/agent"
export PROMPT="Test"
export LLM_MODEL="sonnet"
export OUTPUT_FILE="test.log"
set +e
validate_runner_env 2>/dev/null
result=$?
set -e
if [[ $result -eq 0 ]]; then
    echo "✓ validate_runner_env passes with all vars"
else
    echo "✗ FAILED: Should have passed with all vars (exit code: $result)"
    exit 1
fi

# Test 5: build_debug_env_check
echo ""
echo "Test 5: build_debug_env_check"
DEBUG=true
result=$(build_debug_env_check)
if [[ -n "$result" && "$result" =~ "Container Environment" ]]; then
    echo "✓ build_debug_env_check generates output in DEBUG mode"
else
    echo "✗ FAILED: build_debug_env_check didn't generate expected output"
    exit 1
fi

DEBUG=false
result=$(build_debug_env_check)
if [[ -z "$result" ]]; then
    echo "✓ build_debug_env_check returns empty in production mode"
else
    echo "✗ FAILED: build_debug_env_check should be empty in production mode"
    exit 1
fi

echo ""
echo "=== All common.sh Tests PASSED ==="
