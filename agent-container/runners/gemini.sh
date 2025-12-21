#!/usr/bin/env bash
#
# gemini.sh - Gemini CLI sub-runner
#
# This script builds the command string for running Gemini CLI.
# It is sourced by run-agent.sh and outputs the complete command to stdout.
#
# Expects environment variables from common.sh contract.
#

# Source common utilities
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=./common.sh
source "$SCRIPT_DIR/common.sh"

# Validate required environment
validate_runner_env || exit 1

# Require Gemini API key (try both GEMINI_API_KEY and GOOGLE_API_KEY)
if [[ -z "$GEMINI_API_KEY" && -z "$GOOGLE_API_KEY" ]]; then
    log_error "GEMINI_API_KEY or GOOGLE_API_KEY is required for Gemini"
    exit 1
fi

# =============================================================================
# Build Gemini Command
# =============================================================================

build_gemini_command() {
    local cmd=""

    # Start Gemini command
    local escaped_prompt
    escaped_prompt=$(escape_single_quotes "$PROMPT")
    cmd="gemini -p '${escaped_prompt}'"

    # Model
    if [[ -n "$LLM_MODEL" ]]; then
        cmd+=" --model $LLM_MODEL"
    fi

    # Tee output to file
    cmd+=" 2>&1 | tee ${AGENT_HOME}/output/${OUTPUT_FILE}"

    echo "$cmd"
}

# Output the command to stdout
build_gemini_command
