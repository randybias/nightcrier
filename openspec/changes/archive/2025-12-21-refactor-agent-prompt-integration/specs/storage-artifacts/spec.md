# storage-artifacts Spec Delta

This spec delta modifies the `cloud-storage` spec to include prompt-sent.md as an incident artifact.

## MODIFIED Requirements

### Requirement: Incident Artifacts (MODIFIED)

The incident artifacts stored SHALL include the captured prompt file.

#### Scenario: Prompt file included in artifacts
- **WHEN** incident artifacts are collected for storage
- **THEN** the artifacts SHALL include `prompt-sent.md`
- **AND** the file SHALL be uploaded alongside incident.json, investigation.md, and agent logs

#### Scenario: Prompt file in Azure Blob Storage
- **WHEN** incident artifacts are uploaded to Azure Blob Storage
- **THEN** `prompt-sent.md` SHALL be uploaded to the incident blob path
- **AND** `prompt-sent.md` SHALL appear in the index.html file listing
- **AND** `prompt-sent.md` SHALL have a presigned SAS URL for access

#### Scenario: Prompt file in filesystem storage
- **WHEN** incident artifacts are stored in the local filesystem
- **THEN** `prompt-sent.md` SHALL be written to the incident directory
- **AND** the file path SHALL be included in the artifact URLs map

#### Scenario: Prompt file description in index
- **WHEN** index.html is generated for Azure Blob Storage
- **THEN** the prompt-sent.md entry SHALL have:
  - Name: "Agent Prompt"
  - Description: "Full prompt sent to the AI agent including system prompt and metadata"
  - Badge: "success" (same category as incident.json)

#### Scenario: Missing prompt file handled gracefully
- **WHEN** incident artifacts are collected
- **AND** prompt-sent.md does not exist in the workspace
- **THEN** the storage operation SHALL NOT fail
- **AND** the prompt file SHALL be omitted from the uploaded artifacts
- **RATIONALE**: Prompt capture failure should not prevent incident storage

## ADDED Requirements

### Requirement: Artifact Ordering

The prompt file SHALL appear in a logical position in artifact listings.

#### Scenario: Prompt file order in index
- **WHEN** index.html lists incident artifacts
- **THEN** prompt-sent.md SHALL appear after incident.json
- **AND** prompt-sent.md SHALL appear before agent logs
- **RATIONALE**: Groups context files (incident, permissions, prompt) before output files

## Cross-References

- Related to: `cloud-storage` spec (base storage requirements)
- Related to: `prompt-capture` spec (defines prompt file format)
