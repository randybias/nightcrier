# agent-configuration Spec Delta

This spec delta modifies the `configuration` spec to make agent prompt optional.

## MODIFIED Requirements

### Requirement: Required Configuration Validation

The system SHALL validate required configuration parameters at startup, but agent_prompt is no longer required.

#### Scenario: Optional additional agent prompt
- **WHEN** the application starts
- **AND** `additional_agent_prompt` is not configured
- **THEN** the application SHALL start successfully
- **AND** the agent SHALL receive only the system prompt

#### Scenario: Additional agent prompt provided
- **WHEN** the application starts
- **AND** `additional_agent_prompt` is configured with a non-empty value
- **THEN** the application SHALL pass the additional prompt to the agent
- **AND** the additional prompt SHALL be appended after the system prompt context

### Requirement: Single Source of Configuration Truth

The system SHALL NOT define default values for required configuration parameters in multiple locations, and SHALL handle optional additional_agent_prompt gracefully.

#### Scenario: No duplicate defaults in executor
- **WHEN** the agent executor is initialized
- **THEN** it SHALL receive all configuration values from the Config struct
- **AND** it SHALL NOT define its own default values for agent_timeout, agent_model, agent_allowed_tools
- **AND** it SHALL handle empty `additional_agent_prompt` gracefully without requiring a default

## REMOVED Requirements

### Requirement: Mandatory Agent Prompt

The system SHALL NOT require agent_prompt configuration. The following scenario is removed:

#### Scenario: Missing agent prompt no longer fails
- **WHEN** the application starts
- **AND** `agent_prompt` is not configured
- **THEN** the application SHALL NOT exit with an error
- **AND** the application SHALL start successfully using only the system prompt
- **RATIONALE**: The k8s-troubleshooter skill provides investigation methodology; the system prompt file alone provides sufficient context

## Cross-References

- Related to: `system-prompt` spec (defines what system prompt contains)
- Related to: `prompt-capture` spec (defines how prompts are captured for audit)
