# Proposal: Remove Dead Code

## Summary

Remove unused functions, parameters, and flag variables identified during codebase audit. This cleanup reduces maintenance burden and clarifies what is actually implemented.

## Motivation

A codebase audit identified dead code including:
- A 127-line function that is never called
- Flag variables registered but never used
- An unused function parameter

Dead code creates confusion about what's implemented vs planned, increases cognitive load during maintenance, and can mask real bugs.

## Changes

### Critical: Remove Dead Function

**`cmd/nightcrier/main.go:889-1015`** - `readIncidentArtifacts()`
- 127 lines, never called anywhere
- Leftover from earlier architecture before K8s executor refactor
- Artifact handling now delegated to K8s executor and processor

### High: Remove Unused Parameter

**`cmd/nightcrier/main.go:600`** - `kubeconfig` param in `processEvent()`
- Parameter accepted but never used in function body
- Only appears in a log message string, not actual logic
- Remove from function signature and all call sites

### Medium: Remove Unused Flag Variables

| Variable | File | Lines |
|----------|------|-------|
| `mcpEndpoint` | cmd/nightcrier/main.go | 48, 78 |
| `workspaceRoot` | cmd/nightcrier/main.go | 49, 79 |
| `scriptPath` | cmd/nightcrier/main.go | 50, 80 |
| `agentTimeout` | cmd/nightcrier/main.go | 52, 82 |

These flags are registered with cobra but the variables are never referenced after parsing. Remove both the variable declarations and the flag registrations.

### Low: Remove Deprecated Method

**`internal/config/config.go:722`** - `IsAzureStorageEnabled()`
- Marked deprecated in code comment
- Only used in `printStartupBanner()`
- Replace banner usage with direct `ObjectStorage.Type == "azure"` check, then remove method

### Low: Remove Unused Struct Fields

**`internal/reporting/slack.go:80`** - `LogURLs` in IncidentSummary
- Field defined but never assigned or read
- Not referenced in any spec

**`internal/config/config.go:223`** - `UploadFailedInvestigations` in Config
- Field defined but never referenced in any logic
- Not in any spec

## NOT Removing (Specified but Unimplemented)

The following config fields are **specified in configuration spec** but not yet implemented in logic. These are NOT dead code - they are unimplemented features:

- `AdditionalPrompt` - Specified at configuration/spec.md:59-63
- `AllowedTools` - Specified at configuration/spec.md:78

These should be implemented, not removed.

## Scope

Files modified:
- `cmd/nightcrier/main.go` - Remove function, parameter, flags
- `internal/config/config.go` - Remove deprecated method
- `internal/reporting/slack.go` - Remove unused struct field

## Out of Scope

- Implementing unimplemented spec features (AdditionalPrompt, AllowedTools)
- Adding new functionality
- Refactoring working code

## Risk Assessment

**Low risk** - All items verified as unused via grep. No behavioral changes expected.

Verification approach:
1. Remove code
2. Compile successfully (`go build ./...`)
3. Run existing tests (`go test ./...`)
4. Run vet (`go vet ./...`)
