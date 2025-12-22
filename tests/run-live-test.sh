#!/usr/bin/env bash
#
# run-live-test.sh - Main orchestration script for live test harness
#
# This script coordinates the entire test flow:
# 1. Generate config from template
# 2. Start nightcrier in background
# 3. Induce failure
# 4. Monitor logs for agent execution
# 5. Generate and display report
#
# Usage: ./run-live-test.sh <agent> <test-type> [--debug] [--json]
#
# Arguments:
#   agent:      claude, codex, or gemini
#   test-type:  crashloopbackoff (more types to be added)
#   --debug:    Enable agent debug mode
#   --json:     Output JSON report (default: human-readable)
#
# Example:
#   ./run-live-test.sh claude crashloopbackoff
#   ./run-live-test.sh codex crashloopbackoff --debug --json

set -euo pipefail

# Script directory and paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
TESTS_DIR="${SCRIPT_DIR}"
LIB_DIR="${TESTS_DIR}/lib"
FAILURE_DIR="${TESTS_DIR}/failure-induction"
LOGS_BASE="${TESTS_DIR}/logs"
SECRETS_FILE="${HOME}/dev-secrets/nightcrier-secrets.env"
NIGHTCRIER_BIN="${REPO_ROOT}/nightcrier"

# Source library functions (only the functions, not the main execution)
# For config-generator, we'll call it as a script instead of sourcing
# shellcheck disable=SC1091
source "${LIB_DIR}/log-monitor.sh"
# shellcheck disable=SC1091
source "${LIB_DIR}/report-generator.sh"

# Global state
NIGHTCRIER_PID=""
FAILURE_ACTIVE=false
TEST_ID=""
LOG_DIR=""
CONFIG_FILE=""
CLEANUP_DONE=false

# Usage information
usage() {
    cat >&2 <<EOF
Usage: $0 <agent> <test-type> [--debug] [--json]

Arguments:
  agent       Agent type: claude, codex, or gemini
  test-type   Failure type: crashloopbackoff
  --debug     Enable agent debug mode (optional)
  --json      Output JSON report format (optional)

Examples:
  $0 claude crashloopbackoff
  $0 codex crashloopbackoff --debug
  $0 gemini crashloopbackoff --json

Prerequisites:
  - Secrets file at ${SECRETS_FILE}
  - Nightcrier binary at ${NIGHTCRIER_BIN}
  - kubectl configured with cluster access
EOF
    exit 1
}

# Cleanup function - ensures resources are cleaned up on exit
cleanup() {
    if [ "$CLEANUP_DONE" = true ]; then
        return
    fi
    CLEANUP_DONE=true

    echo ""
    echo "=== Cleanup Started ===" >&2

    # Stop failure induction if active
    if [ "$FAILURE_ACTIVE" = true ]; then
        echo "Cleaning up failure scenario..." >&2
        local failure_script="${FAILURE_DIR}/01_induce_failure_${TEST_TYPE}.sh"
        if [ -f "$failure_script" ]; then
            "$failure_script" stop || echo "Warning: Failure cleanup failed" >&2
        fi
    fi

    # Stop nightcrier if running
    if [ -n "$NIGHTCRIER_PID" ]; then
        echo "Stopping nightcrier (PID: ${NIGHTCRIER_PID})..." >&2
        if kill -0 "$NIGHTCRIER_PID" 2>/dev/null; then
            kill -TERM "$NIGHTCRIER_PID" 2>/dev/null || true
            sleep 2
            if kill -0 "$NIGHTCRIER_PID" 2>/dev/null; then
                kill -KILL "$NIGHTCRIER_PID" 2>/dev/null || true
            fi
            echo "Nightcrier stopped" >&2
        else
            echo "Nightcrier already stopped" >&2
        fi
    fi

    # Clean up agent containers (kept during DEBUG mode for inspection)
    echo "Cleaning up agent containers..." >&2
    local container_count
    container_count=$(docker ps -a --filter "name=nightcrier-agent" --format "{{.ID}}" 2>/dev/null | wc -l)
    if [ "$container_count" -gt 0 ]; then
        docker ps -a --filter "name=nightcrier-agent" --format "{{.ID}}" | xargs docker rm 2>/dev/null || true
        echo "Removed ${container_count} agent container(s)" >&2
    else
        echo "No agent containers to clean up" >&2
    fi

    echo "=== Cleanup Complete ===" >&2
}

# Set trap for cleanup
trap cleanup EXIT INT TERM

# Generate unique test ID
# Format: test-YYYYMMDD-HHMMSS-<6-char-hash>
generate_test_id() {
    local timestamp
    timestamp="$(date +%Y%m%d-%H%M%S)"
    local random_hash
    random_hash="$(head -c 32 /dev/urandom | md5sum | cut -c1-6)"
    echo "test-${timestamp}-${random_hash}"
}

# Create log directory structure and metadata file
create_log_directory() {
    local test_id="$1"
    local agent="$2"
    local test_type="$3"
    local debug_mode="$4"

    local log_dir="${LOGS_BASE}/${test_id}"

    echo "Creating log directory: ${log_dir}" >&2
    mkdir -p "$log_dir"

    # Create metadata file
    local meta_file="${log_dir}/test.meta"
    cat > "$meta_file" <<EOF
TEST_ID="${test_id}"
TEST_AGENT="${agent}"
TEST_TYPE="${test_type}"
TEST_DEBUG="${debug_mode}"
TEST_START_TIME="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
EOF

    echo "$log_dir"
}

# Validate prerequisites
validate_prerequisites() {
    local errors=0

    # Check nightcrier binary
    if [ ! -x "$NIGHTCRIER_BIN" ]; then
        echo "ERROR: Nightcrier binary not found or not executable: ${NIGHTCRIER_BIN}" >&2
        ((errors++))
    fi

    # Check secrets file
    if [ ! -f "$SECRETS_FILE" ]; then
        echo "ERROR: Secrets file not found: ${SECRETS_FILE}" >&2
        echo "Please create it with required secrets" >&2
        ((errors++))
    fi

    # Check kubectl
    if ! command -v kubectl &>/dev/null; then
        echo "ERROR: kubectl not found in PATH" >&2
        ((errors++))
    fi

    if [ $errors -gt 0 ]; then
        echo "ERROR: ${errors} prerequisite check(s) failed" >&2
        exit 1
    fi
}

# Start nightcrier in background
start_nightcrier() {
    local config_file="$1"
    local log_file="$2"

    echo "Starting nightcrier..." >&2
    echo "  Config: ${config_file}" >&2
    echo "  Log:    ${log_file}" >&2

    # Export API keys from secrets file so nightcrier can pass them to agent containers
    # shellcheck disable=SC1090
    source "${SECRETS_FILE}"
    export ANTHROPIC_API_KEY
    export OPENAI_API_KEY
    export GEMINI_API_KEY

    # Start nightcrier in background, redirecting all output to log file
    "${NIGHTCRIER_BIN}" --config "$config_file" >> "$log_file" 2>&1 &
    NIGHTCRIER_PID=$!

    echo "Nightcrier started (PID: ${NIGHTCRIER_PID})" >&2

    # Verify process started
    sleep 1
    if ! kill -0 "$NIGHTCRIER_PID" 2>/dev/null; then
        echo "ERROR: Nightcrier failed to start" >&2
        echo "Check log file: ${log_file}" >&2
        exit 1
    fi

    echo "Nightcrier is running" >&2
}

# Main orchestration function
run_test() {
    local agent="$1"
    local test_type="$2"
    local debug_mode="$3"
    local json_output="$4"

    echo "========================================" >&2
    echo "Live Test Harness" >&2
    echo "========================================" >&2
    echo "Agent:      ${agent}" >&2
    echo "Test Type:  ${test_type}" >&2
    echo "Debug Mode: ${debug_mode}" >&2
    echo "========================================" >&2
    echo "" >&2

    # Step 1: Generate test ID and create log directory
    echo "Step 1: Generating test ID and creating log directory..." >&2
    TEST_ID="$(generate_test_id)"
    LOG_DIR="$(create_log_directory "$TEST_ID" "$agent" "$test_type" "$debug_mode")"
    echo "Test ID: ${TEST_ID}" >&2
    echo "Log Dir: ${LOG_DIR}" >&2
    echo "" >&2

    # Step 2: Validate prerequisites
    echo "Step 2: Validating prerequisites..." >&2
    validate_prerequisites
    echo "Prerequisites validated" >&2
    echo "" >&2

    # Step 3: Generate config from template
    echo "Step 3: Generating configuration..." >&2
    local debug_flag=""
    if [ "$debug_mode" = "true" ]; then
        debug_flag="--debug"
    fi
    "${LIB_DIR}/config-generator.sh" "$agent" $debug_flag
    CONFIG_FILE="${REPO_ROOT}/configs/config-test-${agent}.yaml"
    echo "Config generated: ${CONFIG_FILE}" >&2
    echo "" >&2

    # Step 4: Start nightcrier
    echo "Step 4: Starting nightcrier..." >&2
    local nightcrier_log="${LOG_DIR}/nightcrier.log"
    start_nightcrier "$CONFIG_FILE" "$nightcrier_log"
    echo "" >&2

    # Step 5: Wait for nightcrier to be ready
    echo "Step 5: Waiting for nightcrier to be ready..." >&2
    if ! wait_for_pattern "$nightcrier_log" "subscribed to fault events" 60 "nightcrier ready signal"; then
        echo "ERROR: Nightcrier did not become ready in time" >&2
        exit 1
    fi
    echo "Nightcrier is ready" >&2
    echo "" >&2

    # Step 6: Induce failure
    echo "Step 6: Inducing failure scenario..." >&2
    local failure_script="${FAILURE_DIR}/01_induce_failure_${test_type}.sh"

    if [ ! -x "$failure_script" ]; then
        echo "ERROR: Failure induction script not found or not executable: ${failure_script}" >&2
        exit 1
    fi

    # Export KUBECONFIG so failure induction script uses the correct cluster
    # shellcheck disable=SC1090
    source "${SECRETS_FILE}"
    export KUBECONFIG="${KUBECONFIG_PATH}"

    # Export unique pod name to avoid conflicts when running tests in parallel
    export TEST_POD_NAME="test-crashloop-pod-${TEST_ID}"

    # Capture output to parse TIMEOUT value
    local failure_output
    failure_output="$("$failure_script" start 2>&1)"
    echo "$failure_output" >&2
    FAILURE_ACTIVE=true

    # Extract TIMEOUT value
    local agent_timeout=300  # default
    if echo "$failure_output" | grep -q "TIMEOUT="; then
        agent_timeout="$(echo "$failure_output" | grep "TIMEOUT=" | cut -d= -f2)"
        echo "Agent timeout set to: ${agent_timeout}s" >&2
    else
        echo "Warning: No TIMEOUT specified, using default: ${agent_timeout}s" >&2
    fi
    echo "" >&2

    # Step 7: Monitor logs for incident detection
    echo "Step 7: Monitoring for incident detection..." >&2
    if ! wait_for_pattern "$nightcrier_log" "incident detected\|creating incident workspace" 120 "incident detection"; then
        echo "Warning: Incident not detected in expected time" >&2
    fi
    echo "" >&2

    # Step 8: Monitor logs for agent execution
    echo "Step 8: Monitoring for agent execution (timeout: ${agent_timeout}s)..." >&2
    if ! wait_for_pattern "$nightcrier_log" "starting agent\|agent container started" 60 "agent start"; then
        echo "Warning: Agent did not start in expected time" >&2
    fi
    echo "" >&2

    # Step 9: Wait for agent completion
    echo "Step 9: Waiting for agent completion (timeout: ${agent_timeout}s)..." >&2
    if ! wait_for_pattern "$nightcrier_log" "agent completed\|investigation complete" "$agent_timeout" "agent completion"; then
        echo "Warning: Agent did not complete in expected time" >&2
    fi
    echo "" >&2

    # Step 10: Stop failure induction
    echo "Step 10: Stopping failure scenario..." >&2
    "$failure_script" stop
    FAILURE_ACTIVE=false
    echo "" >&2

    # Step 11: Stop nightcrier (cleanup will handle it)
    echo "Step 11: Stopping nightcrier..." >&2
    # Let cleanup handle this
    echo "" >&2

    # Step 12: Generate report
    echo "Step 12: Generating report..." >&2
    local output_format="human"
    if [ "$json_output" = "true" ]; then
        output_format="json"
    fi
    generate_report "$TEST_ID" "$LOG_DIR" "$output_format"
    echo "" >&2

    echo "========================================" >&2
    echo "Test Complete: ${TEST_ID}" >&2
    echo "========================================" >&2
}

# Parse command line arguments
main() {
    if [ $# -lt 2 ]; then
        usage
    fi

    local agent="$1"
    local test_type="$2"
    local debug_mode="false"
    local json_output="false"

    # Validate agent
    case "$agent" in
        claude|codex|gemini)
            ;;
        *)
            echo "ERROR: Invalid agent '${agent}'" >&2
            echo "Valid agents: claude, codex, gemini" >&2
            usage
            ;;
    esac

    # Validate test type
    case "$test_type" in
        crashloopbackoff)
            ;;
        *)
            echo "ERROR: Invalid test type '${test_type}'" >&2
            echo "Valid test types: crashloopbackoff" >&2
            usage
            ;;
    esac

    # Parse flags
    shift 2
    while [ $# -gt 0 ]; do
        case "$1" in
            --debug)
                debug_mode="true"
                shift
                ;;
            --json)
                json_output="true"
                shift
                ;;
            *)
                echo "ERROR: Unknown flag '${1}'" >&2
                usage
                ;;
        esac
    done

    # Export test type for failure induction scripts
    export TEST_TYPE="$test_type"

    # Run the test
    run_test "$agent" "$test_type" "$debug_mode" "$json_output"
}

main "$@"
