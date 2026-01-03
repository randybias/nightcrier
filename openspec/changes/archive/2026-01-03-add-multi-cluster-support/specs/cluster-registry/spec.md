# Cluster Registry Specification

## ADDED Requirements

### Requirement: Multi-Cluster Configuration

The system SHALL support connecting to multiple kubernetes-mcp-server instances simultaneously through a declarative cluster configuration.

#### Scenario: Clusters defined in configuration file
- **Given** a configuration file with a `clusters` array
- **When** the application starts
- **Then** it SHALL establish connections to all clusters
- **And** it SHALL validate each cluster configuration before connecting

#### Scenario: Cluster name uniqueness
- **Given** a configuration with multiple clusters
- **When** two clusters have the same `name` field
- **Then** the configuration SHALL be rejected with a validation error

#### Scenario: Required cluster fields
- **Given** a cluster configuration entry
- **When** the `name` or `mcp.endpoint` field is missing
- **Then** the configuration SHALL be rejected with a validation error

#### Scenario: At least one cluster required
- **Given** a configuration file
- **When** the `clusters` array is empty or missing
- **Then** the configuration SHALL be rejected with a validation error

### Requirement: Triage Configuration Per Cluster

The system SHALL support enabling or disabling triage per cluster through explicit configuration.

#### Scenario: Triage enabled requires kubeconfig
- **Given** a cluster configuration with `triage.enabled: true`
- **When** the `triage.kubeconfig` field is missing or empty
- **Then** the configuration SHALL be rejected with a validation error

#### Scenario: Triage disabled without kubeconfig
- **Given** a cluster configuration with `triage.enabled: false`
- **When** the `triage.kubeconfig` field is missing
- **Then** the configuration SHALL be accepted
- **And** events from this cluster SHALL be logged but not triaged

#### Scenario: Kubeconfig file validation
- **Given** a cluster with `triage.enabled: true`
- **And** a `triage.kubeconfig` path specified
- **When** the application starts
- **Then** it SHALL verify the kubeconfig file exists
- **And** it SHALL fail startup if the file is not readable

### Requirement: Never Guess Credentials

The system SHALL never guess or infer credentials for cluster access.

#### Scenario: No default kubeconfig fallback
- **Given** a cluster configuration
- **When** `triage.enabled: true` and `triage.kubeconfig` is not specified
- **Then** the system SHALL NOT fall back to `~/.kube/config`
- **And** the configuration SHALL be rejected with an explicit error

#### Scenario: Explicit triage disabled
- **Given** a cluster configuration with `triage.enabled: false`
- **When** a fault event is received from this cluster
- **Then** the system SHALL log the event
- **And** the system SHALL NOT spawn a triage agent
- **And** the system SHALL NOT send a notification

### Requirement: Preflight Permission Validation

The system SHALL validate cluster permissions at startup for clusters with triage enabled.

#### Scenario: Permission check at startup
- **Given** a cluster with `triage.enabled: true`
- **And** a valid kubeconfig file
- **When** the application starts
- **Then** it SHALL run `kubectl auth can-i --list` against the cluster
- **And** it SHALL record the available permissions in memory

#### Scenario: Insufficient permissions warning
- **Given** a cluster with `triage.enabled: true`
- **When** the permission check reveals missing minimum permissions (get pods, get logs, get events)
- **Then** the system SHALL log a warning with the missing permissions
- **And** the system SHALL continue to start (non-fatal)

#### Scenario: Permission validation failure
- **Given** a cluster with `triage.enabled: true`
- **When** the `kubectl auth can-i` command fails (e.g., invalid kubeconfig)
- **Then** the system SHALL fail startup with an error message

### Requirement: Cluster Permissions in Workspace

The system SHALL write validated permissions to the incident workspace.

#### Scenario: Permissions file creation
- **Given** a fault event from a cluster with triage enabled
- **When** the workspace is created
- **Then** the system SHALL write `incident_cluster_permissions.json` to the workspace
- **And** the file SHALL contain the cluster name, validation timestamp, and permission flags

#### Scenario: Agent access to permissions
- **Given** an incident workspace with `incident_cluster_permissions.json`
- **When** the triage agent is invoked
- **Then** the agent SHALL have access to the permissions file
- **And** the agent can use this information to understand available kubectl operations

### Requirement: Connection Lifecycle Management

The system SHALL manage the lifecycle of each cluster connection independently.

#### Scenario: Independent connection failures
- **Given** connections to multiple clusters
- **When** one cluster connection fails
- **Then** other cluster connections SHALL continue operating normally
- **And** the failed connection SHALL attempt reconnection

#### Scenario: Exponential backoff on reconnection
- **Given** a cluster connection that has failed
- **When** reconnection is attempted
- **Then** the system SHALL use exponential backoff starting at 1 second
- **And** the backoff SHALL not exceed 60 seconds

#### Scenario: Connection status tracking
- **Given** a cluster connection
- **When** its status changes (connecting, active, failed)
- **Then** the status SHALL be logged with the cluster name
- **And** the status SHALL be available via health monitoring

### Requirement: Event Aggregation

The system SHALL aggregate events from all connected clusters into a unified event stream.

#### Scenario: Event fan-in
- **Given** events arriving from multiple clusters
- **When** events are received
- **Then** they SHALL be merged into a single event channel
- **And** each event SHALL include the source cluster name

#### Scenario: Cluster metadata on events
- **Given** a fault event from a cluster
- **When** the event is processed
- **Then** the cluster name SHALL be included in logs
- **And** the cluster kubeconfig SHALL be used for agent execution
- **And** the cluster labels SHALL be available for filtering

### Requirement: Cluster-Specific Kubeconfig for Agents

The system SHALL pass the cluster-specific kubeconfig to triage agents.

#### Scenario: Agent receives correct kubeconfig
- **Given** a fault event from cluster "prod-us-east-1"
- **And** cluster "prod-us-east-1" has kubeconfig at `./kubeconfigs/prod-us-east-1.yaml`
- **When** the triage agent is spawned
- **Then** the agent SHALL receive `--kubeconfig ./kubeconfigs/prod-us-east-1.yaml`
- **And** the agent SHALL connect to the correct cluster

#### Scenario: Kubeconfig mounted read-only
- **Given** a triage agent container
- **When** the kubeconfig is mounted
- **Then** it SHALL be mounted read-only
- **And** the agent SHALL NOT be able to modify the kubeconfig

### Requirement: Shared HTTP Transport

The system SHALL use a shared HTTP transport for efficient connection pooling across clusters.

#### Scenario: Connection pool configuration
- **Given** multiple cluster connections
- **When** HTTP connections are established
- **Then** they SHALL share a common transport with pooling
- **And** the transport SHALL support at least 200 idle connections

### Requirement: Health Monitoring

The system SHALL expose health status for all cluster connections.

#### Scenario: Health endpoint
- **Given** the health server is enabled
- **When** a GET request is made to `/health/clusters`
- **Then** the response SHALL include status for each cluster
- **And** the response SHALL include a summary of total/active/unhealthy counts

#### Scenario: Per-cluster health details
- **Given** a cluster connection
- **When** health is queried
- **Then** the response SHALL include:
  - Cluster name
  - Connection status
  - Triage enabled flag
  - Last event timestamp (if any)
  - Last error message (if failed)
  - Event count

### Requirement: Secrets Access Configuration

The system SHALL support an opt-in configuration for secrets and configmaps access.

#### Scenario: Secrets access disabled by default
- **Given** a cluster configuration without `triage.allow_secrets_access` specified
- **When** the permission validation runs
- **Then** secrets and configmaps access SHALL NOT be checked
- **And** the permissions file SHALL indicate `secrets_access_allowed: false`
- **And** a warning SHALL be included explaining how to enable Helm debugging

#### Scenario: Secrets access explicitly enabled
- **Given** a cluster configuration with `triage.allow_secrets_access: true`
- **When** the permission validation runs
- **Then** secrets and configmaps read permissions SHALL be checked
- **And** the permissions file SHALL indicate `secrets_access_allowed: true`
- **And** `can_get_secrets` and `can_get_configmaps` SHALL reflect actual RBAC

#### Scenario: Secrets access enabled but RBAC denies
- **Given** a cluster with `triage.allow_secrets_access: true`
- **And** the kubeconfig ServiceAccount lacks secrets read permission
- **When** permission validation runs
- **Then** `secrets_access_allowed` SHALL be true
- **And** `can_get_secrets` SHALL be false
- **And** a warning SHALL indicate RBAC denies secrets access

### Requirement: MCP API Key Placeholder

The system SHALL support a placeholder for future MCP server authentication.

#### Scenario: API key field ignored
- **Given** a cluster configuration with `mcp.api_key` set
- **When** the application connects to the MCP server
- **Then** the API key SHALL be ignored (not sent)
- **And** the system SHALL log that API key authentication is not yet implemented

#### Scenario: API key documented for future use
- **Given** a configuration example file
- **When** a user reads the configuration
- **Then** the `mcp.api_key` field SHALL be documented as a placeholder
- **And** the placeholder value SHALL clearly indicate it is not functional
