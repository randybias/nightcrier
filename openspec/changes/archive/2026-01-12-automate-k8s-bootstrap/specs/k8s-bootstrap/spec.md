# k8s-bootstrap Specification

## Purpose

Defines requirements for automatic Kubernetes resource provisioning on application startup, enabling self-contained deployment to any Kubernetes cluster without manual setup steps.

## ADDED Requirements

### Requirement: Automatic Namespace Bootstrap

The system SHALL create the target Kubernetes namespace if it does not exist.

#### Scenario: Namespace created on first run
- **WHEN** the application starts
- **AND** the configured namespace does not exist
- **THEN** the namespace SHALL be created
- **AND** the namespace SHALL have label `app=nightcrier`

#### Scenario: Namespace exists from prior run
- **WHEN** the application starts
- **AND** the configured namespace already exists
- **THEN** the application SHALL skip namespace creation
- **AND** the application SHALL continue startup normally

#### Scenario: Namespace creation permission denied
- **WHEN** the application starts
- **AND** the configured namespace does not exist
- **AND** the application lacks permission to create namespaces
- **THEN** the application SHALL exit with status code 1
- **AND** the error message SHALL include "permission denied"
- **AND** the error message SHALL suggest applying RBAC manifests

### Requirement: Automatic RBAC Bootstrap

The system SHALL create ServiceAccount, Role, and RoleBinding for executor operations if they do not exist.

#### Scenario: RBAC resources created
- **WHEN** the application starts
- **AND** ServiceAccount `nightcrier-executor` does not exist
- **THEN** the ServiceAccount SHALL be created in the configured namespace

#### Scenario: Role permissions for Job management
- **WHEN** the Role is created
- **THEN** it SHALL grant `create`, `get`, `list`, `watch`, `delete` on `jobs` in `batch` API group
- **AND** it SHALL grant `create`, `get`, `list`, `delete` on `configmaps` in core API group
- **AND** it SHALL grant `get`, `list`, `watch` on `pods` in core API group
- **AND** it SHALL grant `get` on `pods/log` in core API group
- **AND** it SHALL grant `get`, `list` on `secrets` in core API group

#### Scenario: RoleBinding links ServiceAccount to Role
- **WHEN** the RoleBinding is created
- **THEN** it SHALL bind ServiceAccount `nightcrier-executor` to Role `nightcrier-executor`
- **AND** both SHALL be in the configured namespace

#### Scenario: RBAC resources exist
- **WHEN** the application starts
- **AND** all RBAC resources already exist
- **THEN** the application SHALL skip RBAC creation
- **AND** the application SHALL continue startup normally

### Requirement: API Keys Secret Bootstrap

The system SHALL create the `ai-api-keys` Secret containing LLM API keys if it does not exist.

#### Scenario: Secret created from configuration
- **WHEN** the application starts
- **AND** Secret `ai-api-keys` does not exist
- **AND** at least one API key is configured (anthropic, openai, or gemini)
- **THEN** the Secret SHALL be created in the configured namespace
- **AND** the Secret SHALL have type `Opaque`
- **AND** the Secret SHALL contain keys: `anthropic`, `openai`, `gemini`

#### Scenario: Empty API keys allowed
- **WHEN** creating the `ai-api-keys` Secret
- **AND** an API key is not configured (empty string)
- **THEN** the Secret SHALL still include that key with an empty value

#### Scenario: No API keys configured
- **WHEN** the application starts
- **AND** Secret `ai-api-keys` does not exist
- **AND** no API keys are configured (all empty)
- **THEN** the application SHALL exit with status code 1
- **AND** the error message SHALL state "at least one API key required"

#### Scenario: Secret already exists
- **WHEN** the application starts
- **AND** Secret `ai-api-keys` already exists
- **THEN** the application SHALL NOT modify the existing Secret
- **AND** the application SHALL continue startup normally

#### Scenario: Secret values from environment override config
- **WHEN** the application starts
- **AND** API keys are set via environment variables
- **THEN** environment variable values SHALL take precedence over config file values
- **AND** the precedence order SHALL be: env > config

### Requirement: Kubeconfig Secret Bootstrap

The system SHALL create per-cluster kubeconfig Secrets from files specified in cluster configuration.

#### Scenario: Secret created from kubeconfig file
- **WHEN** the application starts
- **AND** a cluster has `triage.enabled: true`
- **AND** `triage.kubeconfig` specifies a file path
- **AND** Secret `kubeconfig-{cluster-name}` does not exist
- **THEN** the kubeconfig file SHALL be read from disk
- **AND** a Secret SHALL be created with name `kubeconfig-{cluster-name}`
- **AND** the Secret SHALL contain key `config` with file contents as value

#### Scenario: Multiple clusters with separate kubeconfigs
- **WHEN** the application starts with 3 clusters configured
- **AND** all have `triage.enabled: true`
- **THEN** 3 separate Secrets SHALL be created
- **AND** each SHALL be named `kubeconfig-{cluster-name}`

#### Scenario: Kubeconfig file not found
- **WHEN** the application starts
- **AND** a cluster has `triage.enabled: true`
- **AND** the `triage.kubeconfig` file does not exist
- **THEN** the application SHALL exit with status code 1
- **AND** the error message SHALL include the cluster name
- **AND** the error message SHALL include the file path that was not found

#### Scenario: Kubeconfig file not readable
- **WHEN** the application starts
- **AND** a cluster has `triage.enabled: true`
- **AND** the `triage.kubeconfig` file exists but is not readable
- **THEN** the application SHALL exit with status code 1
- **AND** the error message SHALL state "permission denied"

#### Scenario: Secret already exists
- **WHEN** the application starts
- **AND** Secret `kubeconfig-{cluster-name}` already exists
- **THEN** the application SHALL NOT modify the existing Secret
- **AND** the application SHALL continue startup normally

#### Scenario: Triage disabled for cluster
- **WHEN** the application starts
- **AND** a cluster has `triage.enabled: false`
- **THEN** no kubeconfig Secret SHALL be created for that cluster

### Requirement: Kubernetes Client Configuration

The system SHALL connect to Kubernetes using configuration from environment or config file.

#### Scenario: KUBECONFIG environment variable
- **WHEN** the `KUBECONFIG` environment variable is set
- **THEN** the application SHALL use that kubeconfig path for bootstrap operations
- **AND** it SHALL take precedence over `kubeconfig_path` in config file

#### Scenario: Config file kubeconfig_path
- **WHEN** `kubeconfig_path` is set in config file
- **AND** `KUBECONFIG` environment variable is not set
- **THEN** the application SHALL use the configured path for bootstrap operations

#### Scenario: Default kubeconfig location
- **WHEN** neither `KUBECONFIG` nor `kubeconfig_path` are set
- **THEN** the application SHALL use `~/.kube/config` for bootstrap operations

#### Scenario: In-cluster configuration
- **WHEN** the application runs as a Pod in Kubernetes
- **AND** no kubeconfig is configured
- **THEN** the application SHALL use in-cluster ServiceAccount credentials
- **AND** bootstrap operations SHALL succeed if ServiceAccount has required permissions

### Requirement: Bootstrap Idempotency

The system SHALL check for existing resources before creating them, making bootstrap operations idempotent and safe to run multiple times.

#### Scenario: Full bootstrap on empty cluster
- **WHEN** the application starts in a cluster with no nightcrier resources
- **THEN** all resources SHALL be created (namespace, RBAC, Secrets)
- **AND** the application SHALL proceed to normal operation

#### Scenario: Partial bootstrap after manual setup
- **WHEN** the application starts
- **AND** namespace already exists (manually created)
- **AND** RBAC does not exist
- **AND** Secrets do not exist
- **THEN** the application SHALL skip namespace creation
- **AND** the application SHALL create RBAC resources
- **AND** the application SHALL create Secret resources

#### Scenario: No bootstrap when fully setup
- **WHEN** the application starts
- **AND** all required resources already exist
- **THEN** the application SHALL skip all resource creation
- **AND** startup SHALL proceed without Kubernetes API write operations

### Requirement: Bootstrap Error Handling

The system SHALL provide clear error messages when bootstrap fails.

#### Scenario: Permission error with remediation
- **WHEN** bootstrap fails due to insufficient Kubernetes permissions
- **THEN** the error message SHALL state which operation failed
- **AND** the error message SHALL include the missing permission
- **AND** the error message SHALL suggest running `kubectl apply -f deploy/dev/rbac.yaml` for local dev
- **AND** the error message SHALL suggest checking cluster-admin access for remote clusters

#### Scenario: Kubernetes API unavailable
- **WHEN** the application starts
- **AND** the Kubernetes API is unreachable
- **THEN** the application SHALL exit with status code 1
- **AND** the error message SHALL state "cannot connect to Kubernetes"
- **AND** the error message SHALL include the configured API server address

#### Scenario: Invalid kubeconfig file format
- **WHEN** the application starts
- **AND** a kubeconfig file exists but contains invalid YAML
- **THEN** the application SHALL exit with status code 1
- **AND** the error message SHALL state which file is invalid
- **AND** the error message SHALL include the parse error details

### Requirement: Bootstrap Logging

The system SHALL log bootstrap operations for visibility and debugging.

#### Scenario: Resource creation logged
- **WHEN** a resource is created during bootstrap
- **THEN** an info-level log SHALL be written
- **AND** the log SHALL include resource type and name
- **AND** example: "Created namespace nightcrier"

#### Scenario: Resource exists logged
- **WHEN** a resource already exists during bootstrap
- **THEN** a debug-level log SHALL be written
- **AND** the log SHALL indicate the resource was skipped
- **AND** example: "Namespace nightcrier already exists, skipping"

#### Scenario: Bootstrap completion logged
- **WHEN** bootstrap completes successfully
- **THEN** an info-level log SHALL be written
- **AND** the log SHALL summarize what was done
- **AND** example: "Kubernetes bootstrap complete: 0 created, 5 existing"

#### Scenario: Bootstrap failure logged
- **WHEN** bootstrap fails
- **THEN** an error-level log SHALL be written before exiting
- **AND** the log SHALL include the failure reason
- **AND** the log SHALL include remediation guidance
