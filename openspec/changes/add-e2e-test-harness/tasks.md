## Phase 1: Framework Foundation

### 1.1 Core Abstractions
- [ ] 1.1.1 Define `ClusterProvisioner` interface in Go
- [ ] 1.1.2 Define `ClusterSpec` and `ClusterInfo` types
- [ ] 1.1.3 Define `FailureSpec` and failure metadata schema
- [ ] 1.1.4 Define `TestResult` and result storage interface

### 1.2 Configuration Schema
- [ ] 1.2.1 Design test matrix YAML schema
- [ ] 1.2.2 Implement matrix config parser
- [ ] 1.2.3 Add schema validation for matrix configs
- [ ] 1.2.4 Create example matrix configs (regression, comparative)

### 1.3 Test Orchestrator Shell
- [ ] 1.3.1 Create orchestrator entry point (`cmd/testharness/` or script)
- [ ] 1.3.2 Implement matrix expansion (clusters x failures x agents)
- [ ] 1.3.3 Add test run ID generation and metadata tracking
- [ ] 1.3.4 Implement basic sequential execution loop

## Phase 2: Local Development Support

### 2.1 Stub/Local Provisioner
- [ ] 2.1.1 Implement stub provisioner (returns pre-configured cluster info)
- [ ] 2.1.2 Implement kind provisioner for local testing
- [ ] 2.1.3 Add provisioner selection via config
- [ ] 2.1.4 Document local development workflow

### 2.2 Failure Library Migration
- [ ] 2.2.1 Create `tests/failures/` directory structure
- [ ] 2.2.2 Define failure metadata schema (`metadata.yaml`)
- [ ] 2.2.3 Migrate `crashloopbackoff` to new structure
- [ ] 2.2.4 Add `oom-killed` failure scenario
- [ ] 2.2.5 Add `image-pull-error` failure scenario
- [ ] 2.2.6 Implement failure discovery (scan directory for available failures)

### 2.3 Integration with Existing Harness
- [ ] 2.3.1 Create wrapper for `run-live-test.sh` as execution backend
- [ ] 2.3.2 Parse existing harness output for result extraction
- [ ] 2.3.3 Handle harness failures gracefully
- [ ] 2.3.4 Verify backward compatibility (existing workflow unchanged)

## Phase 3: Result Management

### 3.1 Result Storage
- [ ] 3.1.1 Implement on-disk result storage (JSON files)
- [ ] 3.1.2 Create result directory structure per run
- [ ] 3.1.3 Capture test metadata (timing, status, artifacts)
- [ ] 3.1.4 Implement result loading/querying

### 3.2 Reporting
- [ ] 3.2.1 Implement summary report generator (pass/fail counts)
- [ ] 3.2.2 Add per-test detail reports
- [ ] 3.2.3 Create comparison report for comparative testing objective
- [ ] 3.2.4 Add log aggregation across test runs

### 3.3 Real-Time Monitoring
- [ ] 3.3.1 Implement log tailing for active tests
- [ ] 3.3.2 Add progress indicator for matrix execution
- [ ] 3.3.3 Support `--follow` mode for live output

## Phase 4: k0rdent Integration (Blocked on Interface)

### 4.1 k0rdent Provisioner
- [ ] 4.1.1 **BLOCKED**: Define k0rdent interface contract with team
- [ ] 4.1.2 Implement k0rdent provisioner client
- [ ] 4.1.3 Handle cluster artifact parsing (format TBD)
- [ ] 4.1.4 Add credential extraction from k0rdent response
- [ ] 4.1.5 Implement MCP server endpoint discovery

### 4.2 Multi-Cluster Orchestration
- [ ] 4.2.1 Add parallel cluster provisioning
- [ ] 4.2.2 Implement cluster pooling (reuse across tests)
- [ ] 4.2.3 Add cluster health checks before test execution
- [ ] 4.2.4 Implement graceful teardown on test completion

## Phase 5: Parallel Execution

### 5.1 Concurrency
- [ ] 5.1.1 Add worker pool for parallel test execution
- [ ] 5.1.2 Implement configurable parallelism limits
- [ ] 5.1.3 Add resource contention handling
- [ ] 5.1.4 Ensure result collection is thread-safe

### 5.2 Resilience
- [ ] 5.2.1 Add retry logic for transient failures
- [ ] 5.2.2 Implement test timeout enforcement
- [ ] 5.2.3 Add cleanup on interrupted runs (SIGINT/SIGTERM)
- [ ] 5.2.4 Classify failures (infrastructure vs test vs agent)

## Phase 6: CI Integration

### 6.1 CI Support
- [ ] 6.1.1 Create regression suite matrix config
- [ ] 6.1.2 Add CI-friendly output format (JUnit XML or similar)
- [ ] 6.1.3 Document CI pipeline setup
- [ ] 6.1.4 Add exit codes for CI pass/fail detection

## Validation

- [ ] V1 Run single-test matrix with local/stub provisioner
- [ ] V2 Run multi-failure matrix against existing cluster
- [ ] V3 Run comparative test (same failure, multiple agents)
- [ ] V4 Verify result reports are generated correctly
- [ ] V5 Run full regression suite with k0rdent provisioner (Phase 4+)

## Dependencies

- **Phase 4** blocked on k0rdent interface definition
- **Phase 5** can proceed in parallel with Phase 4
- **Phase 6** requires Phases 1-3 complete
