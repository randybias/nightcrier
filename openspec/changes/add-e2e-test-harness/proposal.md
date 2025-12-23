# Change: Add Comprehensive E2E Test Harness

## Why

The existing live test harness (`tests/run-live-test.sh`) validates single-agent, single-failure scenarios against a pre-existing cluster. To ensure nightcrier works reliably across diverse Kubernetes configurations, we need a comprehensive end-to-end test harness that can:

1. Orchestrate multiple clusters with different configurations
2. Run test matrices across agents, failures, and cluster types
3. Enable regression testing for major changes
4. Support comparative analysis of agent/prompt performance

This harness will be critical for:
- Catching regressions before they reach production
- Validating new agent integrations
- Comparing triage quality across different LLM backends
- Ensuring compatibility with various Kubernetes distributions

## What Changes

### New Capabilities

- **Test Matrix Framework** - Define and execute test matrices (clusters x failures x agents)
- **Cluster Provisioning Interface** - Abstract interface for cluster lifecycle management (k0rdent integration TBD)
- **Test Run Orchestration** - Coordinate parallel test execution across multiple clusters
- **Result Aggregation** - Collect, store, and compare results across test runs
- **Failure Library** - Extensible catalog of inducible failures with metadata

### Extensions to Existing

- **live-testing** - Current harness becomes one execution backend; framework wraps it

## Impact

- Affected specs: `live-testing` (extended), new `e2e-test-harness` capability
- Affected code: `tests/` directory expansion, possible new `testing/` package
- External dependencies: k0rdent integration (interface TBD)

## Objectives

This harness supports three primary testing objectives:

| Objective | Description | Scope |
|-----------|-------------|-------|
| **Permutation Testing** | E2E validation of cluster x incident combinations | Full matrix |
| **Regression Testing** | Subset of permutation tests for CI/major changes | Curated subset |
| **Comparative Testing** | Same incident across agents/prompts for quality assessment | Single incident, multiple configs |

## Open Questions

### Cluster Provisioning Interface
- **Status**: TBD - team member handling k0rdent integration
- **Known constraints**: Uses CAPI and Project Sveltos, templates are repeatable
- **Next step**: Negotiate interface contract with k0rdent team

### Cluster Configuration Artifact
- **Status**: TBD - format to be negotiated
- **Options considered**: YAML manifest, directory structure, env file
- **Next step**: Define contract once k0rdent interface is clearer

### Repository Location
- **Status**: Under consideration
- **Question**: Should the test harness live in nightcrier repo or separate repo?
- **Factors**: Complexity growth, CI coupling, reusability across projects
