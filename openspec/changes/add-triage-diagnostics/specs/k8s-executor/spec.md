# k8s-executor Spec Delta

## ADDED Requirements

### Requirement: Pod Log Capture

The K8s executor SHALL capture Kubernetes pod logs for every triage Job and upload them to object storage for diagnostics.

#### Scenario: Capture pod logs on Job completion

- **GIVEN** a triage Job has completed (success or failure)
- **WHEN** the processor detects Job completion
- **THEN** pod logs are fetched from the Kubernetes API
- **AND** logs are uploaded to `{incidentID}/logs/pod-logs.txt` in object storage
- **AND** the upload URL is stored in the `agent_executions` table

#### Scenario: Pod already deleted

- **GIVEN** a triage Job has completed
- **AND** the pod has been deleted (TTL cleanup or manual)
- **WHEN** pod log capture is attempted
- **THEN** a warning is logged
- **AND** no pod logs are uploaded
- **AND** execution continues without error

#### Scenario: Empty pod logs

- **GIVEN** a triage Job has completed
- **AND** the pod logs are empty
- **WHEN** pod log capture is attempted
- **THEN** an empty file is uploaded with a note explaining logs were empty
- **AND** the URL is still stored for consistency

#### Scenario: Large pod logs

- **GIVEN** a triage Job has completed
- **AND** pod logs exceed 1MB
- **WHEN** pod log capture is attempted
- **THEN** logs are truncated to 1MB
- **AND** a "[truncated - full logs exceeded 1MB]" marker is appended
- **AND** truncated logs are uploaded

### Requirement: Pod Logs URL Storage

The system SHALL store the pod logs URL in the database for retrieval.

#### Scenario: URL stored after upload

- **GIVEN** pod logs have been uploaded to object storage
- **WHEN** upload completes successfully
- **THEN** the signed URL is stored in `agent_executions.pod_logs_url`
- **AND** the URL can be retrieved for display in admin UI

#### Scenario: Upload failure

- **GIVEN** pod log upload to object storage fails
- **WHEN** the upload error is caught
- **THEN** a warning is logged with the error details
- **AND** `pod_logs_url` remains NULL
- **AND** triage completion continues without error

## Notes

Pod log capture is a diagnostic enhancement that should never block or fail a triage run. All failures are logged as warnings and execution continues.

The pod logs capture output that would otherwise only be visible via `kubectl logs`, including:
- Container startup messages
- Entrypoint script output (preflight checks, NATS connectivity)
- Any output before the agent logging infrastructure begins
