#!/usr/bin/env bash
#
# codex-post.sh - Codex CLI post-run hooks
#
# This script extracts session artifacts after Codex execution.
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
    log_debug "Post-run: Skipping Codex session extraction (not in DEBUG mode)"
    exit 0
fi

# Require INCIDENT_ID for container name
if [[ -z "${INCIDENT_ID:-}" ]]; then
    log_debug "Post-run: No INCIDENT_ID set, skipping Codex session extraction"
    exit 0
fi

CONTAINER_NAME="nightcrier-agent-${INCIDENT_ID}"
log_debug "Post-run: Extracting Codex session from container: $CONTAINER_NAME"

# =============================================================================
# Extract Codex Session Directory
# =============================================================================

# Try to extract ~/.codex from the container
SESSION_EXTRACT_DIR="$WORKSPACE_DIR/codex-session-src"

if docker cp "$CONTAINER_NAME:/home/agent/.codex" "$SESSION_EXTRACT_DIR" 2>/dev/null; then
    mkdir -p "$WORKSPACE_DIR/logs"

    # Find the most recent session JSONL file in ~/.codex/sessions/
    JSONL_FILE=$(find "$SESSION_EXTRACT_DIR/sessions" -name "*.jsonl" -type f 2>/dev/null | \
                 xargs ls -t 2>/dev/null | head -1)

    # Extract commands from JSONL
    if [[ -n "$JSONL_FILE" && -f "$JSONL_FILE" ]]; then
        extract_commands_from_jsonl \
            "$JSONL_FILE" \
            "$WORKSPACE_DIR/logs/agent-commands-executed.log"
    else
        log_debug "Post-run: No Codex session JSONL file found for command extraction"
    fi

    # Create session archive
    create_archive \
        "$SESSION_EXTRACT_DIR" \
        "$WORKSPACE_DIR/logs/agent-session.tar.gz"

    # Clean up extracted source
    rm -rf "$SESSION_EXTRACT_DIR"

    log_debug "Post-run: Codex session extraction complete"
else
    log_debug "Post-run: Could not extract Codex session (container may have exited or session doesn't exist)"
fi

exit 0
