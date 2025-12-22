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

# Map Claude-style tool names to Gemini CLI tool names
map_tools_to_gemini() {
    local tools="$1"
    local gemini_tools=""

    # Split by comma and map each tool
    IFS=',' read -ra TOOL_ARRAY <<< "$tools"
    for tool in "${TOOL_ARRAY[@]}"; do
        case "$tool" in
            Read)
                gemini_tools+="read_file,"
                ;;
            Write)
                gemini_tools+="write_file,"
                ;;
            Grep)
                gemini_tools+="grep,"
                ;;
            Glob)
                gemini_tools+="glob,"
                ;;
            Bash)
                gemini_tools+="run_shell_command,"
                ;;
            Skill)
                # Gemini doesn't have direct skill equivalent, skip
                ;;
            *)
                log_warn "Unknown tool: $tool (skipping)"
                ;;
        esac
    done

    # Remove trailing comma
    echo "${gemini_tools%,}"
}

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

    # Map and add allowed tools
    if [[ -n "$AGENT_ALLOWED_TOOLS" ]]; then
        local gemini_tools
        gemini_tools=$(map_tools_to_gemini "$AGENT_ALLOWED_TOOLS")
        if [[ -n "$gemini_tools" ]]; then
            cmd+=" --allowed-tools \"$gemini_tools\""
        fi
    fi

    # Tee output to file
    cmd+=" 2>&1 | tee ${WORKSPACE_DIR}/logs/${OUTPUT_FILE}"

    echo "$cmd"
}

# Output the command to stdout
build_gemini_command
