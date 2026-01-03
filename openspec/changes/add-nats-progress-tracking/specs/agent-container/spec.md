## ADDED Requirements

### Requirement: NATS Client in Container

The container SHALL include the NATS CLI for publishing progress events.

#### Scenario: nats-cli binary included
- **WHEN** building the agent container
- **THEN** the image SHALL include the `nats` CLI binary (nats-cli)
- **AND** the binary SHALL be in the container's PATH

#### Scenario: NATS environment variables
- **WHEN** a K8s Job is created with NATS enabled
- **THEN** `NATS_URL` SHALL be injected as an environment variable
- **AND** `NATS_TOKEN` SHALL be injected from Secret `nightcrier-nats`
- **AND** `NATS_ENABLED` SHALL be set to "true"

#### Scenario: NATS environment variables when disabled
- **WHEN** a K8s Job is created with NATS disabled
- **THEN** `NATS_ENABLED` SHALL be set to "false" or not present
- **AND** the entrypoint SHALL skip NATS publishing

### Requirement: Run Event Publishing

The entrypoint SHALL publish run events to NATS.

#### Scenario: Publish run.started after preflight
- **WHEN** the entrypoint starts
- **AND** `NATS_ENABLED=true`
- **AND** preflight validation has passed (see add-entrypoint-preflight-checks)
- **THEN** the entrypoint SHALL publish a message to `triage.$INCIDENT_ID.run.started`
- **AND** the payload SHALL be JSON with incident_id, agent_cli, model, cluster, timestamp
- **AND** this SHALL occur before clone_skills or other setup operations

#### Scenario: Publish run.completed on exit
- **WHEN** the entrypoint teardown runs
- **AND** `NATS_ENABLED=true`
- **THEN** the entrypoint SHALL publish a message to `triage.$INCIDENT_ID.run.completed`
- **AND** the payload SHALL include exit_code and timestamp
- **AND** this SHALL occur before artifact uploads

#### Scenario: NATS publish timeout
- **WHEN** publishing a NATS message
- **THEN** the publish operation SHALL timeout after 3 seconds
- **AND** failures SHALL be logged but not block execution

### Requirement: Claude Hooks Integration

The container SHALL include Claude hook scripts for "executing" events.

#### Scenario: Hook script included
- **WHEN** building the agent container
- **THEN** hook scripts SHALL be placed at `/home/agent/hooks/`
- **AND** `nats-executing.sh` SHALL handle PreToolUse events for Bash

#### Scenario: Claude settings.json configuration
- **WHEN** `AGENT_CLI=claude`
- **AND** `NATS_ENABLED=true`
- **THEN** the entrypoint SHALL create `~/.claude/settings.json` with hook configuration
- **AND** PreToolUse for Bash SHALL invoke `/home/agent/hooks/nats-executing.sh`

#### Scenario: Hook script reads stdin
- **WHEN** a Claude hook executes
- **THEN** the hook script SHALL read JSON from stdin
- **AND** extract `tool_input.command` from the PreToolUse payload

#### Scenario: Hook script publishes executing event
- **WHEN** the hook script runs
- **THEN** it SHALL publish to `triage.$INCIDENT_ID.executing`
- **AND** the `activity` field SHALL contain the first 100 characters of the command

#### Scenario: Hooks disabled when NATS disabled
- **WHEN** `NATS_ENABLED=false` or not set
- **THEN** the entrypoint SHALL NOT create Claude hook configuration
- **OR** the hook script SHALL exit immediately without publishing

### Requirement: Non-Claude Agent Compatibility

The container SHALL handle NATS for agents without hook support.

#### Scenario: Codex run events only
- **WHEN** `AGENT_CLI=codex`
- **AND** `NATS_ENABLED=true`
- **THEN** only run.started and run.completed events SHALL be published
- **AND** "executing" events SHALL NOT be published (no hook support)

#### Scenario: Gemini run events only
- **WHEN** `AGENT_CLI=gemini`
- **AND** `NATS_ENABLED=true`
- **THEN** only run.started and run.completed events SHALL be published
- **AND** "executing" events SHALL NOT be published (hook support unknown)

#### Scenario: Goose run events only
- **WHEN** `AGENT_CLI=goose`
- **AND** `NATS_ENABLED=true`
- **THEN** only run.started and run.completed events SHALL be published
- **AND** "executing" events SHALL NOT be published (hook support unknown)
