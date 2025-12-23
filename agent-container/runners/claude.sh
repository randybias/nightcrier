#!/usr/bin/env bash
#
# claude.sh - Claude Code CLI sub-runner
#
# This script builds the command string for running Claude Code CLI.
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

# Require Claude API key
if [[ -z "$ANTHROPIC_API_KEY" ]]; then
    log_error "ANTHROPIC_API_KEY is required for Claude"
    exit 1
fi

# =============================================================================
# Build Claude Command
# =============================================================================

build_claude_command() {
    local cmd=""

    # Add debug environment check if enabled
    local debug_check
    debug_check=$(build_debug_env_check)
    if [[ -n "$debug_check" ]]; then
        cmd="$debug_check"
    fi

    # Start Claude command
    local escaped_prompt
    escaped_prompt=$(escape_single_quotes "$PROMPT")
    cmd+="claude -p '${escaped_prompt}'"

    # Model
    if [[ -n "$LLM_MODEL" ]]; then
        cmd+=" --model $LLM_MODEL"
    fi

    # Output format
    if [[ -n "$OUTPUT_FORMAT" ]]; then
        cmd+=" --output-format $OUTPUT_FORMAT"
    fi

    # Allowed tools
    if [[ -n "$AGENT_ALLOWED_TOOLS" ]]; then
        cmd+=" --allowedTools $AGENT_ALLOWED_TOOLS"
    fi

    # System prompt (inline)
    if [[ -n "$SYSTEM_PROMPT" ]]; then
        local escaped_system
        escaped_system=$(escape_single_quotes "$SYSTEM_PROMPT")
        cmd+=" --append-system-prompt '${escaped_system}'"
    fi

    # System prompt (file)
    if [[ -n "$SYSTEM_PROMPT_FILE" ]]; then
        cmd+=" --append-system-prompt-file /tmp/system-prompt.txt"
    fi

    # Preloaded context (incident + permissions + initial_triage_report)
    if [[ -n "$PRELOADED_CONTEXT" ]]; then
        local escaped_context
        escaped_context=$(escape_single_quotes "$PRELOADED_CONTEXT")
        cmd+=" --append-system-prompt '${escaped_context}'"
    fi

    # Verbose
    if [[ "$AGENT_VERBOSE" == "true" ]]; then
        cmd+=" --verbose"
    fi

    # Max turns
    if [[ -n "$AGENT_MAX_TURNS" ]]; then
        cmd+=" --max-turns $AGENT_MAX_TURNS"
    fi

    # Tee output to file (uses container path that's mounted from host)
    cmd+=" 2>&1 | tee ${AGENT_HOME}/logs/${OUTPUT_FILE}"

    echo "$cmd"
}

# Output the command to stdout
build_claude_command
