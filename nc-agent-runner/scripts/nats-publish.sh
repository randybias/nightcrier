#!/usr/bin/env bash
#
# NATS event publishing helper for nc-agent-runner
#
# This script provides functions to publish run lifecycle events to NATS.
# It is designed to be sourced by entrypoint.sh and fails gracefully
# when NATS is disabled or unavailable.
#
# Environment variables used:
# - NATS_ENABLED: Set to "true" to enable NATS publishing
# - NATS_SERVER: NATS server URL (e.g., nats://nats.triage.svc.cluster.local:4222)
# - NATS_TOKEN: Authentication token for NATS server
# - INCIDENT_ID: Unique incident identifier
# - CLUSTER: Target cluster name
# - AGENT_CLI: Agent type (claude, codex, gemini, goose)
# - LLM_MODEL: LLM model being used
# - EXIT_CODE: Agent exit code (for completed events)

set -euo pipefail

#######################################
# Build JSON payload for run.started event
# Globals:
#   INCIDENT_ID
#   CLUSTER
#   AGENT_CLI
#   LLM_MODEL
# Arguments:
#   None
# Outputs:
#   JSON string to stdout
#######################################
build_run_started_event() {
    local timestamp
    timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Use jq to build properly escaped JSON
    jq -n \
        --arg incident_id "${INCIDENT_ID}" \
        --arg cluster "${CLUSTER:-unknown}" \
        --arg agent_cli "${AGENT_CLI}" \
        --arg agent_model "${LLM_MODEL:-unknown}" \
        --arg timestamp "${timestamp}" \
        '{
            incident_id: $incident_id,
            cluster: $cluster,
            timestamp: $timestamp,
            event_type: "run.started",
            agent_cli: $agent_cli,
            model: $agent_model
        }'
}

#######################################
# Build JSON payload for run.completed event
# Globals:
#   INCIDENT_ID
#   CLUSTER
#   AGENT_CLI
#   LLM_MODEL
#   EXIT_CODE
# Arguments:
#   None
# Outputs:
#   JSON string to stdout
#######################################
build_run_completed_event() {
    local timestamp
    timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Use jq to build properly escaped JSON
    jq -n \
        --arg incident_id "${INCIDENT_ID}" \
        --arg cluster "${CLUSTER:-unknown}" \
        --arg agent_cli "${AGENT_CLI}" \
        --arg agent_model "${LLM_MODEL:-unknown}" \
        --argjson exit_code "${EXIT_CODE:-1}" \
        --arg timestamp "${timestamp}" \
        '{
            incident_id: $incident_id,
            cluster: $cluster,
            timestamp: $timestamp,
            event_type: "run.completed",
            agent_cli: $agent_cli,
            model: $agent_model,
            exit_code: $exit_code
        }'
}

#######################################
# Publish event to NATS
# Arguments:
#   $1 - NATS subject (e.g., triage.incident-123.run.started)
#   $2 - JSON payload
# Returns:
#   0 on success or if NATS is disabled
#   1 on failure (logs warning but doesn't exit)
#######################################
publish_event() {
    local subject="$1"
    local payload="$2"

    # Check if NATS is enabled
    if [[ "${NATS_ENABLED:-false}" != "true" ]]; then
        echo "NATS publishing disabled, skipping event: ${subject}"
        return 0
    fi

    # Validate required NATS configuration
    if [[ -z "${NATS_SERVER:-}" ]]; then
        echo "WARNING: NATS_SERVER not set, cannot publish event: ${subject}"
        return 0
    fi

    if [[ -z "${NATS_TOKEN:-}" ]]; then
        echo "WARNING: NATS_TOKEN not set, cannot publish event: ${subject}"
        return 0
    fi

    echo "Publishing NATS event: ${subject}"

    # Publish with 3-second timeout
    # Use --token for authentication
    if timeout 3s nats pub \
        --server="${NATS_SERVER}" \
        --token="${NATS_TOKEN}" \
        "${subject}" \
        "${payload}" 2>&1; then
        echo "Successfully published event to ${subject}"
        return 0
    else
        echo "WARNING: Failed to publish NATS event to ${subject} (continuing anyway)"
        return 1
    fi
}

#######################################
# Convenience function: Publish run.started event
# Globals:
#   INCIDENT_ID
# Arguments:
#   None
#######################################
publish_run_started() {
    local subject="triage.${INCIDENT_ID}.run.started"
    local payload
    payload=$(build_run_started_event)
    publish_event "${subject}" "${payload}"
}

#######################################
# Convenience function: Publish run.completed event
# Globals:
#   INCIDENT_ID
#   EXIT_CODE
# Arguments:
#   None
#######################################
publish_run_completed() {
    local subject="triage.${INCIDENT_ID}.run.completed"
    local payload
    payload=$(build_run_completed_event)
    publish_event "${subject}" "${payload}"
}
