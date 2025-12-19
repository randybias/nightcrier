# Cluster Registry Specification

## ADDED Requirements

### Requirement: Multi-Cluster Configuration

The system SHALL support connecting to multiple kubernetes-mcp-server instances simultaneously through a declarative cluster configuration.

#### Scenario: Clusters defined in configuration file
- **Given** a configuration file with a `clusters` array
- **When** the application starts
- **Then** it SHALL establish connections to all enabled clusters
- **And** it SHALL validate each cluster configuration before connecting

#### Scenario: Cluster name uniqueness
- **Given** a configuration with multiple clusters
- **When** two clusters have the same `name` field
- **Then** the configuration SHALL be rejected with a validation error

#### Scenario: Required cluster fields
- **Given** a cluster configuration entry
- **When** the `name`, `mcp_endpoint`, or `kubeconfig` field is missing
- **Then** the configuration SHALL be rejected with a validation error

### Requirement: Backwards-Compatible Single-Cluster Mode

The system SHALL maintain backwards compatibility with single-cluster configuration using `mcp_endpoint`.

#### Scenario: Single endpoint configuration
- **Given** a configuration with `mcp_endpoint` set and no `clusters` array
- **When** the application starts
- **Then** it SHALL connect to the single MCP endpoint
- **And** it SHALL use the default kubeconfig path

#### Scenario: Mutual exclusivity
- **Given** a configuration with both `mcp_endpoint` and `clusters` defined
- **When** the configuration is validated
- **Then** it SHALL be rejected with an error stating both cannot be set

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
  - Last event timestamp (if any)
  - Last error message (if failed)
  - Event count
