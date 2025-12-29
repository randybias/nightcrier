## 1. Remove Health Endpoint Code

- [ ] 1.1 Delete `internal/health/server.go`
- [ ] 1.2 Remove `GetHealth()` method from `internal/cluster/manager.go`
- [ ] 1.3 Remove `healthPort` variable and `--health-port` flag from `cmd/nightcrier/main.go`
- [ ] 1.4 Remove health server import and startup code from `cmd/nightcrier/main.go`
- [ ] 1.5 Remove health endpoint documentation from `README.md`

## 2. Clean Up References

- [ ] 2.1 Remove health-related references from `openspec/changes/add-multi-cluster-support/design.md` (Phase 4 references)
- [ ] 2.2 Remove health-related tasks from `openspec/changes/add-multi-cluster-support/tasks.md`

## 3. Verification

- [ ] 3.1 Build passes: `go build ./...`
- [ ] 3.2 Tests pass: `go test ./...`
- [ ] 3.3 No stray references to health endpoint remain (grep verification)
