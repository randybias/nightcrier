## MODIFIED Requirements

### Requirement: Prompt Construction

The runner SHALL generate a structured prompt that guides the agent to perform read-only triage, delegating domain-specific methodology to the mounted skill.

#### Scenario: Prompt generation

- **WHEN** preparing agent invocation
- **THEN** the runner creates a system prompt containing:
  - Workspace file references (incident.json, permissions)
  - Operational constraints (read-only, timeouts)
  - Output location (output/investigation.md)
  - Reference to mounted skill for investigation methodology
- **AND** the prompt SHALL NOT contain domain-specific troubleshooting instructions

#### Scenario: Skill-agnostic prompt

- **WHEN** different triage skills are mounted (k8s-troubleshooter, cloud-troubleshooter, etc.)
- **THEN** the system prompt remains unchanged
- **AND** domain-specific methodology comes from the skill, not the prompt

#### Scenario: Prompt minimalism

- **WHEN** system prompt is generated
- **THEN** prompt SHOULD be under 30 lines
- **AND** prompt MUST focus on runtime context, not investigation methodology

#### Scenario: Prompt customization by severity

- **WHEN** different incident severities are processed
- **THEN** the prompt MAY indicate urgency level
- **AND** time constraints MAY be communicated
- **BUT** triage depth instructions come from the skill

## ADDED Requirements

### Requirement: Generic Triage System Prompt

The system prompt SHALL be generic and applicable to any IT triage domain, not specific to Kubernetes or any other platform.

#### Scenario: No platform-specific content

- **WHEN** system prompt is generated
- **THEN** prompt SHALL NOT reference:
  - kubectl or any CLI tool
  - Kubernetes resources (pods, namespaces, nodes)
  - Specific monitoring systems
  - Platform-specific file formats
- **AND** all platform-specific content comes from the mounted skill

#### Scenario: Standard workspace structure

- **WHEN** agent starts investigation
- **THEN** workspace contains:
  - `incident.json` - Incident context (format defined by skill)
  - `output/` - Directory for investigation artifacts
  - Skill directory with domain-specific methodology

#### Scenario: Output format delegation

- **WHEN** agent generates investigation report
- **THEN** report format is defined by the mounted skill
- **AND** system prompt only specifies output location (output/investigation.md)
