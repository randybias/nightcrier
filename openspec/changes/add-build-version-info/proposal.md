# Change: Add build time and version to startup banner

## Why

When debugging issues or verifying deployments, it's critical to know which version of the runner is executing and when it was built. Currently the startup banner shows configuration but no build metadata.

## What Changes

- Add build-time variables (Version, BuildTime, GitCommit) injected via `-ldflags` at compile time
- Display version and build time in the startup banner header
- Add `--version` flag to print version info and exit

## Impact

- Affected specs: `walking-skeleton`
- Affected code:
  - `cmd/runner/main.go` - add version variables and display in banner
  - `Makefile` or build script - inject ldflags during build
