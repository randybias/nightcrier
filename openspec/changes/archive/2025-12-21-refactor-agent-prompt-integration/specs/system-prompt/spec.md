# system-prompt Specification

## Purpose

Defines the content and structure of the system prompt that guides AI agent behavior during Kubernetes incident triage. The system prompt is designed to be minimal and skill-enabling, deferring investigation methodology to the k8s-troubleshooter skill.

## ADDED Requirements

### Requirement: Skill-Aware System Prompt

The system prompt SHALL enable the k8s-troubleshooter skill to drive investigation methodology.

#### Scenario: System prompt references skill
- **WHEN** an agent is invoked for incident triage
- **THEN** the system prompt SHALL reference the k8s-troubleshooter skill
- **AND** the system prompt SHALL suggest starting with `incident_triage.sh --skip-dump`
- **AND** the system prompt SHALL NOT specify step-by-step investigation instructions

#### Scenario: System prompt describes workspace files
- **WHEN** an agent is invoked for incident triage
- **THEN** the system prompt SHALL describe available workspace files:
  - `incident.json` - fault event context from monitoring system
  - `incident_cluster_permissions.json` - cluster access permissions
- **AND** the system prompt SHALL NOT dictate the order in which files are read

#### Scenario: System prompt enforces read-only constraint
- **WHEN** an agent is invoked for incident triage
- **THEN** the system prompt SHALL explicitly state read-only constraint
- **AND** the constraint SHALL prohibit kubectl apply, delete, patch, edit
- **AND** the constraint SHALL allow kubectl get, describe, logs, top

#### Scenario: System prompt specifies output location
- **WHEN** an agent is invoked for incident triage
- **THEN** the system prompt SHALL specify output file as `output/investigation.md`
- **AND** the system prompt SHALL describe expected output format:
  - Executive summary with root cause and confidence
  - Evidence and analysis
  - Prioritized recommendations

### Requirement: Minimal Prompt Size

The system prompt SHALL be concise to maximize agent context window for actual investigation.

#### Scenario: System prompt is concise
- **WHEN** the system prompt file is measured
- **THEN** it SHALL contain fewer than 30 lines
- **AND** it SHALL contain fewer than 1000 characters
- **RATIONALE**: Shorter prompts leave more context for investigation artifacts

### Requirement: No Methodology Prescription

The system prompt SHALL NOT prescribe specific investigation steps.

#### Scenario: No step-by-step instructions
- **WHEN** the system prompt is reviewed
- **THEN** it SHALL NOT contain numbered steps like "1. Read X, 2. Do Y, 3. Write Z"
- **AND** it SHALL NOT duplicate methodology already encoded in k8s-troubleshooter skill
- **RATIONALE**: The skill encodes expert methodology; duplicating it creates conflicts

## Cross-References

- Related to: `agent-configuration` spec (system prompt file path configured there)
- Related to: `agent-container` spec (k8s-troubleshooter skill installed there)
- Related to: `prompt-capture` spec (system prompt content captured there)
