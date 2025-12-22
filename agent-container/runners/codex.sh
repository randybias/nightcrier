#!/usr/bin/env bash
#
# codex.sh - Codex CLI sub-runner
#
# This script builds the command string for running Codex CLI.
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

# Require OpenAI API key
if [[ -z "$OPENAI_API_KEY" ]]; then
    log_error "OPENAI_API_KEY is required for Codex"
    exit 1
fi

# =============================================================================
# Build Codex Command
# =============================================================================

build_codex_command() {
    local cmd=""

    # Add debug environment check if enabled
    local debug_check
    debug_check=$(build_debug_env_check)
    if [[ -n "$debug_check" ]]; then
        cmd="$debug_check"
    fi

    # Codex requires login with API key first
    # Store API key in temp file for login, then remove it
    cmd+="echo -n \"\${OPENAI_API_KEY}\" > /tmp/.codex-key && "
    cmd+="codex login --with-api-key < /tmp/.codex-key && "
    cmd+="rm -f /tmp/.codex-key && "

    # Start Codex command with exec mode for non-interactive/headless operation
    # --skip-git-repo-check: needed when not in a git repo
    # --enable skills: enable SKILL.md loading from ~/.codex/skills
    # --dangerously-bypass-approvals-and-sandbox: required in Docker containers without Landlock
    cmd+="codex exec --skip-git-repo-check --enable skills --dangerously-bypass-approvals-and-sandbox"

    # Model mapping for Codex
    # Codex uses OpenAI model names, but we support friendly aliases
    if [[ -n "$LLM_MODEL" ]]; then
        case "$LLM_MODEL" in
            opus|gpt-5-codex)
                cmd+=" -m gpt-5-codex"
                ;;
            sonnet|gpt-5.2)
                cmd+=" -m gpt-5.2"
                ;;
            haiku|gpt-4o)
                cmd+=" -m gpt-4o"
                ;;
            *)
                # Pass through custom model name
                cmd+=" -m $LLM_MODEL"
                ;;
        esac
    fi

    # Add the prompt (AGENTS.md is in /home/agent so Codex finds it automatically)
    local escaped_prompt
    escaped_prompt=$(escape_single_quotes "$PROMPT")
    cmd+=" '${escaped_prompt}'"

    # Tee output to file
    cmd+=" 2>&1 | tee ${WORKSPACE_DIR}/logs/${OUTPUT_FILE}"

    echo "$cmd"
}

# Output the command to stdout
build_codex_command
