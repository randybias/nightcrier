## 1. Implementation

- [ ] 1.1 Add kubeconfig struct types to `internal/cluster/kubeconfig.go` for YAML parsing
- [ ] 1.2 Add `validateKubeconfigContent()` function that parses and validates kubeconfig
- [ ] 1.3 Integrate validation into `ClusterConfig.Validate()` after file existence check
- [ ] 1.4 Add clear error messages for each validation failure case

## 2. Testing

- [ ] 2.1 Add unit tests for valid kubeconfig parsing
- [ ] 2.2 Add unit tests for invalid YAML detection
- [ ] 2.3 Add unit tests for missing required fields (clusters, users, contexts)
- [ ] 2.4 Add unit tests for invalid context reference
- [ ] 2.5 Add test fixtures in `internal/cluster/testdata/`
