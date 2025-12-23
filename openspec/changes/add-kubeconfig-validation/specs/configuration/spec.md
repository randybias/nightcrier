## ADDED Requirements

### Requirement: Kubeconfig Content Validation

The system SHALL validate kubeconfig file contents at startup, not just file existence.

#### Scenario: Invalid YAML syntax
- **WHEN** the application starts
- **AND** a kubeconfig file contains invalid YAML syntax
- **THEN** the application SHALL exit with a non-zero status
- **AND** the error message SHALL indicate "invalid YAML" and include the parse error

#### Scenario: Missing clusters field
- **WHEN** the application starts
- **AND** a kubeconfig file has no `clusters` field or it is empty
- **THEN** the application SHALL exit with a non-zero status
- **AND** the error message SHALL state "kubeconfig must contain at least one cluster"

#### Scenario: Missing users field
- **WHEN** the application starts
- **AND** a kubeconfig file has no `users` field or it is empty
- **THEN** the application SHALL exit with a non-zero status
- **AND** the error message SHALL state "kubeconfig must contain at least one user"

#### Scenario: Missing contexts field
- **WHEN** the application starts
- **AND** a kubeconfig file has no `contexts` field or it is empty
- **THEN** the application SHALL exit with a non-zero status
- **AND** the error message SHALL state "kubeconfig must contain at least one context"

#### Scenario: Invalid current-context reference
- **WHEN** the application starts
- **AND** a kubeconfig file specifies a `current-context` that does not exist in `contexts`
- **THEN** the application SHALL exit with a non-zero status
- **AND** the error message SHALL state which context name was not found

#### Scenario: Valid kubeconfig accepted
- **WHEN** the application starts
- **AND** a kubeconfig file contains valid YAML with clusters, users, and contexts
- **AND** the current-context references a valid context entry
- **THEN** the application SHALL proceed with startup
