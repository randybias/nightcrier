#!/usr/bin/env bash
#
# goose-post.sh - Goose CLI post-run hooks
#
# This script extracts session artifacts after Goose execution.
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
    log_debug "Post-run: Skipping Goose session extraction (not in DEBUG mode)"
    exit 0
fi

# Require INCIDENT_ID for container name
if [[ -z "${INCIDENT_ID:-}" ]]; then
    log_debug "Post-run: No INCIDENT_ID set, skipping Goose session extraction"
    exit 0
fi

CONTAINER_NAME="nightcrier-agent-${INCIDENT_ID}"
log_debug "Post-run: Extracting Goose session from container: $CONTAINER_NAME"

# =============================================================================
# Extract Goose Config Directory
# =============================================================================

# Try to extract ~/.config/goose from the container
SESSION_EXTRACT_DIR="$WORKSPACE_DIR/goose-session-src"

if docker cp "$CONTAINER_NAME:/home/agent/.config/goose" "$SESSION_EXTRACT_DIR" 2>/dev/null; then
    mkdir -p "$WORKSPACE_DIR/logs"

    # Goose uses SQLite database for sessions
    # Command extraction from SQLite is complex - create minimal log for now
    log_debug "Post-run: Goose uses SQLite sessions.db (complex extraction - creating minimal log)"

    {
        echo "# Agent Commands Executed"
        echo "# Agent: goose"
        echo "# Generated: $(date -u +"%Y-%m-%dT%H:%M:%SZ")"
        echo "# Incident: ${INCIDENT_ID:-unknown}"
        echo "#"
        echo "# Note: Goose stores sessions in SQLite database"
        echo "# Session database: ~/.config/goose/sessions.db"
        echo "# Command extraction from SQLite requires additional tooling"
        echo ""

        # If sqlite3 is available, attempt to extract some session info
        if command -v sqlite3 >/dev/null 2>&1 && [[ -f "$SESSION_EXTRACT_DIR/sessions.db" ]]; then
            echo "# Session database found - attempting basic extraction:"
            echo ""
            sqlite3 "$SESSION_EXTRACT_DIR/sessions.db" \
                "SELECT '# Session: ' || name || ' (started: ' || created_at || ')' FROM sessions ORDER BY created_at DESC LIMIT 5;" \
                2>/dev/null || echo "# (SQLite extraction failed)"
        else
            echo "# (sqlite3 not available or sessions.db not found - cannot extract session details)"
        fi
    } > "$WORKSPACE_DIR/logs/agent-commands-executed.log"

    log_debug "Post-run: Created commands log (SQLite extraction requires additional tooling)"

    # Create session archive
    create_archive \
        "$SESSION_EXTRACT_DIR" \
        "$WORKSPACE_DIR/logs/agent-session.tar.gz"

    # Clean up extracted source
    rm -rf "$SESSION_EXTRACT_DIR"

    log_debug "Post-run: Goose session extraction complete"
else
    log_debug "Post-run: Could not extract Goose config (container may have exited or config doesn't exist)"
fi

exit 0
