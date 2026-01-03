# Design: Entrypoint Preflight Checks

## Architecture Decision

The preflight validation will be implemented as a single function (`preflight_check()`) that runs immediately after the startup banner in `main()`, before any expensive operations like git clones or agent path setup.

### Key Design Principles

1. **Fail Fast**: Detect configuration issues in < 5 seconds
2. **Clear Messages**: Every failure includes actionable remediation steps
3. **Ordered Checks**: Run cheap checks first, expensive checks last
4. **Zero Dependencies**: Use only bash built-ins and standard tools (kubectl, test, etc.)
5. **Non-Invasive**: No changes to existing successful execution paths

## Check Ordering Strategy

The checks run in this order (cheap → expensive):

1. **Environment variables** (< 1ms) - bash parameter expansion
2. **File existence** (< 10ms) - filesystem stat operations
3. **File readability** (< 10ms) - test file permissions
4. **API key presence** (< 1ms) - check env vars are non-empty
5. **Output writability** (< 50ms) - create/write/delete test file
6. **kubectl binary** (< 100ms) - run `kubectl version --client`
7. **kubectl connectivity** (< 2s) - test cluster auth

Total preflight time: < 2.2 seconds in worst case, < 0.1 seconds when cluster is unreachable

## Error Message Format

All preflight errors follow this template:

```
========================================
PREFLIGHT CHECK FAILED: <check-name>
========================================
Problem: <what's wrong>
Required: <what's needed>
Fix: <how to fix it>
========================================
```

Example:

```
========================================
PREFLIGHT CHECK FAILED: Kubeconfig Connectivity
========================================
Problem: kubectl auth test failed with error: "Unauthorized"
Required: Valid kubeconfig with non-expired token at /home/agent/.kube/config
Fix: Regenerate kubeconfig token and update the kubeconfig-westeu-cluster1 secret
     kubectl create token kubernetes-triage-readonly -n mcp-system --duration=24h
========================================
```

## Function Structure

```bash
#######################################
# Run all preflight checks before agent execution
# Globals:
#   AGENT_CLI, LLM_MODEL, INCIDENT_ID, OUTPUT_URL_*, API keys
# Arguments:
#   None
# Returns:
#   0 on success, exits with error on failure
#######################################
preflight_check() {
    echo "=========================================="
    echo "Running preflight checks..."
    echo "=========================================="

    local checks_passed=0
    local start_time
    start_time=$(date +%s)

    # Check 1: Environment variables
    check_env_vars || fail_preflight "Environment Variables" "..."
    ((checks_passed++))

    # Check 2: Required files
    check_required_files || fail_preflight "Required Files" "..."
    ((checks_passed++))

    # Check 3: Kubeconfig
    check_kubeconfig || fail_preflight "Kubeconfig" "..."
    ((checks_passed++))

    # Check 4: API keys
    check_api_keys || fail_preflight "API Keys" "..."
    ((checks_passed++))

    # Check 5: Output directory
    check_output_writable || fail_preflight "Output Directory" "..."
    ((checks_passed++))

    # Check 6: kubectl binary
    check_kubectl_binary || fail_preflight "kubectl Binary" "..."
    ((checks_passed++))

    # Check 7: kubectl connectivity
    check_kubectl_connectivity || fail_preflight "kubectl Connectivity" "..."
    ((checks_passed++))

    local end_time
    end_time=$(date +%s)
    local duration=$((end_time - start_time))

    echo "=========================================="
    echo "✓ All preflight checks passed ($checks_passed checks in ${duration}s)"
    echo "=========================================="
    echo ""
}

#######################################
# Fail preflight check with formatted error message
# Arguments:
#   $1 - check name
#   $2 - error message
#######################################
fail_preflight() {
    local check_name="$1"
    local error_msg="$2"

    echo "" >&2
    echo "==========================================" >&2
    echo "PREFLIGHT CHECK FAILED: $check_name" >&2
    echo "==========================================" >&2
    echo "$error_msg" >&2
    echo "==========================================" >&2
    exit 1
}
```

## Individual Check Implementations

### check_env_vars()

```bash
check_env_vars() {
    echo "Checking environment variables..."

    # Call existing validate_env() function
    validate_env 2>&1 | grep -q "required" && return 1

    echo "✓ All required environment variables present"
    return 0
}
```

### check_required_files()

```bash
check_required_files() {
    echo "Checking required files..."

    local required_files=(
        "/home/agent/incident.json"
        "/home/agent/incident_cluster_permissions.json"
        "/home/agent/base-triage-prompt.md"
    )

    for file in "${required_files[@]}"; do
        if [[ ! -f "$file" ]]; then
            echo "✗ Missing file: $file" >&2
            return 1
        fi
        if [[ ! -r "$file" ]]; then
            echo "✗ File not readable: $file" >&2
            return 1
        fi
    done

    echo "✓ All required files present and readable"
    return 0
}
```

### check_kubeconfig()

```bash
check_kubeconfig() {
    echo "Checking kubeconfig..."

    local kubeconfig="/home/agent/.kube/config"

    if [[ ! -f "$kubeconfig" ]]; then
        echo "✗ Kubeconfig not found at $kubeconfig" >&2
        return 1
    fi

    if [[ ! -r "$kubeconfig" ]]; then
        echo "✗ Kubeconfig not readable at $kubeconfig" >&2
        return 1
    fi

    echo "✓ Kubeconfig present and readable"
    return 0
}
```

### check_kubectl_connectivity()

```bash
check_kubectl_connectivity() {
    echo "Checking kubectl connectivity..."

    # Test with timeout to prevent hanging
    if ! timeout 5s kubectl auth can-i get pods --all-namespaces &>/dev/null; then
        local error_output
        error_output=$(kubectl auth can-i get pods --all-namespaces 2>&1 | head -3)
        echo "✗ kubectl auth test failed:" >&2
        echo "$error_output" >&2
        return 1
    fi

    echo "✓ kubectl connectivity verified"
    return 0
}
```

## Trade-offs and Alternatives

### Trade-off: Kubectl Connectivity Check Cost

**Decision**: Include kubectl connectivity test despite 1-2 second cost

**Rationale**:
- Expired tokens are common with time-bounded credentials
- Finding auth failures early saves 20+ seconds of wasted setup
- 2 second overhead is acceptable for improved UX
- Can be made optional later if needed

**Alternative considered**: Skip kubectl test, rely on agent to detect auth failures
- Rejected: Moves error detection too late, harder to troubleshoot

### Trade-off: Check Granularity

**Decision**: Separate checks for each concern (files, kubeconfig, connectivity)

**Rationale**:
- Clear error messages identify exactly what's wrong
- Easier to add/remove individual checks
- Better observability (can see which check failed)

**Alternative considered**: Single monolithic check function
- Rejected: Poor error messages, harder to maintain

### Trade-off: Error Message Verbosity

**Decision**: Include problem + required + fix in every error

**Rationale**:
- Users can self-serve without consulting docs
- Reduces support burden
- Clear remediation steps prevent guessing

**Alternative considered**: Short error messages with error codes
- Rejected: Requires external documentation lookup

## Future Enhancements

### Optional: Skip Kubectl Check for Local Testing

Add environment variable `SKIP_KUBECTL_CHECK=true` to allow running container without cluster access for testing purposes.

### Optional: Preflight Check Metrics

Export preflight check metrics (duration, failures) to OpenTelemetry or similar for observability.

### Optional: Partial Retry Logic

If only kubectl connectivity fails, wait 5s and retry once before failing (handles transient network issues).

## Backward Compatibility

No breaking changes:
- All existing successful runs will continue to work
- Only adds validation, doesn't change execution logic
- Exit codes remain consistent (0 = success, non-zero = failure)
