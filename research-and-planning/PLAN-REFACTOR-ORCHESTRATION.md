# Plan: Refactor `processEvent` to `TriageOrchestrator`

## 1. Objective
Decouple the monolithic `processEvent` function in `cmd/nightcrier/main.go` into a testable, modular `TriageOrchestrator` struct.
**Goal:** Enable unit testing of the triage business logic (circuit breaking, notifications, storage) without needing a full application startup or side effects (like real Docker calls).

## 2. Current State
*   `processEvent` is a standalone function in `main`.
*   It takes ~12 arguments (context, event, clusterName, kubeconfig, permissions, workspaceMgr, executor, slackNotifier, storageBackend, circuitBreaker, config, tuning).
*   It handles:
    1.  Incident Creation (`NewFromEvent`)
    2.  Permission Checks
    3.  Workspace Setup (`WriteToFile`)
    4.  Agent Execution (`executor.Execute`)
    5.  Result Analysis (`detectAgentFailure`)
    6.  Circuit Breaking
    7.  Storage Upload
    8.  Slack Notification

## 3. Proposed Architecture

### 3.1 New Package: `internal/orchestrator`

```go
type Orchestrator struct {
    WorkspaceMgr   WorkspaceManager
    Executor       AgentExecutor
    Storage        StorageBackend
    Notifier       Notifier
    CircuitBreaker CircuitBreaker
    Config         *config.Config
    Tuning         *config.TuningConfig
}

// Interface definitions to allow mocking
type AgentExecutor interface {
    Execute(ctx context.Context, workspacePath, incidentID string) (int, agent.LogPaths, error)
}

type Notifier interface {
    SendIncidentNotification(summary *reporting.IncidentSummary) error
    SendSystemDegradedAlert(ctx context.Context, stats reporting.CircuitBreakerStats) error
    // ...
}
```

### 3.2 The `Process` Method
The logic currently in `processEvent` moves to `Orchestrator.Process`:

```go
func (o *Orchestrator) Process(ctx context.Context, event *events.FaultEvent, clusterContext ClusterContext) error {
    // 1. Create Incident
    // 2. Check Permissions (ClusterContext)
    // 3. Setup Workspace
    // 4. Run Agent (o.Executor.Execute)
    // 5. Handle Result
    // ...
}
```

## 4. Migration Steps

1.  **Define Interfaces:** Create `internal/orchestrator/interfaces.go` defining the dependencies (`AgentExecutor`, `Notifier`, `Storage`).
2.  **Create Struct:** Implement `TriageOrchestrator` in `internal/orchestrator/orchestrator.go`.
3.  **Move Logic:** Copy the body of `processEvent` into `TriageOrchestrator.Process`, replacing direct dependency calls with interface method calls.
4.  **Wire Up Main:** In `cmd/nightcrier/main.go`:
    *   Initialize dependencies.
    *   Create `orch := orchestrator.New(...)`.
    *   Replace the `processEvent(...)` call with `orch.Process(ctx, event, ...)`.

## 5. Benefits for Testing
*   **Unit Tests:** We can write `orchestrator_test.go` where we inject a `MockExecutor` that returns specific exit codes or errors.
*   **Scenarios:** We can easily test "Circuit Breaker Open -> Skip Execution" logic without actually running an agent.
*   **Config Validation:** We can test how the Orchestrator behaves with "Bad Config" (e.g., missing Slack URL) by injecting a partial config object.
