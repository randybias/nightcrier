#!/usr/bin/env bash
#
# Claude Code PreToolUse hook for publishing "executing" events to NATS
#
# This script is invoked by Claude Code before each Bash tool call.
# It publishes the command being executed to NATS for progress tracking.
#
# Input: JSON from stdin with Claude hook format:
#   {"tool_name": "Bash", "tool_input": {"command": "..."}}
#
# Environment variables:
# - NATS_ENABLED: Set to "true" to enable publishing
# - NATS_SERVER: NATS server URL
# - NATS_TOKEN: Authentication token
# - INCIDENT_ID: Incident identifier for subject routing
# - CLUSTER: Target cluster name
# - AGENT_CLI: Agent type (claude)
# - LLM_MODEL: LLM model being used

# Exit immediately if NATS is disabled - this is the fast path
if [[ "${NATS_ENABLED:-false}" != "true" ]]; then
    exit 0
fi

# Read JSON from stdin
input=$(cat)

# Extract command from tool_input.command using jq
# Truncate to 100 characters for activity field
command=$(echo "${input}" | jq -r '.tool_input.command // empty' | head -c 100)

# Exit if no command found (shouldn't happen for Bash tool)
if [[ -z "${command}" ]]; then
    exit 0
fi

# Validate required NATS configuration
if [[ -z "${NATS_SERVER:-}" ]] || [[ -z "${NATS_TOKEN:-}" ]]; then
    exit 0
fi

# Build JSON payload
timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
payload=$(jq -n \
    --arg incident_id "${INCIDENT_ID:-unknown}" \
    --arg cluster "${CLUSTER:-unknown}" \
    --arg agent_cli "${AGENT_CLI:-claude}" \
    --arg agent_model "${LLM_MODEL:-unknown}" \
    --arg timestamp "${timestamp}" \
    --arg activity "${command}" \
    '{
        incident_id: $incident_id,
        cluster: $cluster,
        timestamp: $timestamp,
        event_type: "executing",
        agent_cli: $agent_cli,
        model: $agent_model,
        activity: $activity
    }')

# Publish to NATS with 3-second timeout
# Fire-and-forget: exit 0 regardless of success to not block Claude
subject="triage.${INCIDENT_ID:-unknown}.executing"
timeout 3s nats pub \
    --server="${NATS_SERVER}" \
    --token="${NATS_TOKEN}" \
    "${subject}" \
    "${payload}" >/dev/null 2>&1 || true

exit 0
