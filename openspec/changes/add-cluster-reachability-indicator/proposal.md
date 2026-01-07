# Proposal: Add Cluster Reachability Indicator

## Summary

Add a green/amber/red status indicator to the Monitored Clusters pane in the admin UI showing whether each cluster's MCP endpoint is reachable.

## Motivation

The admin UI currently shows static cluster configuration (name, environment, endpoint, triage enabled). Operators need to see at a glance whether the MCP endpoints are actually reachable to diagnose connection issues.

## Solution

Leverage the existing `ConnectionManager.GetAllConnectionStatuses()` method which already tracks connection state. Pass a `ConnectionStatusProvider` interface to the admin server and query status on each page render.

### Status Mapping

| ConnectionStatus | Indicator | Meaning |
|------------------|-----------|---------|
| `active` | Green | Endpoint reachable, receiving events |
| `connected`, `subscribing` | Amber | Connection in progress |
| `connecting`, `disconnected` | Amber | Transient state, reconnecting |
| `failed` | Red | Endpoint unreachable |

### UI Changes

Add a colored circle indicator in the Monitored Clusters table between the Name and Environment columns (or as first column). Reuse the existing `.indicator` CSS class already defined for the triage status indicators.

## Scope

- Admin UI only (no changes to connection logic)
- Read-only status display (no actions)
- Uses existing connection tracking infrastructure

## Out of Scope

- Health history/timeline
- Manual reconnect button
- Alerting on status changes
