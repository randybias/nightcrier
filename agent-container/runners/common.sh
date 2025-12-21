#!/usr/bin/env bash
#
# common.sh - Shared functions for agent sub-runners
#
# This file provides common utilities used by all agent sub-runners.
# It should be sourced by both the main run-agent.sh orchestrator and
# individual agent runner scripts.
#
# Environment Contract:
# Sub-runners receive these standardized environment variables:
#
#   AGENT_CLI          - Agent name (claude, codex, gemini, goose)
#   AGENT_HOME         - /home/agent in container
#   PROMPT             - The investigation prompt
#   LLM_MODEL          - Model to use
#   AGENT_VERBOSE      - true/false for verbose output
#   AGENT_ALLOWED_TOOLS- Comma-separated tool list
#   SYSTEM_PROMPT      - Inline system prompt (optional)
#   SYSTEM_PROMPT_FILE - Path to system prompt file (optional)
#   OUTPUT_FORMAT      - Output format (optional, agent-specific)
#   AGENT_MAX_TURNS    - Maximum conversation turns (optional)
#   OUTPUT_FILE        - Target output filename
#   WORKSPACE_DIR      - Host workspace directory
#   INCIDENT_ID        - Incident identifier
#   DEBUG              - true/false for debug mode
#
#   # API keys (agent-specific)
#   ANTHROPIC_API_KEY  - For Claude
#   OPENAI_API_KEY     - For Codex
#   GEMINI_API_KEY     - For Gemini
#   GOOGLE_API_KEY     - For Gemini (alternate)
#

# =============================================================================
# Logging Utilities
# =============================================================================

# Log a debug message (only shown when DEBUG=true)
log_debug() {
    if [[ "$DEBUG" == "true" ]]; then
        echo "DEBUG: $*" >&2
    fi
}

# Log an info message
log_info() {
    echo "INFO: $*" >&2
}

# Log an error message
log_error() {
    echo "ERROR: $*" >&2
}

# =============================================================================
# Path Utilities
# =============================================================================

# Get the directory of the current script
get_script_dir() {
    local source="${BASH_SOURCE[0]}"
    while [[ -h "$source" ]]; do
        local dir
        dir="$(cd -P "$(dirname "$source")" && pwd)"
        source="$(readlink "$source")"
        [[ $source != /* ]] && source="$dir/$source"
    done
    cd -P "$(dirname "$source")" && pwd
}

# =============================================================================
# Validation Utilities
# =============================================================================

# Validate that a required environment variable is set
require_env() {
    local var_name="$1"
    local var_value="${!var_name}"

    if [[ -z "$var_value" ]]; then
        log_error "Required environment variable not set: $var_name"
        return 1
    fi
    return 0
}

# Validate all required environment variables for sub-runners
validate_runner_env() {
    local missing=()

    [[ -z "${AGENT_CLI:-}" ]] && missing+=("AGENT_CLI")
    [[ -z "${AGENT_HOME:-}" ]] && missing+=("AGENT_HOME")
    [[ -z "${PROMPT:-}" ]] && missing+=("PROMPT")
    [[ -z "${LLM_MODEL:-}" ]] && missing+=("LLM_MODEL")
    [[ -z "${OUTPUT_FILE:-}" ]] && missing+=("OUTPUT_FILE")

    if [[ ${#missing[@]} -gt 0 ]]; then
        log_error "Missing required environment variables: ${missing[*]}"
        return 1
    fi

    return 0
}

# =============================================================================
# Command Building Utilities
# =============================================================================

# Escape single quotes in a string for safe embedding in shell commands
escape_single_quotes() {
    local str="$1"
    echo "${str//\'/\'\\\'\'}"
}

# Build debug environment check commands (runs inside container)
build_debug_env_check() {
    if [[ "$DEBUG" != "true" ]]; then
        return
    fi

    # Output as single line to avoid issues with bash -c parsing
    echo "echo '=== Container Environment ===' >&2 && echo \"PWD: \$(pwd)\" >&2 && echo \"USER: \$(whoami)\" >&2 && echo \"HOME: \$HOME\" >&2 && echo \"ANTHROPIC_API_KEY: \${ANTHROPIC_API_KEY:+SET}\" >&2 && echo \"OPENAI_API_KEY: \${OPENAI_API_KEY:+SET}\" >&2 && echo \"Kubeconfig exists: \$(test -f \$HOME/.kube/config && echo YES || echo NO)\" >&2 && echo \"incident.json exists: \$(test -f incident.json && echo YES || echo NO)\" >&2 && echo \"Files in workspace: \$(ls -la | wc -l) files\" >&2 && echo '==============================' >&2 && "
}

# =============================================================================
# Session Extraction Utilities
# =============================================================================

# Extract commands from JSONL session files (Claude/Codex format)
# Usage: extract_commands_from_jsonl <jsonl_file> <output_file>
extract_commands_from_jsonl() {
    local jsonl_file="$1"
    local output_file="$2"

    if [[ ! -f "$jsonl_file" ]]; then
        log_debug "No JSONL file found: $jsonl_file"
        return 0
    fi

    log_debug "Extracting commands from: $jsonl_file"

    # Extract Bash tool calls using jq
    {
        echo "# Agent Commands Executed"
        echo "# Agent: ${AGENT_CLI:-unknown}"
        echo "# Generated: $(date -u +"%Y-%m-%dT%H:%M:%SZ")"
        echo "# Incident: ${INCIDENT_ID:-unknown}"
        echo "# Session: $(basename "$jsonl_file")"
        echo "#"
        echo ""

        # Parse JSONL and extract Bash commands
        jq -r '
            select(.type == "assistant") |
            .message.content[]? |
            select(.type == "tool_use" and .name == "Bash") |
            "$ " + .input.command + (if .input.description then " # " + .input.description else "" end)
        ' "$jsonl_file" 2>/dev/null || echo "# (command extraction failed)"

    } > "$output_file"

    local cmd_count
    cmd_count=$(grep -c '^\$' "$output_file" 2>/dev/null || echo "0")
    log_debug "Extracted $cmd_count commands to $(basename "$output_file")"
}

# Create a tar archive of a directory
# Usage: create_archive <source_dir> <output_tar_gz>
create_archive() {
    local source_dir="$1"
    local output_file="$2"

    if [[ ! -d "$source_dir" ]]; then
        log_debug "Source directory not found for archiving: $source_dir"
        return 0
    fi

    local parent_dir
    parent_dir="$(dirname "$source_dir")"
    local dir_name
    dir_name="$(basename "$source_dir")"

    tar -czf "$output_file" -C "$parent_dir" "$dir_name" 2>/dev/null

    if [[ -f "$output_file" ]]; then
        log_debug "Created archive: $output_file"
        return 0
    else
        log_debug "Failed to create archive: $output_file"
        return 1
    fi
}
