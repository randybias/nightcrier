#!/usr/bin/env bash
#
# goose.sh - Goose CLI command builder
#
# This script builds the Goose CLI execution command.
# It is sourced by run-agent.sh and outputs the command to stdout.
#
# Expects environment variables from common.sh contract.
#

# Source common utilities
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=./common.sh
source "$SCRIPT_DIR/common.sh"

# Validate required environment variables
validate_runner_env || exit 1

# Require API key for Goose (provider-specific)
if [[ -z "${OPENAI_API_KEY:-}" && -z "${ANTHROPIC_API_KEY:-}" && -z "${GEMINI_API_KEY:-}" ]]; then
    log_error "At least one API key required for Goose (OPENAI_API_KEY, ANTHROPIC_API_KEY, or GEMINI_API_KEY)"
    exit 1
fi

# Build Goose command
build_goose_command() {
    local cmd=""

    # Add debug environment check if in DEBUG mode
    local debug_check
    debug_check=$(build_debug_env_check)
    if [[ -n "$debug_check" ]]; then
        cmd="$debug_check"
    fi

    # Disable keyring (required for container/headless environments)
    cmd+="export GOOSE_DISABLE_KEYRING=1 && "

    # Start Goose session with prompt
    local escaped_prompt
    escaped_prompt=$(escape_single_quotes "$PROMPT")

    # Goose uses 'goose run --text' for non-interactive execution with prompt
    # --no-session: Run without creating/storing session history for automation
    cmd+="goose run --no-session --text '${escaped_prompt}'"

    # Model selection (if specified)
    if [[ -n "$LLM_MODEL" ]]; then
        cmd+=" --model $LLM_MODEL"
    fi

    # Tee output to file
    cmd+=" 2>&1 | tee ${AGENT_HOME}/logs/${OUTPUT_FILE}"

    echo "$cmd"
}

build_goose_command
