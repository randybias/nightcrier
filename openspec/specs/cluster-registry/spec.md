# cluster-registry Specification

## Purpose

Defines how Nightcrier manages connections to multiple Kubernetes clusters for fault monitoring and triage agent execution. The system separates monitored clusters (where faults are detected) from execution clusters (where triage agent Jobs run).

## Requirements

### Requirement: Multi-Cluster Configuration

The system SHALL support connecting to multiple kubernetes-mcp-server instances simultaneously through a declarative cluster configuration.

#### Scenario: Monitored clusters defined in configuration file
- **Given** a configuration file with a `monitored_clusters` array
- **When** the application starts
- **Then** it SHALL establish connections to all monitored clusters
- **And** it SHALL validate each cluster configuration before connecting

#### Scenario: Cluster name uniqueness
- **Given** a configuration with multiple monitored clusters
- **When** two clusters have the same `name` field
- **Then** the configuration SHALL be rejected with a validation error

#### Scenario: Required cluster fields
- **Given** a monitored cluster configuration entry
- **When** the `name` or `mcp.endpoint` field is missing
- **Then** the configuration SHALL be rejected with a validation error

#### Scenario: Zero clusters allowed at startup
- **Given** a configuration file
- **When** the `monitored_clusters` array is empty or missing
- **Then** the configuration SHALL be accepted with a warning
- **And** the system SHALL poll the database for cluster additions

### Requirement: Execution Cluster Configuration

The system SHALL support an `execution_clusters` array defining Kubernetes clusters where agent Jobs run.

#### Scenario: Execution cluster with all settings
- **When** an execution cluster is configured with all settings
- **Then** the configuration SHALL accept:
  - `name` (required, unique identifier)
  - `kubeconfig_path` (required, path to kubeconfig file)
  - `namespace` (optional, defaults to "nightcrier")
  - `runner_image` (optional, defaults to "nc-agent-runner:latest")
  - `image_pull_policy` (optional, defaults to "IfNotPresent")
  - `timeout` (optional, defaults to 600)
  - `memory_limit` (optional, defaults to "2Gi")
  - `cpu_limit` (optional, defaults to "1")
  - `cleanup_ttl` (optional, defaults to 3600)
  - `max_concurrent_agents` (optional, defaults to 10)

#### Scenario: Execution cluster name uniqueness
- **When** two execution clusters have the same `name` field
- **Then** the configuration SHALL be rejected with a validation error

#### Scenario: Execution cluster kubeconfig validation
- **When** an execution cluster specifies `kubeconfig_path`
- **And** the file does not exist
- **Then** the configuration SHALL be rejected with a validation error

### Requirement: Execution Cluster Pinning

The system SHALL support pinning monitored clusters to specific execution clusters.

#### Scenario: Explicit execution cluster reference
- **Given** a monitored cluster with `triage.execution_cluster: "triage-west"`
- **And** an execution cluster named "triage-west" exists
- **When** a fault event requires triage
- **Then** the agent Job SHALL be created in the "triage-west" execution cluster

#### Scenario: Invalid execution cluster reference
- **Given** a monitored cluster with `triage.execution_cluster: "nonexistent"`
- **And** no execution cluster named "nonexistent" exists
- **When** the configuration is validated
- **Then** the configuration SHALL be rejected with a validation error

#### Scenario: Default execution cluster
- **Given** a monitored cluster without `triage.execution_cluster` specified
- **And** at least one execution cluster is configured
- **When** a fault event requires triage
- **Then** the agent Job SHALL be created in the first configured execution cluster

#### Scenario: No execution cluster available
- **Given** a monitored cluster with `triage.enabled: true`
- **And** no execution clusters are configured
- **When** a fault event requires triage
- **Then** the system SHALL log an error
- **And** the event SHALL NOT be triaged
- **And** a notification SHALL indicate triage was skipped due to no execution cluster

### Requirement: Triage Configuration Per Cluster

The system SHALL support enabling or disabling triage per cluster through explicit configuration.

#### Scenario: Triage enabled requires target kubeconfig
- **Given** a monitored cluster configuration with `triage.enabled: true`
- **When** the `triage.target_kubeconfig_path` field is missing or empty
- **Then** the configuration SHALL be rejected with a validation error

#### Scenario: Triage disabled without kubeconfig
- **Given** a monitored cluster configuration with `triage.enabled: false`
- **When** the `triage.target_kubeconfig_path` field is missing
- **Then** the configuration SHALL be accepted
- **And** events from this cluster SHALL be logged but not triaged

#### Scenario: Target kubeconfig file validation
- **Given** a cluster with `triage.enabled: true`
- **And** a `triage.target_kubeconfig_path` path specified
- **When** the application starts
- **Then** it SHALL verify the kubeconfig file exists
- **And** it SHALL fail startup if the file is not readable

### Requirement: Never Guess Credentials

The system SHALL never guess or infer credentials for cluster access.

#### Scenario: No default kubeconfig fallback
- **Given** a monitored cluster configuration
- **When** `triage.enabled: true` and `triage.target_kubeconfig_path` is not specified
- **Then** the system SHALL NOT fall back to `~/.kube/config`
- **And** the configuration SHALL be rejected with an explicit error

#### Scenario: Explicit triage disabled
- **Given** a monitored cluster configuration with `triage.enabled: false`
- **When** a fault event is received from this cluster
- **Then** the system SHALL log the event
- **And** the system SHALL NOT spawn a triage agent
- **And** the system SHALL NOT send a notification

### Requirement: Database Cluster Storage

The system SHALL support storing cluster configurations in the database.

#### Scenario: Cluster stored in database
- **Given** a monitored cluster configuration inserted into the `monitored_clusters` table
- **When** the system polls the database
- **Then** the cluster SHALL be available for connection

#### Scenario: Database cluster overrides YAML
- **Given** a cluster named "prod-cluster" in both YAML and database
- **When** configuration is loaded
- **Then** the database version SHALL take precedence
- **And** the cluster's `source` field SHALL be "database"

#### Scenario: YAML cluster synced to database
- **Given** a cluster defined only in YAML
- **When** configuration is loaded
- **Then** the cluster SHALL be inserted into the database
- **And** the cluster's `source` field SHALL be "yaml"

#### Scenario: Execution cluster stored in database
- **Given** an execution cluster configuration inserted into the `execution_clusters` table
- **When** the system loads configuration
- **Then** the execution cluster SHALL be available for Job creation

### Requirement: SIGHUP Configuration Reload

The system SHALL reload configuration when receiving SIGHUP signal.

#### Scenario: SIGHUP triggers full reload
- **When** the process receives SIGHUP
- **Then** it SHALL re-read the YAML configuration file
- **And** it SHALL re-read cluster configuration from the database
- **And** database clusters SHALL override YAML clusters by name
- **And** all changed configuration SHALL be applied

#### Scenario: Reload failure handling
- **When** the process receives SIGHUP
- **And** the new configuration fails validation
- **Then** the previous valid configuration SHALL remain in effect
- **And** an error SHALL be logged

#### Scenario: Reload logging
- **When** configuration is reloaded successfully
- **Then** the system SHALL log at INFO level:
  - Number of monitored clusters added/removed
  - Number of execution clusters added/removed
  - Any other significant configuration changes

### Requirement: Cluster Reload Lifecycle

The system SHALL manage cluster connection lifecycle during configuration reload.

#### Scenario: New cluster connection on reload
- **Given** a running system with cluster "cluster-a"
- **When** SIGHUP is received
- **And** the new configuration adds "cluster-b"
- **Then** a connection to "cluster-b" SHALL be established
- **And** the connection to "cluster-a" SHALL continue unchanged

#### Scenario: Cluster removal on reload
- **Given** a running system with clusters "cluster-a" and "cluster-b"
- **When** SIGHUP is received
- **And** the new configuration removes "cluster-b"
- **Then** the connection to "cluster-b" SHALL be gracefully closed
- **And** any in-flight events from "cluster-b" SHALL complete processing
- **And** new events from "cluster-b" SHALL NOT be accepted

#### Scenario: Cluster update on reload
- **Given** a running system with cluster "cluster-a" using endpoint "http://old:8080"
- **When** SIGHUP is received
- **And** the new configuration changes "cluster-a" endpoint to "http://new:8080"
- **Then** the old connection SHALL be closed
- **And** a new connection to "http://new:8080" SHALL be established

### Requirement: Zero Clusters Startup Mode

The system SHALL start successfully even when no clusters are configured.

#### Scenario: Startup with empty cluster configuration
- **When** the application starts
- **And** no monitored clusters are configured in YAML or database
- **Then** the application SHALL start successfully
- **And** it SHALL log a warning about no clusters configured
- **And** it SHALL poll the database every 30 seconds for new clusters

#### Scenario: Cluster added after startup
- **When** the application is running with no clusters
- **And** a cluster is added to the database
- **Then** the next database poll SHALL detect the new cluster
- **And** the system SHALL establish a connection to the new cluster

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
- **And** the backoff SHALL not exceed 300 seconds (5 minutes)

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
- **And** the cluster target kubeconfig SHALL be used for agent execution
- **And** the cluster labels SHALL be available for filtering

### Requirement: Cluster-Specific Kubeconfig for Agents

The system SHALL pass the cluster-specific target kubeconfig to triage agents.

#### Scenario: Agent receives correct target kubeconfig
- **Given** a fault event from monitored cluster "prod-us-east-1"
- **And** cluster "prod-us-east-1" has `triage.target_kubeconfig_path` at `./kubeconfigs/prod-us-east-1-readonly.yaml`
- **When** the triage agent is spawned
- **Then** the agent SHALL receive the target kubeconfig for accessing the monitored cluster
- **And** the agent SHALL connect to the correct target cluster for investigation

#### Scenario: Target kubeconfig mounted read-only
- **Given** a triage agent container
- **When** the target kubeconfig is mounted
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
