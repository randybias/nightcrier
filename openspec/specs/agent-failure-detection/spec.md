# agent-failure-detection Specification

## Purpose
TBD - created by archiving change prevent-spurious-notifications. Update Purpose after archive.
## Requirements
### Requirement: Agent Execution Validation

The runner MUST detect when an agent execution failed to produce valid triage output and distinguish this from successful investigations.

#### Scenario: Exit code indicates failure

**Given** an agent execution completes with exit code 1
**When** the system validates the execution
**Then** the execution MUST be marked as failed
**And** the status MUST be set to "agent_failed"

#### Scenario: Investigation file missing

**Given** an agent execution completes with exit code 0
**But** no `output/investigation.md` file exists
**When** the system validates the execution
**Then** the execution MUST be marked as failed
**And** the failure reason MUST indicate "investigation report missing"

#### Scenario: Investigation file too small

**Given** an agent execution completes with exit code 0
**And** `output/investigation.md` exists
**But** the file size is less than 100 bytes
**When** the system validates the execution
**Then** the execution MUST be marked as failed
**And** the failure reason MUST indicate "investigation report too small"

#### Scenario: Valid investigation produced

**Given** an agent execution completes with exit code 0
**And** `output/investigation.md` exists with size >= 100 bytes
**When** the system validates the execution
**Then** the execution MUST NOT be marked as failed
**And** the status MUST be set based on normal logic (success/failed/error)

### Requirement: Failure Reason Tracking

The system MUST capture and log the reason why an agent execution was determined to have failed.

#### Scenario: Failure reason in result metadata

**Given** an agent execution validation fails
**When** the result.json is written
**Then** the result MUST include a "failure_reason" field
**And** the field MUST contain a human-readable explanation

#### Scenario: Failure reason logged

**Given** an agent execution validation fails
**When** the validation completes
**Then** a WARNING level log MUST be emitted
**And** the log MUST include the incident_id
**And** the log MUST include the failure reason

### Requirement: Status Values

The system MUST use distinct status values to indicate different execution outcomes.

#### Scenario: Status value for agent failures

**Given** the agent execution validation determines the agent failed
**Then** the status value MUST be "agent_failed"
**And** this value MUST be distinct from "success", "failed", and "error"

#### Scenario: Status value for successful investigations

**Given** the agent execution validation passes
**And** the agent completed successfully
**Then** the status value MUST be "success"

#### Scenario: Status value for agent runtime errors

**Given** the agent process fails to start or crashes
**When** the error is captured by the executor
**Then** the status value MUST be "error"

