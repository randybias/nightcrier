# sqlite-testing Specification

## Purpose

Define requirements for comprehensive testing of the SQLite storage backend to ensure reliability before production use.

## ADDED Requirements

### Requirement: Edge Case Unit Tests

The SQLite storage tests MUST cover edge cases that could cause data corruption or unexpected behavior.

#### Scenario: NULL and empty value handling

**Given** a fault event with NULL optional fields (Resource, TriggeringEventID)
**When** CreateIncident is called
**Then** the incident SHOULD be stored successfully
**And** GetIncident SHOULD return the incident with nil values preserved

#### Scenario: Special character handling

**Given** a fault event with special characters in Context field (emoji, newlines, SQL injection attempts)
**When** CreateIncident is called
**Then** the incident SHOULD be stored without data loss or corruption
**And** GetIncident SHOULD return the exact original strings

#### Scenario: Long string handling

**Given** a fault event with a Context field of 10000+ characters
**When** CreateIncident is called
**Then** the incident SHOULD be stored successfully
**And** GetIncident SHOULD return the full Context without truncation

### Requirement: File-Based Database Tests

The SQLite storage tests MUST validate behavior with real database files, not just in-memory mode.

#### Scenario: Database file creation

**Given** a configuration pointing to a non-existent database file path
**When** New() is called
**Then** the database file SHOULD be created
**And** WAL and SHM journal files SHOULD be created alongside

#### Scenario: Database persistence

**Given** a SQLite store with one incident created
**When** the store is closed and reopened
**Then** GetIncident SHOULD return the previously stored incident

#### Scenario: Relative path resolution

**Given** a configuration with a relative database path like "./data/test.db"
**When** New() is called
**Then** the path SHOULD be resolved to an absolute path
**And** the database SHOULD be created at the resolved location

### Requirement: Migration Tests

The migration system MUST be tested to ensure schema changes apply correctly.

#### Scenario: Fresh database migration

**Given** a new empty SQLite database
**When** RunMigrations is called
**Then** all tables (fault_events, incidents, agent_executions, triage_reports) SHOULD be created
**And** all indexes SHOULD be created
**And** all constraints SHOULD be active

#### Scenario: Migration idempotency

**Given** a SQLite database with migrations already applied
**When** RunMigrations is called again
**Then** no errors SHOULD occur
**And** the schema SHOULD remain unchanged

### Requirement: Integration Tests

Integration tests MUST verify SQLite works correctly with the main application.

#### Scenario: Full incident lifecycle

**Given** a running nightcrier instance with SQLite storage
**When** an incident is created, updated, and completed
**Then** all state changes SHOULD be persisted to SQLite
**And** GetIncident SHOULD reflect each state correctly

#### Scenario: Concurrent operations

**Given** a SQLite store in WAL mode
**When** multiple goroutines perform concurrent writes
**Then** all operations SHOULD complete successfully
**And** no data SHOULD be lost or corrupted

### Requirement: Live Test Validation

Live tests MUST verify SQLite storage functions during real incident triage.

#### Scenario: Incident persistence during live test

**Given** a live test running with SQLite storage configured
**When** the agent completes investigation
**Then** the SQLite database file SHOULD exist
**And** querying the database SHOULD return the incident with status "resolved" or "failed"
**And** the triage_reports table SHOULD contain the investigation report

#### Scenario: Database inspection in test report

**Given** a completed live test
**When** the test report is generated
**Then** the report SHOULD include SQLite database statistics
**And** the report SHOULD confirm incident was persisted correctly
