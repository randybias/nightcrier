# Change: Fix FaultEvent struct for resource-faults mode

## Why

The FaultEvent struct in `internal/events/event.go` was designed for the `faults` subscription mode which uses a nested `event` structure. The `resource-faults` mode sends a completely different flat structure with `resource`, `context`, `faultType`, and `severity` at the top level. This causes event.json to be written with empty fields when using `resource-faults` mode.

## What Changes

- **MODIFIED** FaultEvent struct to support BOTH subscription modes
- Add new flat fields: `Resource`, `Context`, `FaultType`, `Severity`, `Timestamp`
- Update helper methods to check both structures
- Maintain backwards compatibility with `faults` mode

## Impact

- Affected specs: `walking-skeleton`
- Affected code:
  - `internal/events/event.go` - FaultEvent struct and helper methods
  - Potentially `internal/events/client.go` - parsing logic

## Evidence

Raw MCP data from `resource-faults` mode:
```json
{
  "cluster": "kind-events-test",
  "context": "Container entered CrashLoopBackOff state...",
  "faultType": "CrashLoop",
  "resource": {
    "apiVersion": "v1",
    "kind": "Pod",
    "name": "crashloop-test",
    "namespace": "default"
  },
  "severity": "critical",
  "subscriptionId": "sub-...",
  "timestamp": "2025-12-19T13:25:10+01:00"
}
```

Current struct expects nested `event` object - which does not exist in this mode.
