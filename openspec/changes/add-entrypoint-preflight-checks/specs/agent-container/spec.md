# agent-container Spec Delta

## ADDED Requirements

### Requirement: Preflight Validation
The entrypoint script SHALL perform comprehensive validation checks before any expensive setup operations to ensure all required configuration and resources are available.

#### Scenario: Early failure detection
- **WHEN** the entrypoint script starts
- **THEN** preflight validation runs immediately after the startup banner
- **AND** preflight validation completes before git clone operations
- **AND** preflight validation completes before agent path setup
- **AND** the total preflight validation time is less than 3 seconds

#### Scenario: Environment variable validation
- **GIVEN** a required environment variable is missing (e.g., OUTPUT_URL_REPORT)
- **WHEN** preflight validation runs
- **THEN** the script exits with error code 1
- **AND** the error message identifies which variable is missing
- **AND** the error message includes remediation steps

#### Scenario: Required file validation
- **GIVEN** the incident.json file is not mounted at /home/agent/incident.json
- **WHEN** preflight validation runs
- **THEN** the script exits with error code 1
- **AND** the error message identifies the missing file
- **AND** the error message explains the ConfigMap mount requirement

#### Scenario: Kubeconfig presence validation
- **GIVEN** the kubeconfig is not mounted at /home/agent/.kube/config
- **WHEN** preflight validation runs
- **THEN** the script exits with error code 1
- **AND** the error message identifies the missing kubeconfig
- **AND** the error message explains the Secret mount requirement

#### Scenario: Kubectl connectivity validation
- **GIVEN** the kubeconfig contains an expired token
- **WHEN** preflight validation runs
- **THEN** kubectl auth can-i test is executed
- **AND** the script exits with error code 1 when auth fails
- **AND** the error message includes the kubectl error output
- **AND** the error message explains how to regenerate credentials

#### Scenario: API key validation
- **GIVEN** AGENT_CLI is "claude" and ANTHROPIC_API_KEY is not set
- **WHEN** preflight validation runs
- **THEN** the script exits with error code 1
- **AND** the error message identifies the missing API key
- **AND** the error message maps the agent to the required key

#### Scenario: Output directory writability validation
- **GIVEN** the output directory is read-only or full
- **WHEN** preflight validation runs
- **THEN** a test file is created and written to /home/agent/output/
- **AND** the script exits with error code 1 if write fails
- **AND** the error message explains the filesystem issue

#### Scenario: Successful preflight validation
- **GIVEN** all required configuration and resources are present
- **WHEN** preflight validation runs
- **THEN** all checks pass without error
- **AND** a success summary is printed showing check count and duration
- **AND** normal execution continues with git clone and agent setup

#### Scenario: Structured error messages
- **GIVEN** any preflight check fails
- **WHEN** the error is reported
- **THEN** the error message includes "PREFLIGHT CHECK FAILED: <check-name>"
- **AND** the error message includes "Problem:" describing what's wrong
- **AND** the error message includes "Required:" describing what's needed
- **AND** the error message includes "Fix:" providing remediation steps
- **AND** the error message is printed to stderr

#### Scenario: Check ordering optimization
- **GIVEN** preflight validation is running
- **WHEN** executing checks
- **THEN** environment variable checks run before file existence checks
- **AND** file existence checks run before kubectl connectivity checks
- **AND** kubectl connectivity checks run last (most expensive)
- **AND** checks stop at the first failure (fail-fast)

## MODIFIED Requirements

### Requirement: Container Startup Sequence
The container entrypoint SHALL execute initialization steps in a specific order to ensure fast failure detection, with preflight validation running before any expensive setup operations.

#### Scenario: Startup order (MODIFIED)
- **WHEN** the container starts
- **THEN** the startup banner is printed first
- **AND** preflight validation runs immediately after banner
- **AND** skills are cloned only after preflight passes
- **AND** agent paths are configured only after preflight passes
- **AND** the agent is invoked only after all setup completes

*Previous behavior: Skills cloned before validation, causing wasted work on config errors*

## Notes

This change enhances the existing agent-container capability by adding fail-fast validation. It's purely additive - no existing functionality is removed or changed, only validation is added earlier in the execution flow.

The preflight checks create a clear separation between:
1. **Configuration validation** (< 3s) - Are we set up correctly?
2. **Setup operations** (5-10s) - Clone skills, configure paths
3. **Agent execution** (60-300s) - Run the investigation

This allows operators to quickly identify configuration issues without waiting for expensive setup operations or agent execution.
