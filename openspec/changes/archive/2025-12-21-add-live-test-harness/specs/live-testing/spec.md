# Spec: Live Testing

## ADDED Requirements

### Requirement: Test Configuration Management

The test harness SHALL manage configuration through templates and secret injection to prevent credential leakage.

#### Scenario: Generate test configuration from template

**Given** a configuration template at `tests/config-templates/test-claude.yaml.tmpl`
**And** secrets loaded from `~/dev-secrets/nightcrier-secrets.env`
**When** the config generator is run
**Then** a valid configuration file is created at `configs/config-test-claude.yaml`
**And** all placeholder variables are replaced with actual secret values
**And** the generated config is not committed to version control

#### Scenario: Handle missing secrets

**Given** a configuration template with placeholder `${ANTHROPIC_API_KEY}`
**And** the secrets file does NOT contain `ANTHROPIC_API_KEY`
**When** the config generator is run
**Then** the script SHALL exit with error code 1
**And** an error message SHALL indicate which secret is missing

### Requirement: Failure Induction

The test harness SHALL induce real Kubernetes failures to trigger agent triage.

#### Scenario: Induce crashloopbackoff failure

**Given** a Kubernetes cluster is accessible
**And** admin privileges are available
**When** `01_induce_failure_crashloopbackoff.sh start` is executed
**Then** a pod SHALL be created that crashes repeatedly
**And** the pod enters CrashLoopBackOff state within 30 seconds
**And** the MCP server emits a fault event for the crash

#### Scenario: Clean up induced failure

**Given** a crashloopbackoff failure has been induced
**When** `01_induce_failure_crashloopbackoff.sh stop` is executed
**Then** the crashing pod SHALL be deleted
**And** no test artifacts remain in the cluster
**And** the script exits with code 0

#### Scenario: Failure script with invalid argument

**Given** any failure induction script
**When** the script is executed with an argument other than `start` or `stop`
**Then** the script SHALL exit with error code 1
**And** usage information SHALL be printed to stderr

### Requirement: Test Orchestration

The test harness SHALL execute end-to-end tests with nightcrier and validate results.

#### Scenario: Run successful test cycle

**Given** secrets are available at `~/dev-secrets/nightcrier-secrets.env`
**And** a kubeconfig is available at `~/dev-secrets/eastus-cluster1-admin.yaml`
**When** `run-live-test.sh claude crashloopbackoff` is executed
**Then** nightcrier SHALL start in background mode
**And** nightcrier SHALL subscribe to the MCP server
**And** the crashloopbackoff failure SHALL be induced
**And** nightcrier SHALL trigger agent execution
**And** the agent SHALL complete investigation
**And** the failure SHALL be cleaned up
**And** nightcrier SHALL be stopped
**And** a test report SHALL be displayed

#### Scenario: Test with DEBUG mode enabled

**Given** all preconditions for a test run
**When** `run-live-test.sh claude crashloopbackoff --debug` is executed
**Then** the generated config SHALL have `agent_debug: true`
**And** the agent container SHALL produce debug-level logs
**And** session artifacts SHALL include full session tarball

#### Scenario: Test timeout

**Given** a test is running  **And** the agent does not complete within 5 minutes
**When** the timeout expires
**Then** the test SHALL be marked as FAILED
**And** nightcrier SHALL be stopped
**And** the failure SHALL be cleaned up
**And** the report SHALL indicate timeout

#### Scenario: Cleanup on script interruption

**Given** a test is running
**When** the script receives SIGINT or SIGTERM
**Then** nightcrier SHALL be stopped gracefully
**And** the induced failure SHALL be cleaned up
**And** the script SHALL exit with code 130 (interrupted)

### Requirement: Log Monitoring

The test harness SHALL monitor nightcrier logs to track test progress.

#### Scenario: Detect MCP subscription

**Given** nightcrier is starting
**When** the log contains "subscribed to fault events"
**Then** the monitoring function SHALL return success
**And** the test SHALL proceed to failure induction

#### Scenario: Detect agent execution start

**Given** a failure has been induced
**When** the log contains "starting.*agent"
**Then** the monitoring function SHALL return success
**And** the test SHALL wait for agent completion

#### Scenario: Detect agent completion

**Given** an agent is executing
**When** the log contains "Agent completed"
**Then** the monitoring function SHALL return success
**And** the test SHALL proceed to cleanup

#### Scenario: Timeout waiting for log pattern

**Given** the monitoring function is waiting for a pattern
**And** the pattern does not appear within the timeout
**When** the timeout expires
**Then** the monitoring function SHALL return failure
**And** the test SHALL be aborted

### Requirement: Report Generation

The test harness SHALL generate structured reports of test results.

#### Scenario: Generate successful test report

**Given** a test has completed successfully
**When** the report generator is run
**Then** a human-readable report SHALL be displayed to stdout
**And** the report SHALL include test metadata (agent, test type, timestamps)
**And** the report SHALL include a timeline of key events
**And** the report SHALL list generated artifacts with sizes
**And** the report SHALL show validation checklist with pass/fail markers
**And** a JSON report SHALL be written to `tests/logs/report-<timestamp>.json`

#### Scenario: Report for failed test

**Given** a test has failed
**When** the report generator is run
**Then** the report SHALL indicate FAILED status
**And** the report SHALL include error details from logs
**And** the report SHALL show which validation step failed

### Requirement: Secret Management

The test harness SHALL keep secrets out of version control while making them available to tests.

#### Scenario: Load secrets from external file

**Given** a secrets file exists at `~/dev-secrets/nightcrier-secrets.env`
**When** the test script sources the secrets file
**Then** all secret values SHALL be available as environment variables
**And** the secrets file path SHALL NOT be within the repository

#### Scenario: Prevent secrets in logs

**Given** secrets are loaded as environment variables
**When** test logs are written
**Then** secret values SHALL NOT appear in plaintext in logs
**And** configuration files generated in the repository SHALL NOT be tracked by git

### Requirement: Multi-Agent Support

The test harness SHALL support testing all triage agents (claude, codex, gemini).

#### Scenario: Run test with different agents

**Given** secrets for all agents are available
**When** `run-live-test.sh <agent> crashloopbackoff` is executed
**Where** `<agent>` is one of: claude, codex, gemini
**Then** the correct configuration template SHALL be used
**And** the correct agent CLI SHALL be configured
**And** the test SHALL execute successfully
**And** agent-specific artifacts SHALL be collected

#### Scenario: Invalid agent name

**Given** any test invocation
**When** an unsupported agent name is provided
**Then** the script SHALL exit with error code 1
**And** valid agent names SHALL be listed in the error message
