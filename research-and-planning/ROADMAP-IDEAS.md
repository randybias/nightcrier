# Roadmap

This document tracks future enhancements and features planned for Nightcrier.

## Future Enhancements

### Observability and Metrics

#### Prometheus Metrics for Agent Failures

Add comprehensive Prometheus metrics to monitor agent execution health and circuit breaker state.

**Metrics to Add:**

1. **`agent_failures_total`** (Counter)
   - Description: Total count of agent execution failures
   - Labels: `failure_reason` (e.g., "exit_code_non_zero", "investigation_missing", "investigation_too_small")
   - Use case: Track failure patterns and identify most common failure modes

2. **`circuit_breaker_state`** (Gauge)
   - Description: Current state of the circuit breaker (0 = closed/healthy, 1 = open/degraded)
   - Use case: Monitor system health status, trigger external alerts
   - Transitions: 0 → 1 when threshold reached, 1 → 0 on recovery

3. **`agent_execution_success_rate`** (Histogram)
   - Description: Ratio of successful agent executions over time window
   - Buckets: [0.5, 0.75, 0.90, 0.95, 0.99, 1.0]
   - Use case: SLO monitoring, capacity planning

**Implementation Location:**
- Add metrics to `internal/reporting/circuit_breaker.go`
- Instrument `cmd/runner/main.go` in the agent execution flow
- Add Prometheus HTTP endpoint in `cmd/runner/main.go`

**Configuration:**
```yaml
# Enable Prometheus metrics endpoint
metrics:
  enabled: true
  port: 9090
  path: /metrics
```

**Documentation:**
- Add metrics documentation to README.md
- Include example Prometheus queries
- Provide sample Grafana dashboard JSON

**Priority:** Medium
**Estimated Effort:** 2-3 days
**Dependencies:** None

**Related Work:**
- Originally planned as Task 15 in prevent-spurious-notifications change
- Deferred to allow focus on core functionality
- Can be implemented independently without breaking changes

---

## Completed Features

### ✅ Prevent Spurious Notifications on Agent Failures
- **Status:** Completed
- **Date:** 2025-12-19
- **Description:** Implemented circuit breaker pattern to prevent notification spam during LLM API outages
- **Key Features:**
  - Agent execution validation
  - Conditional storage upload and notifications
  - Circuit breaker with configurable threshold
  - System degraded/recovered alerts
  - Comprehensive configuration options
- **OpenSpec Change:** `prevent-spurious-notifications`

---

## How to Use This Roadmap

### Proposing New Features

1. Create a new OpenSpec change proposal: `openspec proposal create <feature-name>`
2. Add the proposed feature to this roadmap under "Future Enhancements"
3. Discuss with the team and refine the proposal
4. Once approved, implement following the OpenSpec workflow

### Implementing Roadmap Items

1. Check that dependencies are met
2. Create an OpenSpec change if not already exists
3. Implement following the tasks in the change proposal
4. Update this roadmap to move the item from "Future Enhancements" to "Completed Features"

### Priority Levels

- **High:** Critical for operations, blocking other work, or high user demand
- **Medium:** Valuable enhancement, not blocking, reasonable effort
- **Low:** Nice-to-have, can be deferred indefinitely

---

## Contributing

To propose a new feature or enhancement:
1. Open an issue describing the use case and benefits
2. If significant, create an OpenSpec proposal
3. Add to this roadmap with priority and effort estimates
4. Discuss with maintainers before implementation
