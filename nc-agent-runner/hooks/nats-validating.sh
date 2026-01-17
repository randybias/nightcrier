#!/usr/bin/env bash
#
# Claude Code Stop hook wrapper that publishes validation events to NATS
#
# This script wraps the validation process and publishes start/complete events.
# It's designed to be used as a wrapper around the actual validation script.
#
# Environment variables:
# - NATS_ENABLED: Set to "true" to enable publishing
# - NATS_SERVER: NATS server URL
# - NATS_TOKEN: Authentication token
# - INCIDENT_ID: Incident identifier for subject routing
# - CLUSTER: Target cluster name
# - AGENT_CLI: Agent type (claude)
# - LLM_MODEL: LLM model being used
# - VALIDATION_SCRIPT: Path to actual validation script (required)

set -euo pipefail

# Validation script must be provided as environment variable
VALIDATION_SCRIPT="${VALIDATION_SCRIPT:-}"
if [[ -z "$VALIDATION_SCRIPT" ]]; then
    echo "ERROR: VALIDATION_SCRIPT environment variable not set"
    exit 1
fi

if [[ ! -x "$VALIDATION_SCRIPT" ]]; then
    echo "ERROR: Validation script not found or not executable: $VALIDATION_SCRIPT"
    exit 1
fi

#######################################
# Publish NATS event
# Arguments:
#   $1 - Event type (validating.started, validating.completed, validating.failed)
#   $2 - Exit code (optional, for completed/failed events)
#   $3 - Activity message (optional)
#######################################
publish_nats_event() {
    local event_type="$1"
    local exit_code="${2:-0}"
    local activity="${3:-Validating triage report format}"

    # Exit immediately if NATS is disabled
    if [[ "${NATS_ENABLED:-false}" != "true" ]]; then
        return 0
    fi

    # Validate required NATS configuration
    if [[ -z "${NATS_SERVER:-}" ]] || [[ -z "${NATS_TOKEN:-}" ]]; then
        return 0
    fi

    # Build JSON payload
    local timestamp
    timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    local payload
    payload=$(jq -n \
        --arg incident_id "${INCIDENT_ID:-unknown}" \
        --arg cluster "${CLUSTER:-unknown}" \
        --arg agent_cli "${AGENT_CLI:-claude}" \
        --arg agent_model "${LLM_MODEL:-unknown}" \
        --arg timestamp "${timestamp}" \
        --arg event_type "${event_type}" \
        --arg activity "${activity}" \
        --argjson exit_code "${exit_code}" \
        '{
            incident_id: $incident_id,
            cluster: $cluster,
            timestamp: $timestamp,
            event_type: $event_type,
            agent_cli: $agent_cli,
            model: $agent_model,
            activity: $activity,
            exit_code: $exit_code
        }')

    # Publish to NATS with 3-second timeout
    # Fire-and-forget: don't block on NATS failures
    local subject="triage.${INCIDENT_ID:-unknown}.${event_type}"
    timeout 3s nats pub \
        --server="${NATS_SERVER}" \
        --token="${NATS_TOKEN}" \
        "${subject}" \
        "${payload}" >/dev/null 2>&1 || true
}

# Publish validation started event
publish_nats_event "validating.started" 0 "Starting triage report validation"

# Run the actual validation script and capture exit code
validation_exit_code=0
validation_output=""
validation_output=$("$VALIDATION_SCRIPT" 2>&1) || validation_exit_code=$?

# Publish completion event and output JSON for Claude Code Stop hook
# Claude Code expects JSON: {"decision": "allow"} or {"decision": "block", "reason": "..."}
case $validation_exit_code in
    0)
        publish_nats_event "validating.completed" 0 "Report validation passed"
        # Allow stop - validation passed
        jq -n '{decision: "allow"}'
        ;;
    1)
        publish_nats_event "validating.failed" 1 "Report validation failed (critical errors)"
        # Block stop - validation failed, include output as reason
        jq -n --arg reason "$validation_output" '{decision: "block", reason: $reason}'
        ;;
    2)
        publish_nats_event "validating.completed" 2 "Report validation passed with warnings"
        # Allow stop but include warnings - validation passed with warnings
        jq -n --arg reason "$validation_output" '{decision: "allow", reason: $reason}'
        ;;
    *)
        publish_nats_event "validating.failed" "$validation_exit_code" "Report validation error"
        # Block stop - unexpected error
        jq -n --arg reason "Validation script exited with code $validation_exit_code: $validation_output" '{decision: "block", reason: $reason}'
        ;;
esac

# Always exit 0 - the JSON decision controls whether Claude stops
# A non-zero exit here would be treated as hook failure, not validation failure
exit 0
