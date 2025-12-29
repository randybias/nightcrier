## ADDED Requirements

### Requirement: Single Run Mode

The system SHALL support a single-run execution mode for test harness integration.

#### Scenario: Single run flag accepted

- **WHEN** nightcrier is started with `--single-run` flag
- **THEN** the application SHALL start normally and connect to the MCP server
- **AND** the flag SHALL be recognized without error

#### Scenario: Process first event and exit

- **WHEN** nightcrier is running in single-run mode
- **AND** the first fault event is received
- **THEN** the event SHALL be fully processed (agent execution, reporting, notifications)
- **AND** after processing completes, nightcrier SHALL exit with code 0

#### Scenario: Subsequent events dropped

- **WHEN** nightcrier is running in single-run mode
- **AND** the first fault event processing has triggered shutdown
- **AND** additional fault events arrive
- **THEN** those events SHALL be dropped (not processed)
- **AND** the application SHALL continue its graceful shutdown

#### Scenario: Normal mode unchanged

- **WHEN** nightcrier is started without `--single-run` flag
- **THEN** the application SHALL run continuously
- **AND** all fault events SHALL be processed
- **AND** the application SHALL only exit on SIGINT/SIGTERM
