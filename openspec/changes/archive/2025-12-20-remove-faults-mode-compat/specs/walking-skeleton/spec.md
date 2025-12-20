## MODIFIED Requirements

### Requirement: MCP Event Subscription

The system SHALL connect to kubernetes-mcp-server via MCP StreamableHTTP protocol and subscribe to fault events using resource-faults mode.

#### Scenario: Successful MCP connection
- **GIVEN** a valid `K8S_CLUSTER_MCP_ENDPOINT` URL
- **WHEN** the runner starts
- **THEN** it connects via StreamableHTTP, initializes a session, and subscribes with `events_subscribe(mode="resource-faults")`

#### Scenario: Event reception
- **GIVEN** an active MCP subscription
- **WHEN** a fault occurs in the cluster
- **THEN** the runner receives a `logging/message` notification with `logger="kubernetes/resource-faults"` containing FaultEvent data with flat structure (resource, context, faultType, severity, timestamp)

#### Scenario: Event parsing
- **GIVEN** a fault notification
- **WHEN** the event is received
- **THEN** the JSON is parsed into a FaultEvent struct with flat structure (faultId, subscriptionId, cluster, resource.kind, resource.name, resource.namespace, context, faultType, severity, timestamp)
- **AND** a unique faultId (UUID) is generated for tracing

#### Scenario: Helper method access
- **GIVEN** a parsed FaultEvent
- **WHEN** helper methods (GetResourceName, GetResourceKind, GetNamespace, GetSeverity, GetContext, GetTimestamp, GetReason) are called
- **THEN** the correct values are returned from the flat resource-faults structure

## REMOVED Requirements

### Requirement: Dual-mode subscription support
**Reason**: kubernetes-mcp-server has removed the deprecated "faults" mode. Only resource-faults mode is supported.

**Migration**: Update kubernetes-mcp-server to a version supporting resource-faults mode. Configuration variable `SUBSCRIBE_MODE` is no longer needed (or should be hardcoded to "resource-faults").

**Removed scenarios:**
- Event reception (faults mode)
- Event parsing (faults mode)
- Helper method compatibility between modes

**Naming changes:**
- `EventID` renamed to `FaultID` to accurately reflect that we only handle fault notifications, not general events
