## MODIFIED Requirements

### Requirement: Multi-Cluster Configuration

The system SHALL support connecting to multiple kubernetes-mcp-server instances simultaneously through a declarative cluster configuration.

#### Scenario: Monitored clusters defined in configuration file

- **GIVEN** a configuration file with a `monitored_clusters` array
- **WHEN** the application starts
- **THEN** it SHALL establish connections to all monitored clusters
- **AND** it SHALL validate each cluster configuration before connecting

#### Scenario: Cluster name uniqueness

- **GIVEN** a configuration with multiple monitored clusters
- **WHEN** two clusters have the same `name` field
- **THEN** the configuration SHALL be rejected with a validation error

#### Scenario: Required cluster fields

- **GIVEN** a monitored cluster configuration entry
- **WHEN** the `name` or `mcp.endpoint` field is missing
- **THEN** the configuration SHALL be rejected with a validation error

#### Scenario: Zero clusters allowed

- **GIVEN** a configuration file
- **WHEN** the `monitored_clusters` array is empty or missing
- **THEN** the configuration SHALL be accepted with a warning
- **AND** the system SHALL poll the database for cluster additions

### Requirement: Triage Configuration Per Cluster

The system SHALL support enabling or disabling triage per cluster through explicit configuration.

#### Scenario: Triage enabled requires target kubeconfig

- **GIVEN** a monitored cluster configuration with `triage.enabled: true`
- **WHEN** the `triage.target_kubeconfig_path` field is missing or empty
- **THEN** the configuration SHALL be rejected with a validation error

#### Scenario: Triage disabled without kubeconfig

- **GIVEN** a monitored cluster configuration with `triage.enabled: false`
- **WHEN** the `triage.target_kubeconfig_path` field is missing
- **THEN** the configuration SHALL be accepted
- **AND** events from this cluster SHALL be logged but not triaged

#### Scenario: Target kubeconfig file validation

- **GIVEN** a cluster with `triage.enabled: true`
- **AND** a `triage.target_kubeconfig_path` path specified
- **WHEN** the application starts
- **THEN** it SHALL verify the kubeconfig file exists
- **AND** it SHALL fail startup if the file is not readable

### Requirement: Never Guess Credentials

The system SHALL never guess or infer credentials for cluster access.

#### Scenario: No default kubeconfig fallback

- **GIVEN** a monitored cluster configuration
- **WHEN** `triage.enabled: true` and `triage.target_kubeconfig_path` is not specified
- **THEN** the system SHALL NOT fall back to `~/.kube/config`
- **AND** the configuration SHALL be rejected with an explicit error

#### Scenario: Explicit triage disabled

- **GIVEN** a monitored cluster configuration with `triage.enabled: false`
- **WHEN** a fault event is received from this cluster
- **THEN** the system SHALL log the event
- **AND** the system SHALL NOT spawn a triage agent
- **AND** the system SHALL NOT send a notification

### Requirement: Cluster-Specific Kubeconfig for Agents

The system SHALL pass the cluster-specific target kubeconfig to triage agents.

#### Scenario: Agent receives correct target kubeconfig

- **GIVEN** a fault event from monitored cluster "prod-us-east-1"
- **AND** cluster "prod-us-east-1" has `triage.target_kubeconfig_path` at `./kubeconfigs/prod-us-east-1-readonly.yaml`
- **WHEN** the triage agent is spawned
- **THEN** the agent SHALL receive the target kubeconfig for accessing the monitored cluster
- **AND** the agent SHALL connect to the correct target cluster for investigation

#### Scenario: Target kubeconfig mounted read-only

- **GIVEN** a triage agent container
- **WHEN** the target kubeconfig is mounted
- **THEN** it SHALL be mounted read-only
- **AND** the agent SHALL NOT be able to modify the kubeconfig

## ADDED Requirements

### Requirement: Execution Cluster Pinning

The system SHALL support pinning monitored clusters to specific execution clusters.

#### Scenario: Explicit execution cluster reference

- **GIVEN** a monitored cluster with `triage.execution_cluster: "triage-west"`
- **AND** an execution cluster named "triage-west" exists
- **WHEN** a fault event requires triage
- **THEN** the agent Job SHALL be created in the "triage-west" execution cluster

#### Scenario: Invalid execution cluster reference

- **GIVEN** a monitored cluster with `triage.execution_cluster: "nonexistent"`
- **AND** no execution cluster named "nonexistent" exists
- **WHEN** the configuration is validated
- **THEN** the configuration SHALL be rejected with a validation error

#### Scenario: Default execution cluster

- **GIVEN** a monitored cluster without `triage.execution_cluster` specified
- **AND** at least one execution cluster is configured
- **WHEN** a fault event requires triage
- **THEN** the agent Job SHALL be created in the first configured execution cluster

#### Scenario: No execution cluster available

- **GIVEN** a monitored cluster with `triage.enabled: true`
- **AND** no execution clusters are configured
- **WHEN** a fault event requires triage
- **THEN** the system SHALL log an error
- **AND** the event SHALL NOT be triaged
- **AND** a notification SHALL indicate triage was skipped due to no execution cluster

### Requirement: Database Cluster Storage

The system SHALL support storing cluster configurations in the database.

#### Scenario: Cluster stored in database

- **GIVEN** a monitored cluster configuration inserted into the `monitored_clusters` table
- **WHEN** the system polls the database
- **THEN** the cluster SHALL be available for connection

#### Scenario: Database cluster overrides YAML

- **GIVEN** a cluster named "prod-cluster" in both YAML and database
- **WHEN** configuration is loaded
- **THEN** the database version SHALL take precedence
- **AND** the cluster's `source` field SHALL be "database"

#### Scenario: YAML cluster synced to database

- **GIVEN** a cluster defined only in YAML
- **WHEN** configuration is loaded
- **THEN** the cluster SHALL be inserted into the database
- **AND** the cluster's `source` field SHALL be "yaml"

#### Scenario: Execution cluster stored in database

- **GIVEN** an execution cluster configuration inserted into the `execution_clusters` table
- **WHEN** the system loads configuration
- **THEN** the execution cluster SHALL be available for Job creation

### Requirement: Cluster Reload Lifecycle

The system SHALL manage cluster connection lifecycle during configuration reload.

#### Scenario: New cluster connection on reload

- **GIVEN** a running system with cluster "cluster-a"
- **WHEN** SIGHUP is received
- **AND** the new configuration adds "cluster-b"
- **THEN** a connection to "cluster-b" SHALL be established
- **AND** the connection to "cluster-a" SHALL continue unchanged

#### Scenario: Cluster removal on reload

- **GIVEN** a running system with clusters "cluster-a" and "cluster-b"
- **WHEN** SIGHUP is received
- **AND** the new configuration removes "cluster-b"
- **THEN** the connection to "cluster-b" SHALL be gracefully closed
- **AND** any in-flight events from "cluster-b" SHALL complete processing
- **AND** new events from "cluster-b" SHALL NOT be accepted

#### Scenario: Cluster update on reload

- **GIVEN** a running system with cluster "cluster-a" using endpoint "http://old:8080"
- **WHEN** SIGHUP is received
- **AND** the new configuration changes "cluster-a" endpoint to "http://new:8080"
- **THEN** the old connection SHALL be closed
- **AND** a new connection to "http://new:8080" SHALL be established

## RENAMED Requirements

- FROM: `### Requirement: Multi-Cluster Configuration` scenario "Clusters defined in configuration file"
- TO: `### Requirement: Multi-Cluster Configuration` scenario "Monitored clusters defined in configuration file"
