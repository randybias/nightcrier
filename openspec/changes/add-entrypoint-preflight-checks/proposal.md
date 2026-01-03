# Proposal: Add Entrypoint Preflight Checks

## Why

Agent Jobs are currently failing late in the execution flow when configuration is invalid, wasting compute resources and making troubleshooting difficult. Recent production issues (expired kubeconfig tokens) took 20+ seconds to detect, after expensive setup operations like git clone. This proposal adds comprehensive preflight validation that runs in < 3 seconds and provides clear, actionable error messages for configuration problems.

**User Value**: Operators get immediate feedback on configuration errors instead of waiting through failed setup operations. Failed Jobs consume fewer resources and provide clear remediation guidance.

**Technical Value**: Clear separation between configuration validation (fast), setup operations (expensive), and agent execution (very expensive) makes system behavior more predictable and easier to troubleshoot.

## Problem Statement

The nc-agent-runner entrypoint script (`nc-agent-runner/entrypoint.sh`) currently validates environment variables late in the execution flow, after performing setup operations like cloning skills and configuring agent paths. This creates several problems:

1. **Wasted Resources**: The container performs expensive operations (git clone, filesystem setup) before discovering it lacks required credentials
2. **Poor Error Context**: When failures occur during agent execution, it's unclear whether the issue is missing configuration or an actual runtime problem
3. **Delayed Failure Detection**: Jobs appear to be "running" for several seconds before failing, making it harder to diagnose configuration issues
4. **Silent Failures**: Some missing resources (like kubeconfig, incident files) aren't validated at all and only cause failures deep in agent execution

The recent discovery of expired kubeconfig tokens highlights this issue - the container successfully started, performed all setup, and only failed when the agent tried to execute kubectl commands.

## Proposed Solution

Add comprehensive preflight validation at the very beginning of the entrypoint script that:

1. **Validates all required environment variables** (already exists but runs too late)
2. **Checks for required files and resources**:
   - Kubeconfig exists at `/home/agent/.kube/config`
   - Incident data file exists at `/home/agent/incident.json`
   - Cluster permissions file exists (if provided)
   - Base triage prompt file exists (if provided)
3. **Tests kubectl connectivity** to ensure kubeconfig is valid and not expired
4. **Validates API keys are present** (at least one) for the selected agent
5. **Fails fast** with clear error messages before any expensive operations

## Success Criteria

1. Preflight validation runs immediately after the startup banner
2. All checks complete in < 2 seconds (before git clone or other setup)
3. Clear, actionable error messages identify exactly what's missing
4. Jobs fail within 5 seconds when configuration is invalid (vs 20+ seconds currently)
5. All existing successful runs continue to work unchanged

## Impact Assessment

### User Experience
- **Positive**: Faster feedback on configuration errors
- **Positive**: Clear error messages guide troubleshooting
- **Positive**: Less wasted compute time on misconfigured jobs

### System Behavior
- **Neutral**: No changes to successful execution paths
- **Positive**: Failed jobs consume fewer resources
- **Positive**: Clearer distinction between config errors vs runtime errors

### Implementation Risk
- **Low**: Purely additive validation, doesn't change existing logic
- **Low**: Can be implemented incrementally (start with critical checks)

## Alternatives Considered

### Alternative 1: Validate in Go code before Job creation
- **Pros**: Catch errors even earlier, never create invalid Jobs
- **Cons**: Go code can't validate kubeconfig expiry without complex token parsing
- **Cons**: Separates validation logic from execution environment
- **Decision**: Complement, not replace - both layers provide value

### Alternative 2: Add health probe to container
- **Pros**: Kubernetes-native failure detection
- **Cons**: Startup probes add complexity and delay startup
- **Cons**: Doesn't provide clear error messages to logs
- **Decision**: Not suitable - we need fast-fail with clear messages

## Dependencies

None. This is a self-contained improvement to the entrypoint script.

## Open Questions

1. Should kubectl connectivity test be required or optional (controlled by env var)?
   - **Recommendation**: Required for Jobs that mount kubeconfig, skipped for local testing
2. Should we test writable access to output directory?
   - **Recommendation**: Yes, quick filesystem test before proceeding
3. How detailed should API key validation be (present vs format check)?
   - **Recommendation**: Just check presence, not format (format varies by provider)
