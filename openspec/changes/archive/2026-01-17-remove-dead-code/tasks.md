# Tasks: Remove Dead Code

## Critical

- [ ] Delete `readIncidentArtifacts()` function (main.go lines 889-1015)
- [ ] Verify no references exist after deletion

## High

- [ ] Remove `kubeconfig` parameter from `processEvent()` function signature
- [ ] Update all call sites of `processEvent()` to remove kubeconfig argument
- [ ] Update log message at line 631 that mentions kubeconfig

## Medium - Unused Flag Variables

- [ ] Remove `mcpEndpoint` variable declaration (line 48)
- [ ] Remove `mcpEndpoint` flag registration (line 78)
- [ ] Remove `workspaceRoot` variable declaration (line 49)
- [ ] Remove `workspaceRoot` flag registration (line 79)
- [ ] Remove `scriptPath` variable declaration (line 50)
- [ ] Remove `scriptPath` flag registration (line 80)
- [ ] Remove `agentTimeout` variable declaration (line 52)
- [ ] Remove `agentTimeout` flag registration (line 82)

## Low - Deprecated Method

- [ ] Replace `IsAzureStorageEnabled()` call in banner with `cfg.ObjectStorage.Type == "azure"`
- [ ] Remove `IsAzureStorageEnabled()` method from config.go

## Low - Unused Struct Fields

- [ ] Remove `LogURLs` field from IncidentSummary struct (slack.go line 80)
- [ ] Remove `UploadFailedInvestigations` field from Config struct (config.go line 223)

## Verification

- [ ] Run `go build ./...` - must compile
- [ ] Run `go test ./...` - all tests must pass
- [ ] Run `go vet ./...` - no warnings
