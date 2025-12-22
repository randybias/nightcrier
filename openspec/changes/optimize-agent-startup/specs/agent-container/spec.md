# agent-container Spec Delta

## ADDED Requirements

### Requirement: Context Preloading
The agent runner SHALL preload incident context before agent execution to minimize redundant operations.

#### Scenario: Incident context preloading
- **WHEN** run-agent.sh is invoked with a workspace containing incident.json
- **THEN** the incident.json contents are read from the host
- **AND** the contents are wrapped in `<incident>` XML tags
- **AND** the tagged content is included in the agent's initial prompt

#### Scenario: Permissions preloading
- **WHEN** run-agent.sh is invoked with a workspace containing incident_cluster_permissions.json
- **THEN** the permissions file contents are read from the host
- **AND** the contents are wrapped in `<permissions>` XML tags
- **AND** the tagged content is included in the agent's initial prompt

#### Scenario: Baseline triage preloading
- **WHEN** run-agent.sh is invoked and incident_triage.sh is available
- **THEN** the script is executed with `--skip-dump` flag before agent starts
- **AND** the triage output is wrapped in `<triage>` XML tags
- **AND** the tagged output is included in the agent's initial prompt
- **AND** execution is limited to 30 seconds timeout

#### Scenario: Graceful triage failure
- **WHEN** incident_triage.sh execution fails or times out
- **THEN** a warning is logged indicating triage unavailable
- **AND** the agent proceeds without triage context
- **AND** the failure does not prevent agent execution

#### Scenario: Context size monitoring
- **WHEN** preloaded context exceeds 8,000 tokens (estimated)
- **THEN** a warning is logged indicating large context size
- **AND** triage output is truncated if total exceeds 10,000 tokens
- **AND** incident.json and permissions are never truncated

### Requirement: Structured Investigation Reports
The agent runner SHALL enforce a standardized investigation report structure.

#### Scenario: Report structure template
- **WHEN** an agent is invoked for incident investigation
- **THEN** the system prompt includes explicit report structure requirements
- **AND** the structure specifies exactly four sections in order:
  1. Problem Statement
  2. Summary of Findings (with Root Cause and Confidence Level)
  3. Recommended Immediate Remediation Steps
  4. Supporting Evidence and Work Done
- **AND** the template provides markdown formatting examples

#### Scenario: Report structure validation
- **WHEN** an agent completes investigation and writes output/investigation.md
- **THEN** the report is validated for required section headers
- **AND** the report is validated for Root Cause field presence
- **AND** the report is validated for Confidence Level field presence
- **AND** validation results are logged

#### Scenario: Report versioning
- **WHEN** a structured investigation report is generated
- **THEN** the report includes a version header comment
- **AND** the version is set to "2.0" for structured format
- **AND** legacy reports (version 1.0) can still be parsed during migration

## MODIFIED Requirements

### Requirement: Built-in Skills
The container SHALL include the k8s-troubleshooter skill for Kubernetes diagnostics.

#### Scenario: Triage script availability (MODIFIED)
- **WHEN** the container is started
- **THEN** the k8s-troubleshooter skill is mounted at ~/.claude/skills/k8s-troubleshooter
- **AND** the incident_triage.sh script is executable
- **AND** the script can be invoked from the host before agent execution (NEW)
- **AND** the script supports `--skip-dump` flag for baseline diagnostics (NEW)

## Cross-References

- **Depends on**: agent-container (existing) - provides Docker container and runner infrastructure
- **Impacts**: agent-logging - structured reports improve log parsing and analysis
- **Impacts**: cloud-storage - consistent report format simplifies upload and indexing
