# Tasks: Automate Kubernetes Bootstrap

## Phase 1: Bootstrap Package Foundation

### 1.1 Create Bootstrap Package Structure
- [ ] Create `internal/bootstrap/` directory
- [ ] Create `bootstrap.go` with `Bootstrapper` interface and `Manager` implementation
- [ ] Create `types.go` with `Config`, `ClusterConfig`, and `Result` types
- [ ] Add basic constructor `NewManager(kubeClient, config)`

### 1.2 Implement Namespace Bootstrap
- [ ] Create `namespace.go` with namespace creation logic
- [ ] Implement `ensureNamespace(ctx, name)` with existence check
- [ ] Add Namespace creation with proper labels (`app=nightcrier`)
- [ ] Handle already-exists gracefully (idempotent)
- [ ] Add unit tests for namespace operations

### 1.3 Implement RBAC Bootstrap
- [ ] Create `rbac.go` with RBAC resource creation
- [ ] Implement `ensureServiceAccount(ctx)` with existence check
- [ ] Implement `ensureRole(ctx)` with proper permissions (Jobs, ConfigMaps, Pods, Secrets)
- [ ] Implement `ensureRoleBinding(ctx)` linking ServiceAccount to Role
- [ ] Handle already-exists gracefully (idempotent)
- [ ] Add unit tests for RBAC operations

### 1.4 Implement API Keys Secret Bootstrap
- [ ] Create `secrets.go` with Secret creation logic
- [ ] Implement `ensureAPIKeysSecret(ctx, apiKeys)` with existence check
- [ ] Validate at least one API key is non-empty
- [ ] Create Secret with all three keys (anthropic, openai, gemini)
- [ ] Handle already-exists gracefully (skip, don't update)
- [ ] Add unit tests for API keys Secret operations

### 1.5 Implement Kubeconfig Secrets Bootstrap
- [ ] Add `ensureKubeconfigSecret(ctx, clusterName, kubeconfigPath)` to `secrets.go`
- [ ] Validate kubeconfig file exists and is readable
- [ ] Read file contents into memory
- [ ] Create Secret with name `kubeconfig-{cluster-name}` and key `config`
- [ ] Handle already-exists gracefully (skip, don't update)
- [ ] Add unit tests for kubeconfig Secret operations

## Phase 2: Integration with Main Application

### 2.1 Add Bootstrap to Startup Sequence
- [ ] Modify `cmd/nightcrier/main.go` to call bootstrap after config load
- [ ] Create Kubernetes client using kubeconfig precedence (KUBECONFIG env > config > default)
- [ ] Initialize bootstrap Manager with client and config
- [ ] Call `Bootstrap(ctx)` before initializing other components
- [ ] Handle bootstrap errors by exiting with status code 1

### 2.2 Wire Configuration to Bootstrap
- [ ] Pass `cfg.K8sNamespace` to bootstrap Manager
- [ ] Pass API keys from config/env to bootstrap Manager:
  - `cfg.AnthropicAPIKey`
  - `cfg.OpenAIAPIKey`
  - `cfg.GeminiAPIKey`
- [ ] Map `cfg.Clusters` array to bootstrap ClusterConfig format
- [ ] Extract `triage.kubeconfig` path for each enabled cluster

### 2.3 Add Bootstrap Logging
- [ ] Log info message when starting bootstrap: "Starting Kubernetes bootstrap..."
- [ ] Log debug messages for existing resources: "Namespace already exists, skipping"
- [ ] Log info messages for created resources: "Created namespace nightcrier"
- [ ] Log summary on completion: "Kubernetes bootstrap complete: 2 created, 3 existing"
- [ ] Log error with remediation guidance on failure

### 2.4 Improve Error Messages
- [ ] Add permission error handler with kubectl apply suggestion
- [ ] Add file not found error with full path display
- [ ] Add file permission error with chmod guidance
- [ ] Add API connection error with server address
- [ ] Add invalid YAML error with parse details

## Phase 3: Testing

### 3.1 Unit Tests
- [ ] Test `ensureNamespace` with mock client (create, exists, error cases)
- [ ] Test `ensureServiceAccount`, `ensureRole`, `ensureRoleBinding` with mocks
- [ ] Test `ensureAPIKeysSecret` with various API key combinations
- [ ] Test `ensureKubeconfigSecret` with file read scenarios
- [ ] Test bootstrap Manager orchestration (happy path, partial existing, full existing)
- [ ] Test error handling (permission denied, file missing, API unavailable)
- [ ] Verify idempotency (running bootstrap twice creates resources once)

### 3.2 Integration Tests
- [ ] Create `internal/bootstrap/integration_test.go` with kind cluster setup
- [ ] Test bootstrap against empty kind cluster (all resources created)
- [ ] Test bootstrap with manually pre-created namespace (skip namespace, create rest)
- [ ] Test bootstrap with all resources existing (skip all)
- [ ] Test with multiple clusters configured (multiple kubeconfig Secrets)
- [ ] Test with invalid kubeconfig path (should fail)
- [ ] Test with insufficient permissions (should fail with clear error)

### 3.3 Manual Testing
- [ ] Test local kind cluster startup (should bootstrap automatically)
- [ ] Test remote cluster connection (via kubeconfig_path)
- [ ] Test with `KUBECONFIG` environment variable override
- [ ] Test startup with resources already present (should skip)
- [ ] Test startup with missing kubeconfig file (should fail clearly)
- [ ] Verify Secrets contain correct data (base64 decode and check)

## Phase 4: Documentation and Cleanup

### 4.1 Update Documentation
- [ ] Add "Kubernetes Bootstrap" section to main README
- [ ] Document required Kubernetes permissions for bootstrap
- [ ] Add troubleshooting section for common bootstrap errors
- [ ] Update deployment guide with automatic setup flow
- [ ] Document manual setup alternative (for air-gapped environments)

### 4.2 Simplify dev-setup.sh
- [ ] Remove `apply_manifests()` function (nightcrier does this now)
- [ ] Remove `create_secrets()` function (nightcrier does this now)
- [ ] Remove `create_kubeconfig_secret()` function (nightcrier does this now)
- [ ] Keep only kind cluster creation and image loading
- [ ] Update script comments to reflect reduced scope
- [ ] Update verification checks to rely on nightcrier bootstrap

### 4.3 Update Configuration Examples
- [ ] Add comments in `config.example.yaml` explaining bootstrap behavior
- [ ] Document kubeconfig_path vs triage.kubeconfig distinction
- [ ] Add note that API keys can be environment variables or config file
- [ ] Clarify that kubeconfig_path is for admin access (bootstrap), triage.kubeconfig is for agents

### 4.4 Add Migration Notes
- [ ] Document that existing deployments don't need migration (idempotent)
- [ ] Add note that manually created resources will not be modified
- [ ] Explain precedence: manual resources > bootstrap (we never overwrite)
- [ ] Add troubleshooting for "permission denied" in existing clusters

## Phase 5: Validation and Review

### 5.1 Code Review Preparation
- [ ] Run `go vet ./internal/bootstrap/...`
- [ ] Run `golangci-lint run ./internal/bootstrap/...`
- [ ] Verify all unit tests pass: `go test ./internal/bootstrap/...`
- [ ] Verify integration tests pass: `go test -tags=integration ./internal/bootstrap/...`
- [ ] Check code coverage: `go test -cover ./internal/bootstrap/...` (target: >80%)

### 5.2 OpenSpec Validation
- [ ] Run `openspec validate automate-k8s-bootstrap --strict`
- [ ] Resolve any validation errors
- [ ] Verify all requirements have scenarios
- [ ] Verify all tasks are linked to requirements
- [ ] Check that spec delta relationships are clear

### 5.3 End-to-End Validation
- [ ] Start fresh kind cluster
- [ ] Run nightcrier with default config
- [ ] Verify namespace created: `kubectl get ns nightcrier`
- [ ] Verify RBAC created: `kubectl get sa,role,rolebinding -n nightcrier`
- [ ] Verify Secrets created: `kubectl get secrets -n nightcrier`
- [ ] Verify triage Job can be created successfully
- [ ] Check logs for clear bootstrap messages

## Dependencies

- **Phase 2 depends on Phase 1**: Bootstrap package must exist before integration
- **Phase 3 depends on Phase 2**: Can't test integration until wired into main
- **Phase 4 can run in parallel with Phase 3**: Documentation doesn't block testing
- **Phase 5 depends on Phases 1-4**: Final validation requires complete implementation

## Estimated Effort

- Phase 1: 8 hours (core bootstrap logic)
- Phase 2: 3 hours (integration)
- Phase 3: 6 hours (comprehensive testing)
- Phase 4: 2 hours (docs and cleanup)
- Phase 5: 2 hours (validation)

**Total: ~21 hours**

## Success Criteria

1. ✅ Nightcrier starts successfully on empty kind cluster without manual setup
2. ✅ All required Kubernetes resources are created automatically
3. ✅ Bootstrap is idempotent (running twice doesn't fail or duplicate)
4. ✅ Remote cluster deployment works via kubeconfig configuration
5. ✅ Clear error messages for permission failures and missing files
6. ✅ All unit and integration tests pass
7. ✅ `openspec validate` passes with no errors
8. ✅ dev-setup.sh is simplified (50%+ reduction in lines)
