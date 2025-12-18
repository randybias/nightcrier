#!/usr/bin/env bash
# Stub agent script for testing
# Reads event context and writes investigation results

WORKSPACE="${1:-.}"
INCIDENT_ID="${2:-unknown}"

echo "=== Stub Agent Starting ==="
echo "Workspace: $WORKSPACE"
echo "Incident ID: $INCIDENT_ID"

# Read and display event context
if [ -f "$WORKSPACE/event.json" ]; then
    echo "=== Event Context ==="
    cat "$WORKSPACE/event.json"
    echo ""
fi

# Write investigation output
mkdir -p "$WORKSPACE/output"
cat > "$WORKSPACE/output/investigation.md" << REPORT
# Investigation Report

**Incident ID:** $INCIDENT_ID
**Timestamp:** $(date -u +"%Y-%m-%dT%H:%M:%SZ")

## Summary
Stub agent completed investigation.

## Findings
- Event context analyzed
- No automated remediation performed

## Recommendations
- Review container configuration
- Check image and command settings
REPORT

echo "=== Investigation complete ==="
echo "Output written to: $WORKSPACE/output/investigation.md"
exit 0
