# Proposal: Rename Codebase to "nightcrier"

## Summary

Rename this project from `kubernetes-mcp-alerts-event-runner` to `nightcrier`, a reference to a town crier that operates "at night" to announce events (in this case, Kubernetes problems).

## Motivation

The current name `kubernetes-mcp-alerts-event-runner` is:
- Verbose and technical
- Hard to remember and type
- Doesn't convey the project's purpose evocatively

The name "nightcrier" captures the essence of the system: a herald that announces problems (faults) in your Kubernetes clusters, often catching issues during off-hours when teams aren't watching.

## Scope

### Files and Locations to Rename

1. **Go Module** (`go.mod`)
   - From: `github.com/rbias/kubernetes-mcp-alerts-event-runner`
   - To: `github.com/rbias/nightcrier`

2. **Import Paths** (all Go files)
   - Update all internal package imports

3. **MCP Client Identity** (`internal/events/client.go`)
   - From: `Name: "kubernetes-mcp-alerts-event-runner"`
   - To: `Name: "nightcrier"`

4. **Binary Name** (`cmd/runner/`)
   - Rename directory from `cmd/runner/` to `cmd/nightcrier/`
   - Binary output will be `nightcrier` instead of `runner`

5. **Configuration References**
   - Update config search path from `/etc/runner/` to `/etc/nightcrier/`
   - Update config file comments and examples

6. **Docker Image Name**
   - From: `k8s-triage-agent`
   - To: `nightcrier-agent`

7. **Documentation**
   - Update `README.md`
   - Update `openspec/project.md`
   - Update all OpenSpec proposal/task files

8. **Container/Makefile**
   - Update `agent-container/Makefile` IMAGE_NAME default

### Items NOT Renamed

- GitHub repository path (user handles this separately)
- External references in kubernetes-mcp-server (if any)

## Impact Assessment

| Area | Impact | Risk |
|------|--------|------|
| Go imports | High - all imports change | Low - sed/replace handles it |
| Config loading | Medium - path changes | Low - additive change |
| Docker images | Medium - new name | Low - rebuild required |
| Documentation | Medium - many files | Low - text changes only |
| OpenSpec files | Low - text updates | Low |

## Backwards Compatibility

- Old configuration file paths (`/etc/runner/`) could be supported as fallback during transition
- Document migration path for existing deployments

## Alternatives Considered

1. **Keep current name**: Rejected - too verbose for daily use
2. **Partial rename (binary only)**: Creates confusion between module and binary names
3. **Alias approach**: Adds complexity without benefits

## Decision

Proceed with full rename to maintain consistency across the codebase.
