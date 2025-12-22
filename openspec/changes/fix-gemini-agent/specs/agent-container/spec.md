# agent-container Spec Delta

## MODIFIED Requirements

### Requirement: Multi-Agent Container
The system SHALL provide a Docker container capable of running multiple AI CLI agents for Kubernetes incident triage.

#### Scenario: Gemini tool mapping (NEW)
- **WHEN** agent-allowed tools are specified for Gemini
- **THEN** Claude-style tool names are mapped to Gemini equivalents:
  - Read → read_file
  - Write → write_file
  - Grep → grep
  - Glob → glob
  - Bash → run_shell_command
  - Skill → (not supported, skipped silently)
- **AND** unmapped tools are logged with warnings
- **AND** execution continues with available tools only

## ADDED Requirements

### Requirement: Gemini Session Artifact Extraction
The Gemini agent runner SHALL extract session artifacts in DEBUG mode for analysis.

#### Scenario: Gemini session extraction
- **WHEN** Gemini agent completes and DEBUG mode is enabled
- **THEN** the ~/.gemini directory is extracted from the container
- **AND** logs.json file is located in session directory structure
- **AND** bash commands are extracted from logs.json using jq
- **AND** extracted commands are written to agent-commands-executed.log

#### Scenario: Gemini session format parsing
- **WHEN** parsing Gemini logs.json for commands
- **THEN** JSON (not JSONL) format is used
- **AND** tool_use events with tool_name="bash" are extracted
- **AND** command and description fields are captured
- **AND** malformed or missing logs.json is handled gracefully with warning

#### Scenario: Gemini archive creation
- **WHEN** session extraction completes successfully
- **THEN** session directory is archived to agent-session.tar.gz
- **AND** archive is stored in workspace logs directory
- **AND** extracted source directory is cleaned up

## Cross-References

- **Depends on**: agent-container (existing) - base multi-agent infrastructure
- **Impacts**: agent-logging - Gemini session format differs from Claude/Codex JSONL
