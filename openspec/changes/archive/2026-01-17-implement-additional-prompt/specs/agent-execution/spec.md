# agent-execution Spec Delta

## MODIFIED Requirements

### Requirement: Concurrent Agent Execution

The system SHALL execute agents concurrently up to a configured global limit while strictly serializing execution per cluster.

#### Scenario: Additional prompt appended to triage prompt
- **GIVEN** `agent.additional_prompt` is configured with operator text
- **WHEN** the triage prompt is built for agent execution
- **THEN** the additional prompt SHALL be appended after all other prompt components
- **AND** the additional prompt SHALL appear under a `## Additional Operator Instructions` heading
- **AND** the complete prompt (including additional prompt) SHALL be captured in `prompt-sent.md`

#### Scenario: Empty additional prompt omitted
- **GIVEN** `agent.additional_prompt` is empty or not configured
- **WHEN** the triage prompt is built for agent execution
- **THEN** no additional prompt section SHALL be appended
- **AND** the base triage prompt SHALL drive investigation methodology

#### Scenario: Additional prompt mounted in container
- **GIVEN** `agent.additional_prompt` is configured
- **WHEN** the agent Job is created
- **THEN** the ConfigMap SHALL include `additional-prompt.md` with the operator text
- **AND** the file SHALL be mounted at `/home/agent/additional-prompt.md`
