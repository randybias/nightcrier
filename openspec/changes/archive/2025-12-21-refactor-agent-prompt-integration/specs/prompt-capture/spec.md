# prompt-capture Specification

## Purpose

Defines how the full prompt sent to the AI agent is captured for auditability, debugging, and compliance. The captured prompt includes system prompt content, any additional prompt, and metadata about the invocation context.

## ADDED Requirements

### Requirement: Prompt Capture Before Execution

The system SHALL capture the complete prompt before agent execution begins.

#### Scenario: Prompt captured before subprocess launch
- **WHEN** the agent executor prepares to launch the agent subprocess
- **THEN** it SHALL write `prompt-sent.md` to the workspace
- **AND** the file SHALL be written BEFORE the subprocess is started
- **RATIONALE**: Captures prompt even if agent crashes or times out

#### Scenario: Prompt file written to workspace
- **WHEN** prompt capture occurs
- **THEN** the file SHALL be written to `{workspace}/prompt-sent.md`
- **AND** the file SHALL use markdown format for human readability

### Requirement: Prompt Metadata

The captured prompt SHALL include metadata about the invocation context.

#### Scenario: Metadata header included
- **WHEN** prompt-sent.md is generated
- **THEN** it SHALL include a metadata section with:
  - Timestamp (ISO 8601 format)
  - Incident ID
  - Cluster name
  - Agent CLI (claude, codex, goose, gemini)
  - Model name

#### Scenario: Metadata format
- **GIVEN** an incident with ID "abc-123" on cluster "westeu-cluster1"
- **WHEN** prompt-sent.md is generated at 2025-12-21T14:30:00Z
- **THEN** the metadata section SHALL appear as:
```markdown
## Metadata
- Timestamp: 2025-12-21T14:30:00Z
- Incident ID: abc-123
- Cluster: westeu-cluster1
- Agent CLI: claude
- Model: haiku
```

### Requirement: Full Prompt Content

The captured prompt SHALL include all prompt content sent to the agent.

#### Scenario: System prompt content included
- **WHEN** prompt-sent.md is generated
- **THEN** it SHALL include the full content of the system prompt file
- **AND** the content SHALL be under a "## System Prompt" heading

#### Scenario: Additional prompt content included
- **WHEN** prompt-sent.md is generated
- **AND** additional_agent_prompt was configured
- **THEN** it SHALL include the additional prompt content
- **AND** the content SHALL be under a "## Additional Prompt" heading

#### Scenario: No additional prompt
- **WHEN** prompt-sent.md is generated
- **AND** additional_agent_prompt was NOT configured or empty
- **THEN** the "## Additional Prompt" section SHALL state "None provided"

### Requirement: Prompt File Format

The captured prompt file SHALL follow a consistent markdown format.

#### Scenario: File structure
- **WHEN** prompt-sent.md is generated
- **THEN** it SHALL have the following structure:
```markdown
# Prompt Sent to Agent

## Metadata
(metadata fields)

## System Prompt
(contents of system prompt file)

## Additional Prompt
(additional prompt or "None provided")
```

## Cross-References

- Related to: `system-prompt` spec (system prompt content captured here)
- Related to: `agent-configuration` spec (additional prompt configured there)
- Related to: `storage-artifacts` spec (prompt file stored there)
