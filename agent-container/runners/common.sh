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

# Log a warning message
log_warning() {
    echo "WARNING: $*" >&2
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

# Write standard command log header
# Usage: write_command_log_header <output_file> <session_file>
write_command_log_header() {
    local output_file="$1"
    local session_file="$2"

    {
        echo "# Agent Commands Executed"
        echo "# Agent: ${AGENT_CLI:-unknown}"
        echo "# Generated: $(date -u +"%Y-%m-%dT%H:%M:%SZ")"
        echo "# Incident: ${INCIDENT_ID:-unknown}"
        echo "# Session: $(basename "$session_file")"
        echo "#"
        echo ""
    } > "$output_file"
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

# =============================================================================
# Context Preloading Utilities
# =============================================================================

# Preload incident context files and optionally run K8s triage script
# Returns formatted context string with XML-style tags
# Usage: preload_incident_context <workspace_dir> <skills_cache_dir> [disable_triage]
preload_incident_context() {
    local workspace_dir="$1"
    local skills_cache_dir="$2"
    local disable_triage="${3:-false}"
    local context=""

    log_debug "Preloading incident context from: $workspace_dir"

    # Read incident.json
    if [[ -f "$workspace_dir/incident.json" ]]; then
        log_debug "Reading incident.json"
        context+="<incident>\n"
        context+="$(cat "$workspace_dir/incident.json")\n"
        context+="</incident>\n\n"
    else
        log_debug "incident.json not found"
    fi

    # Read incident_cluster_permissions.json
    if [[ -f "$workspace_dir/incident_cluster_permissions.json" ]]; then
        log_debug "Reading incident_cluster_permissions.json"
        context+="<kubernetes_cluster_access_permissions>\n"
        context+="$(cat "$workspace_dir/incident_cluster_permissions.json")\n"
        context+="</kubernetes_cluster_access_permissions>\n\n"
    else
        log_debug "incident_cluster_permissions.json not found"
    fi

    # Run K8s triage script if not disabled
    if [[ "$disable_triage" != "true" && -n "$skills_cache_dir" ]]; then
        local triage_script="${skills_cache_dir}/k8s4agents/skills/k8s-troubleshooter/scripts/incident_triage.sh"

        if [[ -x "$triage_script" ]]; then
            log_debug "Executing K8s triage script: $triage_script"
            local triage_output
            if triage_output=$(timeout 30 "$triage_script" --skip-dump --output-dir "${workspace_dir}/triage" 2>&1); then
                context+="<initial_triage_report>\n"
                context+="${triage_output}\n"
                context+="</initial_triage_report>\n\n"
                log_debug "K8s triage completed successfully"
            else
                log_warning "K8s triage script failed or timed out, agent will run triage itself"
            fi
        else
            log_debug "K8s triage script not found or not executable: $triage_script"
            log_debug "Agent will run triage via skill"
        fi
    elif [[ "$disable_triage" == "true" ]]; then
        log_debug "Triage preloading disabled, agent will run triage itself"
    fi

    echo -e "$context"
}

# Append preloaded context to prompt-sent.md for audit trail
# Usage: append_preloaded_context_to_audit <workspace_dir> <preloaded_context>
append_preloaded_context_to_audit() {
    local workspace_dir="$1"
    local context="$2"
    local prompt_file="${workspace_dir}/prompt-sent.md"

    if [[ ! -f "$prompt_file" ]]; then
        log_warning "prompt-sent.md not found at: $prompt_file"
        return 1
    fi

    if [[ -z "$context" ]]; then
        log_debug "No preloaded context to append"
        return 0
    fi

    log_debug "Appending preloaded context to prompt-sent.md"

    # Append the preloaded context section
    {
        cat << 'PRELOAD_MARKER'

## Preloaded Context

The following context was preloaded into the agent's system prompt before execution:

PRELOAD_MARKER
        echo '```'
        echo -e "$context"
        echo '```'
    } >> "$prompt_file"

    log_debug "Preloaded context appended to audit trail"
}

# Monitor context size and warn if too large
# Usage: monitor_context_size <context_string>
monitor_context_size() {
    local context="$1"
    local char_count=${#context}
    local token_estimate=$((char_count / 4))

    if [[ $token_estimate -gt 10000 ]]; then
        log_warning "Preloaded context is very large: ~${token_estimate} tokens (may hit model limits)"
        # TODO: Implement truncation strategy if needed
    elif [[ $token_estimate -gt 8000 ]]; then
        log_warning "Preloaded context is large: ~${token_estimate} tokens"
    else
        log_debug "Preloaded context size: ~${token_estimate} tokens"
    fi
}
