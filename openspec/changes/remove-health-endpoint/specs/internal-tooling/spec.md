## REMOVED Requirements

### Requirement: Health Monitoring HTTP Endpoint

The system previously exposed an HTTP endpoint on port 8080 for cluster health monitoring. This endpoint was introduced without proper specification and is being removed until a comprehensive observability plan is designed.

**Reason**: Premature implementation without proper observability design. An open port with partial monitoring creates false confidence in system observability.

**Migration**: None required - feature was not formally released or documented in specs.

#### Scenario: Health endpoint removal

- **WHEN** nightcrier starts
- **THEN** no HTTP server SHALL be started on port 8080
- **AND** no `--health-port` CLI flag SHALL be available
