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
- **WHEN** run-agent.sh is invoked and skill triage script is available
- **THEN** the script is executed before agent starts
- **AND** the triage output is wrapped in `<triage>` XML tags
- **AND** the tagged output is included in the agent's initial prompt
- **AND** execution is limited to 30 seconds timeout

#### Scenario: Graceful triage failure
- **WHEN** triage script execution fails or times out
- **THEN** a warning is logged indicating triage unavailable
- **AND** the agent proceeds without triage context
- **AND** the failure does not prevent agent execution

#### Scenario: Context size monitoring
- **WHEN** preloaded context exceeds 8,000 tokens (estimated)
- **THEN** a warning is logged indicating large context size
- **AND** triage output is truncated if total exceeds 10,000 tokens
- **AND** incident.json and permissions are never truncated

#### Scenario: Context injection location
- **WHEN** the agent command is constructed
- **THEN** preloaded context is inserted between system prompt and user prompt
- **AND** the system prompt remains generic (domain-agnostic)
- **AND** the preloaded context provides domain-specific data

## Cross-References

- **Depends on**: agent-container (existing) - provides Docker container and runner infrastructure
- **Related**: k8s4agents/add-standardized-report-format - report format defined in skill
- **Related**: generalize-triage-system-prompt - system prompt remains generic
