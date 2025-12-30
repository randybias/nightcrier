# Proposal: add-preflight-validation

## Problem Statement

Nightcrier currently has validation scattered across multiple locations:
- Config validation in `internal/config/config.go` (Validate() method)
- File existence checks in `cmd/nightcrier/main.go` (system prompt file)
- Implicit failures during component initialization (storage, state store, K8s client)

This scattered approach has several problems:

1. **Silent failures**: Missing files or misconfigured paths may not fail until agents run (e.g., agent ran without system prompt)
2. **Inconsistent error handling**: Some checks warn and continue, others fail hard
3. **Poor discoverability**: No clear place to add new validation checks
4. **Delayed failures**: Some issues only surface after partial initialization

**Recent incident**: Agent executed without system prompt because the configured file path was incorrect (`triage-system-prompt.md` vs `base-triage-prompt.md`). The check only warned and set the path to empty, causing agents to run with no instructions.

## Proposed Solution

Implement a **centralized pre-flight validation system** that runs before any component initialization:

1. Create a dedicated `internal/preflight` package with a `Validator` interface
2. Consolidate all critical checks into a single validation phase
3. Execute pre-flight validation immediately after config load, before any components initialize
4. Provide clear, actionable error messages with remediation steps
5. Document when and how to add new validation checks

### Validation Categories

1. **Critical Files** (must exist and be readable)
   - Agent system prompt file
   - Triage kubeconfig files (per cluster)
   - Migration directories for state storage
   - Skills directories (if not auto-cloned)

2. **Configuration Consistency** (values must be valid and compatible)
   - API keys (at least one present)
   - Storage URLs (valid format)
   - Numeric ranges (already in config.Validate())
   - Enum values (already in config.Validate())

3. **External Dependencies** (services must be reachable)
   - Kubernetes cluster connectivity (optional: can defer to bootstrap)
   - Object storage endpoint (optional: can defer to first use)

### Scope

**In scope:**
- Create `internal/preflight` package with validator interface
- Implement file existence validators
- Implement configuration consistency validators
- Integrate pre-flight checks into main.go startup sequence
- Document how to add new validators
- Move existing scattered checks into pre-flight system

**Out of scope:**
- Network connectivity checks (defer to component init for better error context)
- Deep validation of file contents (defer to component that uses the file)
- Migration validation beyond directory existence

## Benefits

1. **Fail fast**: Catch configuration errors before any work begins
2. **Clear errors**: Single place with consistent, helpful error messages
3. **Maintainability**: Easy to find and add new validation checks
4. **Documentation**: Self-documenting validation requirements
5. **Testing**: Centralized validation logic is easier to unit test

## Trade-offs

### Pros
- Prevents wasted work and confusing partial failures
- Clear separation between validation and initialization
- Easier to add new checks
- Better error messages with remediation guidance

### Cons
- Adds ~50ms to startup time (negligible)
- Some duplication with existing config.Validate()
- May delay errors that could be caught later (but with worse context)

## Open Questions

1. Should pre-flight checks be bypassable with a flag (e.g., `--skip-preflight`)? **Recommendation: No, these are critical checks**
2. Should we validate network connectivity to external services? **Recommendation: No, defer to component init**
3. Should we validate Kubernetes RBAC permissions during preflight? **Recommendation: No, bootstrap handles this**

## Success Criteria

1. All critical file paths validated before component initialization
2. Clear, actionable error messages when validation fails
3. Developer documentation for adding new validators
4. Application fails fast on startup with misconfiguration
5. No silent failures or degraded operation mode
