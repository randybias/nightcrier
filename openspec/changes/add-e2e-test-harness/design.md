## Context

Nightcrier needs comprehensive E2E testing to validate triage agents work correctly across:
- Different Kubernetes distributions and configurations
- Various failure types and severities
- Multiple AI agent backends with different prompts

The current `tests/` harness handles single-cluster, single-failure testing. This design extends that foundation into a configurable test matrix framework.

### Stakeholders
- Nightcrier developers (regression testing)
- k0rdent team (cluster provisioning integration)
- AI/ML team (agent comparison testing)

### Constraints
- k0rdent integration interface is TBD
- Cluster artifact format is TBD
- Must not break existing live test scripts
- Should be runnable locally for development

## Goals / Non-Goals

### Goals
- Define clean abstractions for cluster provisioning (pluggable backends)
- Enable test matrix definition via configuration
- Support all three testing objectives (permutation, regression, comparative)
- Aggregate results for analysis and trend tracking
- Enable real-time debugging during test runs
- Preserve existing live-test workflow as fallback

### Non-Goals
- Implementing k0rdent provisioner (team member handles)
- Building a web UI for test management (future consideration)
- Automated performance benchmarking (out of scope)
- Multi-tenancy or shared test infrastructure

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Test Matrix Definition                        │
│                    (YAML: clusters x failures x agents)              │
└─────────────────────────────────┬───────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────────┐
│                        Test Orchestrator                             │
│  - Parses matrix definition                                          │
│  - Manages test run lifecycle                                        │
│  - Coordinates parallel execution                                    │
│  - Aggregates results                                                │
└────────┬────────────────────────┬────────────────────────┬──────────┘
         │                        │                        │
         ▼                        ▼                        ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│ Cluster Manager │    │ Failure Library │    │ Test Executor   │
│                 │    │                 │    │                 │
│ - Provisioner   │    │ - Failure defs  │    │ - Nightcrier    │
│   interface     │    │ - Induction     │    │   lifecycle     │
│ - k0rdent hook  │    │   scripts       │    │ - Log capture   │
│ - Local/kind    │    │ - Cleanup       │    │ - Result parse  │
│   (fallback)    │    │ - Metadata      │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                        │                        │
         └────────────────────────┼────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────────┐
│                        Result Store                                  │
│  - Test run metadata                                                 │
│  - Per-test results (pass/fail, artifacts)                          │
│  - Comparison views                                                  │
│  - Trend data (optional)                                             │
└─────────────────────────────────────────────────────────────────────┘
```

## Decisions

### D1: Pluggable Cluster Provisioner Interface

**Decision**: Define abstract `ClusterProvisioner` interface; implementations are pluggable.

**Interface** (conceptual):
```go
type ClusterProvisioner interface {
    // Provision creates a cluster and returns connection info
    Provision(ctx context.Context, spec ClusterSpec) (*ClusterInfo, error)

    // Teardown destroys a cluster
    Teardown(ctx context.Context, clusterID string) error

    // Status checks cluster health
    Status(ctx context.Context, clusterID string) (ClusterStatus, error)
}

type ClusterInfo struct {
    ID            string
    Kubeconfig    []byte           // or path
    MCPEndpoint   string           // kubernetes-mcp-server endpoint
    Metadata      map[string]string // cluster-specific info
}
```

**Rationale**: k0rdent integration is TBD; clean interface allows development to proceed independently.

**Alternatives considered**:
- Direct k0rdent API calls - rejected (too coupled, interface unknown)
- Shell script hooks only - rejected (insufficient structure for complex workflows)

### D2: Configuration-Driven Test Matrices

**Decision**: Test matrices defined in YAML configuration files.

**Example**:
```yaml
# test-matrices/regression-suite.yaml
name: regression-suite
description: Standard regression tests for CI

clusters:
  - id: single-node
    provisioner: k0rdent
    template: single-node-k8s-1.29
  - id: multi-node
    provisioner: k0rdent
    template: multi-node-k8s-1.29

failures:
  - crashloopbackoff
  - oom-killed
  - image-pull-error

agents:
  - claude
  - codex

execution:
  parallel_clusters: 2
  parallel_tests_per_cluster: 1
  timeout_per_test: 10m
```

**Rationale**: YAML is human-readable, version-controllable, and supports complex structures.

### D3: Failure Library with Metadata

**Decision**: Failures defined as scripts + metadata files in structured directory.

**Structure**:
```
tests/failures/
├── crashloopbackoff/
│   ├── metadata.yaml    # name, description, expected_duration, severity
│   ├── induce.sh        # start/stop interface
│   └── validate.sh      # optional: verify failure state
├── oom-killed/
│   ├── metadata.yaml
│   ├── induce.sh
│   └── validate.sh
└── ...
```

**Rationale**: Extensible, self-documenting, maintains current shell script approach.

### D4: On-Disk Result Storage (Initial)

**Decision**: Store results as structured files on disk (JSON/YAML).

**Structure**:
```
test-results/
├── 2024-12-23-regression-suite-abc123/
│   ├── run.meta.yaml           # run metadata, timing, status
│   ├── matrix.yaml             # frozen matrix definition
│   ├── results/
│   │   ├── single-node--crashloopbackoff--claude.json
│   │   ├── single-node--crashloopbackoff--codex.json
│   │   └── ...
│   └── logs/
│       ├── orchestrator.log
│       └── ...
└── ...
```

**Rationale**: Simple, no external dependencies, supports grep/jq analysis. Can migrate to SQLite later if needed.

### D5: Existing Harness as Execution Backend

**Decision**: Wrap existing `run-live-test.sh` rather than rewrite.

**Rationale**:
- Proven to work
- Reduces risk
- Allows incremental migration
- New orchestrator handles coordination; existing script handles single-test execution

## Decision Points (Pending Evaluation)

### DP1: Chaos Engineering Tools for Failure Injection

**Status**: Needs further evaluation

**Context**: Chaos engineering tools (Chaos Mesh, Litmus, ChaosBlade) provide sophisticated failure injection mechanisms. However, there's a fundamental mismatch with our use case:

| Chaos Engineering (Typical) | Our Use Case (Triage Testing) |
|----------------------------|-------------------------------|
| Inject failure into running system | Create failure condition |
| Verify system **recovers** gracefully | Want failure to **persist** for detection |
| Success = system stays up | Success = triage agent diagnoses correctly |
| Auto-cleanup after recovery | Cleanup only after triage completes |
| Tests resilience | Tests observability + diagnosis |

**Implications**:
- We may need only the *injection* capabilities, not the full workflow
- Different failure types may require different tools
- A "meta-harness" layer would be needed to orchestrate multiple chaos tools
- Custom lifecycle management (inject → wait for triage → assess → cleanup)

**Options**:

| Option | Pros | Cons |
|--------|------|------|
| **A: Custom shell scripts (current)** | Simple, full control, works today | Manual maintenance, limited failure types |
| **B: Chaos Mesh CRDs (injection only)** | Declarative, rich failure types, K8s-native | Overkill? Need to disable auto-recovery features |
| **C: Multiple tools via meta-harness** | Best tool for each failure type | Complexity, multiple dependencies |
| **D: Hybrid (scripts + selective chaos tools)** | Pragmatic, adopt as needed | Inconsistent interface |

**Recommendation**: Start with Option A (current scripts), evaluate Option D as failure library grows. Revisit when we need failure types beyond simple pod/container failures.

**Action needed**: Prototype one chaos tool integration to understand effort/value ratio.

### DP2: Test Orchestration Framework (Testkube vs Custom)

**Status**: Needs further evaluation

**Context**: Testkube provides Kubernetes-native test orchestration with parallel execution, multi-cluster support, and result aggregation. However, our requirements include nightcrier-specific lifecycle management.

**Our specific needs not covered by standard frameworks**:
1. Nightcrier process lifecycle (start, wait for MCP subscription, monitor logs)
2. Agent comparison across LLM backends (not a standard test pattern)
3. Triage quality assessment (not just pass/fail)
4. k0rdent integration (custom provisioner)

**Options**:

| Option | Pros | Cons |
|--------|------|------|
| **A: Custom thin orchestrator** | Full control, simple debugging, matches our workflow | Build/maintain ourselves |
| **B: Testkube with custom executor** | Parallel execution, JUnit, result store built-in | Learning curve, may fight the framework |
| **C: Testkube for outer loop, custom for inner** | Best of both? | Two systems to understand |

**Recommendation**: Start with Option A (custom thin layer). If parallelization/CI integration becomes painful, evaluate migrating to Option C.

**Decision criteria for revisiting**:
- >20 test cases in regression suite
- Need for complex parallelization across clusters
- CI integration requiring JUnit/standard formats

## Risks / Trade-offs

| Risk | Impact | Mitigation |
|------|--------|------------|
| k0rdent interface delays | Blocks cluster provisioning | Local/kind fallback; stub provisioner for dev |
| Test matrix explosion | Long run times | Curated regression subset; parallel execution |
| Flaky tests | False negatives | Retry logic; failure classification |
| Repo complexity growth | Maintenance burden | Consider separate repo if complexity exceeds threshold |
| Chaos tool mismatch | Wrong tool for job | Start simple (scripts), prototype before adopting |
| Over-engineering | Delayed delivery | Build thin layer first, adopt frameworks only when pain justifies |

## Migration Plan

1. **Phase 1**: Framework scaffolding (orchestrator, interfaces, config parsing)
2. **Phase 2**: Local provisioner (kind/k3d) for development
3. **Phase 3**: k0rdent provisioner integration (pending interface)
4. **Phase 4**: Result aggregation and comparison tools
5. **Phase 5**: CI integration

Rollback: Each phase is additive; existing `tests/run-live-test.sh` remains functional throughout.

## Open Questions

1. **k0rdent Interface Contract** - What methods/data does k0rdent expose for cluster lifecycle?
2. **Cluster Artifact Format** - What does k0rdent return when a cluster is provisioned?
3. **MCP Server Deployment** - Does k0rdent deploy kubernetes-mcp-server, or is that our responsibility?
4. **Credential Management** - How are cluster credentials (kubeconfig, API keys) securely passed?
5. **Repository Location** - At what complexity threshold do we split into separate repo?
6. **Chaos Tool Viability** - Can chaos engineering tools (Chaos Mesh, Litmus) be configured to inject-and-hold rather than inject-and-recover? Need prototype to validate.
7. **Failure Type Coverage** - What failure types do we need beyond pod crashes? Network partitions, disk pressure, node failures? This drives tool selection.
