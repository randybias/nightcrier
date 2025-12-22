# agent-container Spec Delta

## ADDED Requirements

### Requirement: Model Validation
The agent runner SHALL validate model configuration before execution.

#### Scenario: Pre-flight model validation
- **WHEN** run-agent.sh is invoked with any agent
- **THEN** the agent's model name is validated against known valid identifiers
- **AND** validation fails fast with clear error message if model invalid
- **AND** error message suggests valid alternatives if available

#### Scenario: API key validation
- **WHEN** run-agent.sh is invoked with any agent
- **THEN** the required API key environment variable is checked for presence
- **AND** API key validity is tested via API call if possible
- **AND** validation fails with clear error if key missing or invalid
- **AND** error message includes where to obtain/configure the key

#### Scenario: API quota checking
- **WHEN** pre-flight validation runs
- **THEN** API quota status is queried if the provider API supports it
- **AND** quota exhaustion is detected and reported with warning
- **AND** execution proceeds with warning (not hard failure) if quota low
- **AND** error message includes resolution steps (wait, upgrade plan, alternative model)

### Requirement: Enhanced Error Handling
The agent runner SHALL provide clear diagnostics for all failure modes.

#### Scenario: API quota exhaustion detection
- **WHEN** an agent execution fails due to quota exhaustion
- **THEN** the specific quota error is detected from API response
- **AND** a clear error message is logged indicating quota limit reached
- **AND** the error message includes resolution steps specific to the provider
- **AND** the agent execution fails with distinct exit code 10

#### Scenario: Timeout handling
- **WHEN** an agent execution exceeds the configured timeout
- **THEN** the agent process is terminated gracefully
- **AND** partial results are preserved if available
- **AND** a timeout event is logged with context (timeout value, elapsed time)
- **AND** the agent execution fails with distinct exit code 11

#### Scenario: Enhanced error messages
- **WHEN** any agent error occurs
- **THEN** the error message includes the specific failure reason
- **AND** the error message includes resolution steps
- **AND** full API error details are logged in DEBUG mode
- **AND** a distinct exit code is returned based on error type

## MODIFIED Requirements

### Requirement: Multi-Agent Container
The system SHALL provide a Docker container capable of running multiple AI CLI agents for Kubernetes incident triage.

#### Scenario: Agent execution with timeout (NEW)
- **WHEN** run-agent.sh is invoked with any agent
- **THEN** a timeout is configured based on test type or CLI argument
- **AND** agent execution is bounded by the timeout value
- **AND** timeout expiry triggers graceful termination
- **AND** timeout events are logged with context

## Cross-References

- **Depends on**: agent-container (existing) - base multi-agent infrastructure
- **Impacts**: test-harness - validation integrated into test orchestration
- **Related to**: agent-logging - error logging format consistency
