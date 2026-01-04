# Proposal: Enhance Admin UI

## Summary

Add three enhancements to the admin UI dashboard:
1. A monitored clusters pane showing configured clusters
2. Delete action for incidents
3. Cancel action for running triages (kills K8s job)

## Motivation

The current admin UI shows running triages and incidents but lacks:
- Visibility into which clusters are being monitored
- Ability to clean up incidents from the UI
- Ability to cancel stuck or unwanted triage runs

## Scope

- Add a new "Monitored Clusters" pane above Running Triages
- Add DELETE endpoint and button for incidents
- Add CANCEL endpoint and button for running triages that kills the K8s Job

## Non-Goals

- Deleting faults (incidents reference faults, keep them)
- Deleting completed triage execution records (archive later)
- Authentication/authorization (local dev tool only)

## Design Notes

### Clusters Pane
- Read-only display of `monitored_clusters` from config
- Show: name, environment, MCP endpoint, triage enabled status
- Requires passing cluster config to admin UI server

### Delete Incident
- POST /admin/incidents/{id}/delete
- Removes incident record only
- Associated agent_executions remain (for historical reference)
- Confirmation via JavaScript confirm() dialog

### Cancel Triage
- POST /admin/triages/{execution_id}/cancel
- Kills the K8s Job for that execution
- Marks execution as cancelled (sets job_completed_at, error_message)
- Row disappears from Running Triages after refresh
