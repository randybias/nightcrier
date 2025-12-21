# Design Document: Conditional Debug Mode Logging

## Overview

This document details the architectural design for making agent log capture and storage conditional on DEBUG mode, ensuring sensitive data (secrets, API calls, internal reasoning) is not exposed in production incidents.

## Problem Statement

The original log capture implementation always wrote logs and uploaded them to storage, regardless of log level:
- Agent stdout/stderr/combined logs
- Could contain API keys, sensitive commands, internal Claude reasoning

Risk: In production, logs might be accessible to operators who shouldn't see internal debugging details.

Solution: Only capture and store logs when `log_level: debug` is configured.

## Architecture

### Log Capture Flow

```
Executor.ExecuteWithPrompt()
  ├─ Check: e.config.Debug == true?
  │  ├─ YES: Create LogCapture
  │  │   ├─ Create {workspace}/logs/ directory
  │  │   ├─ Open agent-stdout.log
  │  │   ├─ Open agent-stderr.log
  │  │   └─ Open agent-full.log (with timestamps)
  │  └─ NO: Skip LogCapture creation (logCapture = nil)
  ├─ Execute agent command
  │  └─ If logCapture != nil: Capture to files via TeeReader
  └─ Return LogPaths (or empty if no capture)

main.go readIncidentArtifacts()
  ├─ Check for log files in {workspace}/logs/
  ├─ Read only if files exist
  └─ Return IncidentArtifacts with optional AgentLogs

Storage.SaveIncident()
  ├─ Upload artifacts (always)
  ├─ Upload logs (only if AgentLogs populated)
  │  └─ Logs only populated if they were captured
  └─ Generate index.html with all available URLs
```

### Session Archive Capture Flow

```
run-agent.sh after agent execution
  ├─ Check: DEBUG == "true"?
  │  ├─ YES:
  │  │   ├─ Check if ~/.claude exists
  │  │   ├─ tar -czf ${AGENT_HOME}/output/claude-session.tar.gz -C ${HOME} .claude
  │  │   └─ Archive contains complete session history
  │  └─ NO: Skip archive creation
  └─ Archive stays in output/ directory

main.go readIncidentArtifacts()
  ├─ Check for claude-session.tar.gz in {workspace}/output/
  ├─ Read only if file exists
  └─ Return IncidentArtifacts with optional SessionArchive

Storage.SaveIncident()
  └─ Upload session archive (only if it exists)
```

## Key Design Decisions

### 1. Conditional Creation vs Conditional Upload

**Chosen Approach**: Conditional creation in executor

Rationale:
- Simpler implementation (no empty log files to handle)
- Better performance (no unnecessary file I/O in production)
- Clearer semantics (logs only exist if DEBUG mode)

Alternative (not chosen): Always create logs, only upload if DEBUG
- More complex upload logic
- Wastes disk space and I/O
- Harder to reason about state

### 2. LogCapture Modification

Change `NewLogCapture()` signature:
```go
// Before
func NewLogCapture(workspacePath string) (*LogCapture, error)

// After
func NewLogCapture(workspacePath string, debug bool) (*LogCapture, error) {
    if !debug {
        return nil, nil  // No logging in production
    }
    // ... create log files ...
}
```

In `ExecuteWithPrompt()`:
```go
var logCapture *LogCapture
if e.config.Debug {
    logCapture, err = NewLogCapture(workspacePath, e.config.Debug)
    if err != nil {
        return -1, LogPaths{}, err
    }
    defer logCapture.Close()
}

// Continue with execution...
// TeeReader operations check logCapture != nil
```

### 3. Index.html Filtering

Files are automatically included only if they exist in `artifactURLs` map:

```go
// In generateIndexHTML loop
for _, filename := range orderedFiles {
    if url, exists := artifactURLs[filename]; exists {
        // Include in HTML
    }
}
```

In production (non-DEBUG):
- No logs exist
- No session archive exists
- index.html only includes: investigation.html, investigation.md, incident.json, permissions.json

In debug mode:
- All files exist
- index.html includes everything, with logs/session at bottom

### 4. Session Archive Location

Store in `{workspace}/output/` directory:
- Alongside other output files
- Gets read by `readIncidentArtifacts()` in main.go
- Uploaded to storage like other artifacts
- Eventually copied to `logs/` folder in storage structure

### 5. Slack Message (Already Done)

Slack notifications do NOT include log links:
- Removed in previous change
- Message stays focused on investigation results
- Operators see index.html for detailed artifacts

## Security Implications

### Debug Mode (log_level: debug)
- Full logs captured with all internal details
- Session archive with complete Claude reasoning
- Should only be used for development/troubleshooting
- All logs marked as debug artifacts in comments

### Production Mode (log_level: info/warn/error)
- No logs captured
- No session archives
- No secrets or sensitive data exposed
- Clean artifacts: report, incident data, permissions only

## Implementation Sequence

1. **Phase 1: Make log capture conditional**
   - Modify `NewLogCapture()` to accept debug parameter
   - Update executor to check DEBUG before creating LogCapture
   - Update log reading in main.go to handle missing logs

2. **Phase 2: Session archive capture**
   - Modify run-agent.sh to create archive if DEBUG
   - Update artifact reading in main.go
   - Update index.html descriptions

3. **Phase 3: Validation**
   - Test in DEBUG mode (all files present)
   - Test in production mode (no logs)
   - Verify index.html rendering in both modes
   - Verify Slack message stays clean

## Backwards Compatibility

No breaking changes:
- Existing production runs continue to work (no logs captured)
- Existing debug infrastructure unchanged
- Storage upload logic is defensive (only uploads if files exist)
- Log fields in IncidentArtifacts are optional (can be nil/empty)

## Testing Strategy

### Unit Tests (not required - manual preferred)
- Mock file system for conditional creation
- Verify LogCapture returns nil when debug=false
- Verify artifact reading handles missing files

### Integration Tests (runtime validation)

1. **Debug mode test**:
   ```bash
   nightcrier --log-level debug
   # Trigger incident
   # Verify: logs exist, index.html includes logs, session archive present
   ```

2. **Production mode test**:
   ```bash
   nightcrier --log-level info
   # Trigger incident
   # Verify: no logs, no session archive, index.html clean
   ```

3. **Storage tests**:
   - Verify logs uploaded to Azure (DEBUG mode)
   - Verify no logs uploaded (production mode)
   - Verify index.html generation in both modes

## Post-Run Hooks Architecture

### Design Rationale

Rather than hardcoding post-execution tasks, run-agent.sh now features a pluggable post-run hooks system. This allows future features beyond just log capture without increasing complexity.

### Implementation

Simple, straightforward design with minimal code:

```bash
# Post-run hook: Extract Claude session archive (DEBUG mode only)
post_run_extract_claude_session() {
    if [[ "$DEBUG" != "true" ]]; then
        return 0
    fi

    CONTAINER_NAME="nightcrier-agent-${INCIDENT_ID}"
    if [[ -z "$INCIDENT_ID" ]]; then
        return 0
    fi

    echo "DEBUG: Post-run: Extracting Claude session from container: $CONTAINER_NAME" >&2

    # Extract the session directory from the container (single path, no searching)
    if docker cp "$CONTAINER_NAME:/home/agent/.claude" "$WORKSPACE_DIR/claude-session-src" 2>/dev/null; then
        mkdir -p "$WORKSPACE_DIR/logs"
        cd "$WORKSPACE_DIR"
        tar -czf "$WORKSPACE_DIR/logs/claude-session.tar.gz" -C "$WORKSPACE_DIR" claude-session-src
        echo "DEBUG: Post-run: Claude session archive saved to $WORKSPACE_DIR/logs/claude-session.tar.gz" >&2
        rm -rf "$WORKSPACE_DIR/claude-session-src"
        return 0
    else
        echo "DEBUG: Post-run: Could not extract Claude session (session may not exist)" >&2
        return 0
    fi
}

# Execute all post-run hooks
post_run_extract_claude_session
```

### Extensibility

Adding new post-run tasks is straightforward:

```bash
post_run_cleanup_containers() {
    # cleanup logic
}

# Execute all post-run hooks (add new ones here)
post_run_extract_claude_session
post_run_cleanup_containers  # <- new hook
```

### Key Design Decisions

1. **Single known location** - Session always at `/home/agent/.claude` (agent's home directory)
2. **No searching** - Direct docker cp, no fallback logic or filesystem searching
3. **Graceful degradation** - Returns 0 even if session doesn't exist (doesn't block incident)
4. **Minimal code** - 24 lines vs 60 with fallbacks; favors simplicity
5. **Clear error messages** - Debug output for troubleshooting

### Future Post-Run Hooks

Candidates for future post-run hooks:
1. **Container cleanup**: Remove exited containers after session extraction
2. **Metrics collection**: Gather container resource usage
3. **Artifact validation**: Verify logs and archives before completion
4. **Cleanup notifications**: Alert if cleanup fails
5. **Log compression**: Compress large logs before upload

## Implementation Results

### ✅ Successfully Implemented
1. **Conditional log capture** - Works perfectly
   - DEBUG mode: captures stdout, stderr, combined logs with timestamps
   - Production mode: zero overhead, uses io.Discard

2. **Session archive extraction** - Simplified and reliable
   - Single docker cp from known location: `/home/agent/.claude`
   - No searching or fallback logic
   - Graceful handling if session missing
   - Returns 0 in all cases (doesn't fail incident)

3. **Post-run hooks architecture** - Clean, minimal design
   - Simple function-based hook system
   - Easy to extend with new hooks
   - 24-line implementation (minimal code)
   - Ready for future features

4. **Container persistence strategy** - Conditional cleanup
   - Production: `--rm` enabled (auto-cleanup)
   - DEBUG: `--rm` disabled (kept for extraction)
   - Named containers: `nightcrier-agent-${INCIDENT_ID}`

5. **Storage integration** - Both backends working
   - Azure Blob Storage upload
   - Filesystem storage
   - Index.html auto-filtering

6. **Index.html generation** - Smart filtering
   - Only shows files that exist
   - Auto-filters based on mode (DEBUG vs production)

7. **Permissions cleanup** - Storage optimized
   - Removed raw_output field (6KB reduction per incident)

### Key Design Decisions Made

1. **Session location: Single known path**
   - Always at `/home/agent/.claude`
   - No fallback searching
   - No need for multiple attempts
   - Clear and maintainable

2. **Post-run hooks: Minimal design**
   - Function-based, not hardcoded
   - Early returns for clarity
   - Consistent "Post-run:" debug logging
   - 24 lines of clean code

3. **Error handling: Graceful always**
   - Missing session returns 0 (doesn't fail)
   - Debug output for visibility
   - No stack unwinding
   - Incident completes successfully

4. **Container management: Conditional persistence**
   - Smart --rm flag based on mode
   - Named containers for tracking
   - Reliable extraction window

## Future Enhancements

1. **Log compression**: Compress logs before upload if size > threshold
2. **Log rotation**: Keep only last N logs per cluster
3. **Session archive analysis**: Parse session archives to extract structured data
4. **Audit logging**: Separate audit log for what permissions agent had (always captured)
5. **Container cleanup**: Add option to remove exited containers after session extraction
