# agent-execution Spec Delta

## ADDED Requirements

### Requirement: Tool Restrictions

The system SHALL enforce tool restrictions for agent execution based on the configured `allowed_tools` list.

#### Scenario: Tool restrictions passed to Claude agent
- **GIVEN** `agent.allowed_tools` is configured as "Read,Grep,Glob,Bash"
- **AND** the agent CLI is `claude`
- **WHEN** the agent Job is created
- **THEN** the Job SHALL include `ALLOWED_TOOLS` environment variable
- **AND** the entrypoint SHALL pass `--allowedTools "Read,Grep,Glob,Bash"` to Claude

#### Scenario: Tool restrictions translated for Codex agent
- **GIVEN** `agent.allowed_tools` is configured as "Read,Write,Bash"
- **AND** the agent CLI is `codex`
- **WHEN** the entrypoint runs the agent
- **THEN** the entrypoint SHALL translate to Codex format: `--allow Read --allow Write --allow Bash`
- **AND** `--dangerously-bypass-approvals-and-sandbox` SHALL NOT be used when restrictions are configured

#### Scenario: Tool restrictions warning for unsupported agent
- **GIVEN** `agent.allowed_tools` is configured
- **AND** the agent CLI is `gemini`
- **WHEN** the entrypoint runs the agent
- **THEN** a WARNING log SHALL be emitted stating tool restrictions are not enforced
- **AND** the agent SHALL run without restrictions
- **AND** the warning SHALL include the configured tools that cannot be enforced

#### Scenario: Empty tool restrictions uses agent defaults
- **GIVEN** `agent.allowed_tools` is empty or not configured
- **WHEN** the agent Job is created
- **THEN** no tool restriction flags SHALL be passed to the agent
- **AND** a DEBUG log SHALL note that default tool access is being used

#### Scenario: Tool restrictions captured in audit trail
- **GIVEN** `agent.allowed_tools` is configured
- **WHEN** the agent execution completes
- **THEN** `prompt-sent.md` SHALL include metadata showing:
  - The configured allowed tools
  - Whether restrictions were enforced (per agent type)

