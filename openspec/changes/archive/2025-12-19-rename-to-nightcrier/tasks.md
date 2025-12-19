# Implementation Tasks: Rename to "nightcrier"

## Prerequisites
- [ ] Create GitHub repository `nightcrier` (or rename existing repo)

---

## 1. Go Module and Imports
- [x] 1.1 Update `go.mod` module path to `github.com/rbias/nightcrier`
- [x] 1.2 Update imports in `cmd/runner/main.go`
- [x] 1.3 Update imports in `internal/agent/context.go`
- [x] 1.4 Run `go mod tidy` to update `go.sum`
- [x] 1.5 Verify build with `go build ./...`

## 2. Command Directory
- [x] 2.1 Rename `cmd/runner/` to `cmd/nightcrier/`
- [x] 2.2 Update any Makefile or build scripts referencing `cmd/runner`

## 3. MCP Client Identity
- [x] 3.1 Update `internal/events/client.go` Implementation Name from `kubernetes-mcp-alerts-event-runner` to `nightcrier`
- [x] 3.2 Consider updating Version to `1.0.0` to mark the rename milestone

## 4. Configuration System
- [x] 4.1 Update config search path in `internal/config/config.go` from `/etc/runner/` to `/etc/nightcrier/`
- [x] 4.2 Update comments in `configs/config.example.yaml`
- [x] 4.3 Update comments in `configs/config-test.yaml`
- [x] 4.4 Update any references to `runner --config` in documentation

## 5. Docker/Container
- [x] 5.1 Update `agent-container/Makefile` IMAGE_NAME default from `k8s-triage-agent` to `nightcrier-agent`
- [x] 5.2 Update `agent-container/README.md` if it references old names
- [x] 5.3 Update `internal/config/config.go` default for `agent_image` from `k8s-triage-agent:latest` to `nightcrier-agent:latest`
- [x] 5.4 Update `configs/config.example.yaml` agent_image default
- [x] 5.5 Update `configs/config-test.yaml` agent_image value

## 6. Documentation
- [x] 6.1 Update `README.md` title and all references
- [x] 6.2 Update `openspec/project.md` purpose and architecture sections
- [x] 6.3 Update architecture diagram in `openspec/project.md`

## 7. OpenSpec Files
- [x] 7.1 Update `openspec/changes/implement-event-intake/proposal.md` references
- [x] 7.2 Update `openspec/changes/implement-agent-runtime/proposal.md` references
- [x] 7.3 Update `openspec/changes/implement-reporting/proposal.md` references
- [x] 7.4 Update any other OpenSpec files with old name references

## 8. Verification
- [x] 8.1 Run `go build ./cmd/nightcrier`
- [x] 8.2 Run all tests with `go test ./...`
- [x] 8.3 Verify config loading works with new paths
- [ ] 8.4 Build Docker image with new name
- [ ] 8.5 Run end-to-end test with kubernetes-mcp-server
- [x] 8.6 Grep codebase for any remaining old name references: `rg "kubernetes-mcp-alerts-event-runner|k8s-triage-agent" --type-not binary`

## 9. Cleanup
- [ ] 9.1 Update git remote URL if repository was renamed
- [ ] 9.2 Tag release as `v1.0.0-nightcrier` or similar to mark rename
- [ ] 9.3 Archive this change proposal

---

## Parallelization Notes

Tasks 1-6 can be executed in parallel after task 1.1 (go.mod change) is complete, as imports must be updated first.

Task 8 (verification) must be done after all other tasks.
