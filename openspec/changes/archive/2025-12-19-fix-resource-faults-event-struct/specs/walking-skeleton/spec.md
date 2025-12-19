## MODIFIED Requirements

### Requirement: MCP Event Subscription

The system SHALL connect to kubernetes-mcp-server via MCP StreamableHTTP protocol and subscribe to fault events.

#### Scenario: Successful MCP connection
- **GIVEN** a valid `K8S_CLUSTER_MCP_ENDPOINT` URL
- **WHEN** the runner starts
- **THEN** it connects via StreamableHTTP, initializes a session, and subscribes with `events_subscribe(mode=<configured-mode>)`

#### Scenario: Event reception (faults mode)
- **GIVEN** an active MCP subscription with mode="faults"
- **WHEN** a fault occurs in the cluster
- **THEN** the runner receives a `logging/message` notification with `logger="kubernetes/faults"` containing FaultEvent data with nested event structure

#### Scenario: Event reception (resource-faults mode)
- **GIVEN** an active MCP subscription with mode="resource-faults"
- **WHEN** a fault occurs in the cluster
- **THEN** the runner receives a `logging/message` notification with `logger="kubernetes/resource-faults"` containing FaultEvent data with flat structure (resource, context, faultType, severity, timestamp)

#### Scenario: Event parsing (faults mode)
- **GIVEN** a fault notification from faults mode
- **WHEN** the event is received
- **THEN** the JSON is parsed into a FaultEvent struct with nested event object (subscriptionId, cluster, event.namespace, event.reason, event.message, event.involvedObject)

#### Scenario: Event parsing (resource-faults mode)
- **GIVEN** a fault notification from resource-faults mode
- **WHEN** the event is received
- **THEN** the JSON is parsed into a FaultEvent struct with flat structure (subscriptionId, cluster, resource.kind, resource.name, resource.namespace, context, faultType, severity, timestamp)

#### Scenario: Helper method compatibility
- **GIVEN** a parsed FaultEvent from either subscription mode
- **WHEN** helper methods (GetResourceName, GetResourceKind, GetNamespace, GetSeverity) are called
- **THEN** the correct values are returned regardless of which mode was used
