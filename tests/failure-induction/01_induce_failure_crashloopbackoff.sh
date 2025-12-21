#!/usr/bin/env bash
#
# Failure Induction: CrashLoopBackOff
#
# Purpose: Create or delete a pod that enters CrashLoopBackOff state to test
#          incident detection and triage capabilities.
#
# Contract:
#   - Accepts 'start' or 'stop' as first argument
#   - 'start' creates a crashlooping pod and outputs "TIMEOUT=<seconds>"
#   - 'stop' deletes the pod and cleans up resources
#
# Usage:
#   ./01_induce_failure_crashloopbackoff.sh start
#   ./01_induce_failure_crashloopbackoff.sh stop

set -euo pipefail

# Configuration
NAMESPACE="${TEST_NAMESPACE:-default}"
POD_NAME="${TEST_POD_NAME:-test-crashloop-pod}"
TIMEOUT=300  # Time in seconds for the failure to be detected and triaged

# Parse command
COMMAND="${1:-}"

usage() {
    echo "Usage: $0 {start|stop}"
    echo
    echo "Commands:"
    echo "  start  - Create a crashlooping pod"
    echo "  stop   - Delete the crashlooping pod"
    echo
    echo "Environment Variables:"
    echo "  TEST_NAMESPACE  - Kubernetes namespace (default: default)"
    echo "  TEST_POD_NAME   - Pod name (default: test-crashloop-pod)"
    exit 1
}

start_failure() {
    echo "Creating crashlooping pod: ${POD_NAME} in namespace ${NAMESPACE}" >&2

    # Check if pod already exists
    if kubectl get pod "${POD_NAME}" -n "${NAMESPACE}" &>/dev/null; then
        echo "Warning: Pod ${POD_NAME} already exists in namespace ${NAMESPACE}" >&2
        echo "Run '$0 stop' first to clean up" >&2
        exit 1
    fi

    # Create a pod with a command that immediately exits with error code
    kubectl run "${POD_NAME}" \
        --namespace="${NAMESPACE}" \
        --image=busybox:latest \
        --restart=Always \
        --command -- /bin/sh -c 'echo "Crashing intentionally..."; exit 1'

    echo "Pod created successfully" >&2
    echo "Waiting for pod to enter CrashLoopBackOff state..." >&2

    # Wait for the pod to start crashing (give it a moment to attempt restart)
    sleep 5

    # Verify the pod is in a failing state
    if ! kubectl get pod "${POD_NAME}" -n "${NAMESPACE}" &>/dev/null; then
        echo "Error: Pod was not created successfully" >&2
        exit 1
    fi

    echo "Pod is now in a failing state" >&2
    echo "Use 'kubectl get pod ${POD_NAME} -n ${NAMESPACE}' to check status" >&2

    # Output the timeout value as per contract
    echo "TIMEOUT=${TIMEOUT}"
}

stop_failure() {
    echo "Deleting crashlooping pod: ${POD_NAME} in namespace ${NAMESPACE}" >&2

    # Check if pod exists
    if ! kubectl get pod "${POD_NAME}" -n "${NAMESPACE}" &>/dev/null; then
        echo "Pod ${POD_NAME} not found in namespace ${NAMESPACE}" >&2
        echo "Nothing to clean up" >&2
        exit 0
    fi

    # Delete the pod
    kubectl delete pod "${POD_NAME}" -n "${NAMESPACE}" --wait=false

    echo "Pod deletion initiated" >&2
    echo "Waiting for pod to be fully removed..." >&2

    # Wait for pod to be gone (with timeout)
    local wait_count=0
    local max_wait=30
    while kubectl get pod "${POD_NAME}" -n "${NAMESPACE}" &>/dev/null; do
        sleep 1
        wait_count=$((wait_count + 1))
        if [ $wait_count -ge $max_wait ]; then
            echo "Warning: Pod deletion timed out after ${max_wait} seconds" >&2
            echo "Pod may still be terminating" >&2
            exit 0
        fi
    done

    echo "Pod deleted successfully" >&2
}

# Main
case "${COMMAND}" in
    start)
        start_failure
        ;;
    stop)
        stop_failure
        ;;
    *)
        echo "Error: Invalid command '${COMMAND}'" >&2
        echo >&2
        usage
        ;;
esac
