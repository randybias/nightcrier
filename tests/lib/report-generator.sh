#!/usr/bin/env bash
# report-generator.sh - Generate test reports from logs and artifacts
#
# Functions for analyzing test results and generating reports

# Generate a test report from logs and artifacts
# Usage: generate_report <test_id> <log_dir> <output_format>
#   output_format: "json" or "human"
generate_report() {
    local test_id="$1"
    local log_dir="$2"
    local output_format="${3:-human}"

    local nightcrier_log="${log_dir}/nightcrier.log"
    local meta_file="${log_dir}/test.meta"
    local report_file="${log_dir}/report.json"

    # Extract metadata
    local agent=""
    local test_type=""
    local start_time=""

    if [ -f "$meta_file" ]; then
        # shellcheck disable=SC1090
        source "$meta_file"
        agent="$TEST_AGENT"
        test_type="$TEST_TYPE"
        start_time="$TEST_START_TIME"
    fi

    # Analyze logs for key events
    local incident_detected=false
    local agent_started=false
    local agent_completed=false
    local failure_induced=false

    if [ -f "$nightcrier_log" ]; then
        grep -q "subscribed to fault events" "$nightcrier_log" && failure_induced=true
        grep -q "incident detected\|creating incident workspace" "$nightcrier_log" && incident_detected=true
        grep -q "starting agent\|agent container started" "$nightcrier_log" && agent_started=true
        grep -q "agent completed\|investigation complete" "$nightcrier_log" && agent_completed=true
    fi

    # Determine overall status
    local status="UNKNOWN"
    if [ "$agent_completed" = true ]; then
        status="PASS"
    elif [ "$agent_started" = true ]; then
        status="INCOMPLETE"
    elif [ "$incident_detected" = true ]; then
        status="AGENT_FAILED"
    elif [ "$failure_induced" = true ]; then
        status="NO_DETECTION"
    else
        status="FAILED"
    fi

    # Generate JSON report
    cat > "$report_file" <<EOF
{
  "test_id": "$test_id",
  "agent": "$agent",
  "test_type": "$test_type",
  "start_time": "$start_time",
  "end_time": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "status": "$status",
  "events": {
    "failure_induced": $failure_induced,
    "incident_detected": $incident_detected,
    "agent_started": $agent_started,
    "agent_completed": $agent_completed
  },
  "logs": {
    "nightcrier_log": "$nightcrier_log"
  }
}
EOF

    # Output based on format
    if [ "$output_format" = "json" ]; then
        cat "$report_file"
    else
        # Human-readable format
        cat <<EOF

========================================
Test Report: $test_id
========================================

Agent:          $agent
Test Type:      $test_type
Status:         $status
Start Time:     $start_time
End Time:       $(date -u +%Y-%m-%dT%H:%M:%SZ)

Events:
  - Failure Induced:    $failure_induced
  - Incident Detected:  $incident_detected
  - Agent Started:      $agent_started
  - Agent Completed:    $agent_completed

Logs:
  - Nightcrier: $nightcrier_log

Report saved to: $report_file

========================================
EOF
    fi
}
