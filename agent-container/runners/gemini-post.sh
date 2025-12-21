#!/usr/bin/env bash
#
# gemini-post.sh - Gemini CLI post-run hooks
#
# This script extracts session artifacts after Gemini execution.
# It is sourced by run-agent.sh after the agent completes.
#
# Expects environment variables from common.sh contract.
#

# Source common utilities
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=./common.sh
source "$SCRIPT_DIR/common.sh"

# Only run in DEBUG mode
if [[ "$DEBUG" != "true" ]]; then
    log_debug "Post-run: Skipping Gemini session extraction (not in DEBUG mode)"
    exit 0
fi

# Require INCIDENT_ID for container name
if [[ -z "${INCIDENT_ID:-}" ]]; then
    log_debug "Post-run: No INCIDENT_ID set, skipping Gemini session extraction"
    exit 0
fi

CONTAINER_NAME="nightcrier-agent-${INCIDENT_ID}"
log_debug "Post-run: Extracting Gemini session from container: $CONTAINER_NAME"

# =============================================================================
# Extract Gemini Session Directory
# =============================================================================

# Try to extract ~/.gemini from the container
SESSION_EXTRACT_DIR="$WORKSPACE_DIR/gemini-session-src"

if docker cp "$CONTAINER_NAME:/home/agent/.gemini" "$SESSION_EXTRACT_DIR" 2>/dev/null; then
    mkdir -p "$WORKSPACE_DIR/logs"

    # Find session JSON files in ~/.gemini/tmp/*/chats/session-*.json
    LOG_FILES=$(find "$SESSION_EXTRACT_DIR/tmp" -path "*/chats/session-*.json" -type f 2>/dev/null)

    # Extract commands from Gemini session JSON format
    if [[ -n "$LOG_FILES" ]]; then
        # Use the most recent session file
        LOGS_JSON=$(echo "$LOG_FILES" | xargs ls -t 2>/dev/null | head -1)

        if [[ -f "$LOGS_JSON" ]]; then
            log_debug "Post-run: Extracting commands from Gemini logs.json"

            # Create agent-commands-executed.log
            {
                echo "# Agent Commands Executed"
                echo "# Agent: gemini"
                echo "# Generated: $(date -u +"%Y-%m-%dT%H:%M:%SZ")"
                echo "# Incident: ${INCIDENT_ID:-unknown}"
                echo "#"
                echo "# Note: Command extraction from Gemini logs.json format"
                echo ""

                # Parse JSON (not JSONL) for command extraction
                # Gemini logs.json contains an array of events
                jq -r '
                    .[] |
                    select(.type == "tool_use" and .tool_name == "bash") |
                    "$ " + .tool_input.command + (if .tool_input.description then " # " + .tool_input.description else "" end)
                ' "$LOGS_JSON" 2>/dev/null || echo "# (command extraction failed or no commands found)"

            } > "$WORKSPACE_DIR/logs/agent-commands-executed.log"

            cmd_count=$(grep -c '^\$' "$WORKSPACE_DIR/logs/agent-commands-executed.log" 2>/dev/null || echo "0")
            log_debug "Post-run: Extracted $cmd_count commands from Gemini session"
        fi
    else
        log_debug "Post-run: No Gemini logs.json files found for command extraction"
    fi

    # Create session archive
    create_archive \
        "$SESSION_EXTRACT_DIR" \
        "$WORKSPACE_DIR/logs/agent-session.tar.gz"

    # Clean up extracted source
    rm -rf "$SESSION_EXTRACT_DIR"

    log_debug "Post-run: Gemini session extraction complete"
else
    log_debug "Post-run: Could not extract Gemini session (container may have exited or session doesn't exist)"
fi

exit 0
