#!/usr/bin/env bash
# log-monitor.sh - Monitor logs for expected patterns with timeout
#
# Functions for tailing and monitoring log files with pattern matching and timeouts

# Wait for a specific pattern in a log file
# Usage: wait_for_pattern <log_file> <pattern> <timeout_seconds> [description]
wait_for_pattern() {
    local log_file="$1"
    local pattern="$2"
    local timeout="${3:-60}"
    local description="${4:-pattern}"

    echo "Waiting for ${description} in ${log_file} (timeout: ${timeout}s)..." >&2

    local elapsed=0
    local interval=1

    while [ $elapsed -lt "$timeout" ]; do
        if [ -f "$log_file" ] && grep -q "$pattern" "$log_file"; then
            echo "Found ${description} after ${elapsed}s" >&2
            return 0
        fi
        sleep "$interval"
        elapsed=$((elapsed + interval))
    done

    echo "ERROR: Timeout waiting for ${description} after ${timeout}s" >&2
    return 1
}

# Tail a log file until a pattern appears or timeout
# Usage: tail_until_pattern <log_file> <pattern> <timeout_seconds> [description]
tail_until_pattern() {
    local log_file="$1"
    local pattern="$2"
    local timeout="${3:-60}"
    local description="${4:-pattern}"

    echo "Tailing ${log_file} for ${description} (timeout: ${timeout}s)..." >&2

    # Use timeout command with grep to monitor the log
    if timeout "$timeout" tail -f "$log_file" 2>/dev/null | grep -q -m 1 "$pattern"; then
        echo "Found ${description}" >&2
        return 0
    else
        echo "ERROR: Timeout waiting for ${description} after ${timeout}s" >&2
        return 1
    fi
}

# Wait for log file to be created
# Usage: wait_for_log_file <log_file> <timeout_seconds>
wait_for_log_file() {
    local log_file="$1"
    local timeout="${2:-30}"

    echo "Waiting for log file ${log_file} (timeout: ${timeout}s)..." >&2

    local elapsed=0
    local interval=1

    while [ $elapsed -lt "$timeout" ]; do
        if [ -f "$log_file" ]; then
            echo "Log file created after ${elapsed}s" >&2
            return 0
        fi
        sleep "$interval"
        elapsed=$((elapsed + interval))
    done

    echo "ERROR: Log file ${log_file} not created after ${timeout}s" >&2
    return 1
}
