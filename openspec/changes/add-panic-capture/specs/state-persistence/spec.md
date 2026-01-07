## ADDED Requirements

### Requirement: Panic Report Storage

The StateStore SHALL support storing panic reports for post-mortem analysis.

#### Scenario: Save panic report
- **GIVEN** nightcrier is running and connected to the database
- **WHEN** a panic occurs in the main goroutine or a monitored goroutine
- **THEN** a panic report SHALL be saved to the `panic_reports` table
- **AND** the report SHALL include timestamp, error message, full stack trace, and component name.

#### Scenario: Panic report fields
- **WHEN** a panic report is saved
- **THEN** the report SHALL contain:
  - `id`: Unique identifier (UUID)
  - `timestamp`: When the panic occurred
  - `error`: The panic value as a string
  - `stack`: Full stack trace from `debug.Stack()`
  - `component`: Which part of the system panicked (e.g., "main", "dispatcher", "mcp-client")
  - `instance_id`: The nightcrier instance identifier (if available)
  - `analyzed`: Boolean flag for future analysis tracking (default: false)

#### Scenario: Query unanalyzed panics
- **GIVEN** panic reports exist in the database with `analyzed=false`
- **WHEN** `GetUnanalyzedPanicReports` is called
- **THEN** the method SHALL return all panic reports where `analyzed=false`
- **AND** results SHALL be ordered by timestamp descending.

#### Scenario: Mark panic as analyzed
- **GIVEN** a panic report with `analyzed=false`
- **WHEN** `MarkPanicAnalyzed` is called with the report ID
- **THEN** the `analyzed` field SHALL be set to `true`.
