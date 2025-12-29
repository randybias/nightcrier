# Change: Remove half-baked health endpoint

## Why

The health monitoring HTTP endpoint on port 8080 was introduced prematurely as part of the multi-cluster support work without a proper design for observability. Having an open port with a partial monitoring solution is worse than having no monitoring at all - it creates the illusion of observability without the proper implementation (metrics, alerting, integration with monitoring stacks).

The endpoint should be removed until a comprehensive observability plan is in place.

## What Changes

- **REMOVED**: Health monitoring HTTP server (`internal/health/server.go`)
- **REMOVED**: `--health-port` CLI flag from `cmd/nightcrier/main.go`
- **REMOVED**: `GetHealth()` method from `ConnectionManager`
- **REMOVED**: Health-related imports and startup code from `main.go`
- **REMOVED**: Documentation references to health endpoint in README

## Impact

- Affected specs: none (health monitoring was never spec'd as a requirement)
- Affected code:
  - `internal/health/server.go` - deleted
  - `internal/cluster/manager.go` - `GetHealth()` method removed
  - `cmd/nightcrier/main.go` - health server startup code and flag removed
  - `README.md` - health endpoint documentation removed
- No breaking changes to users (feature was not part of any formal spec)
