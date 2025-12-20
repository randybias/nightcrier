## MODIFIED Requirements

### Requirement: MCP Event Subscription

The system SHALL connect to kubernetes-mcp-server via MCP StreamableHTTP protocol and subscribe to fault events.

#### Scenario: Successful MCP connection
- **GIVEN** a valid `K8S_CLUSTER_MCP_ENDPOINT` URL
- **WHEN** the runner starts
- **THEN** it connects via StreamableHTTP, initializes a session, and subscribes with `events_subscribe(mode="resource-faults")`

#### Scenario: Event reception
- **GIVEN** an active MCP subscription
- **WHEN** a fault occurs in the cluster
- **THEN** the runner receives a `logging/message` notification with `logger="kubernetes/resource-faults"` containing FaultEvent data with flat structure (faultId, resource, context, faultType, severity, timestamp)

#### Scenario: Event parsing
- **GIVEN** a fault notification
- **WHEN** the event is received
- **THEN** the JSON is parsed into a FaultEvent struct with fields: faultId (from upstream), subscriptionId, cluster, resource (including uid), context, faultType, severity, timestamp
- **AND** ReceivedAt is set locally for internal timing

#### Scenario: FaultID from upstream
- **GIVEN** a fault notification from kubernetes-mcp-server
- **WHEN** the event is parsed
- **THEN** the FaultID field is populated from the upstream `faultId` field (not generated locally)
- **AND** the FaultID is a stable identifier (same fault condition = same FaultID)

#### Scenario: Helper method access
- **GIVEN** a parsed FaultEvent
- **WHEN** helper methods (GetResourceName, GetResourceKind, GetNamespace, GetSeverity, GetContext, GetTimestamp, GetReason) are called
- **THEN** the correct values are returned from the flat structure
