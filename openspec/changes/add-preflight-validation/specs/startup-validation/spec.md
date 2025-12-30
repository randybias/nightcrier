# Specification: startup-validation

## Overview

Defines requirements for pre-flight validation that ensures critical files and configuration parameters are valid before application initialization begins.

## ADDED Requirements

### Requirement: Pre-flight validation MUST execute before component initialization

The system MUST validate all critical configuration and file dependencies immediately after config loading and before any component initialization.

#### Scenario: Valid configuration passes pre-flight validation quickly
**Given** all required files exist and configuration is valid
**When** pre-flight validation runs
**Then** validation completes in < 100ms
**And** application proceeds to component initialization
**And** log shows "pre-flight validation passed"

#### Scenario: Missing system prompt file fails validation
**Given** `agent_system_prompt_file` is configured
**And** the file does not exist at the specified path
**When** pre-flight validation runs
**Then** validation fails with error "Agent System Prompt File validation failed"
**And** error message includes the configured path
**And** error message includes remediation steps
**And** application exits without initializing components

#### Scenario: Empty system prompt configuration fails validation
**Given** `agent_system_prompt_file` is empty or not configured
**When** pre-flight validation runs
**Then** validation fails with error "agent system prompt file not configured"
**And** error message states the field is required
**And** application exits without initializing components

### Requirement: Triage kubeconfig files MUST be validated for enabled clusters

The system MUST verify that triage kubeconfig files exist and are readable for all clusters with triage enabled.

#### Scenario: All triage kubeconfig files exist
**Given** cluster "prod" has `triage.enabled = true`
**And** cluster "prod" has `triage.kubeconfig = "./kubeconfigs/prod.yaml"`
**And** the file exists at "./kubeconfigs/prod.yaml"
**When** pre-flight validation runs
**Then** validation passes for cluster "prod"

#### Scenario: Missing triage kubeconfig file fails validation
**Given** cluster "prod" has `triage.enabled = true`
**And** cluster "prod" has `triage.kubeconfig = "./kubeconfigs/missing.yaml"`
**And** the file does not exist
**When** pre-flight validation runs
**Then** validation fails with error "Triage Kubeconfig validation failed"
**And** error message includes cluster name "prod"
**And** error message includes the configured path
**And** error message includes remediation steps

#### Scenario: Disabled cluster kubeconfig is not validated
**Given** cluster "dev" has `triage.enabled = false`
**And** cluster "dev" has `triage.kubeconfig = "./kubeconfigs/nonexistent.yaml"`
**And** the file does not exist
**When** pre-flight validation runs
**Then** validation passes (disabled clusters are skipped)

#### Scenario: Cluster without kubeconfig path is skipped
**Given** cluster "staging" has `triage.enabled = true`
**And** cluster "staging" has no `triage.kubeconfig` set (empty)
**When** pre-flight validation runs
**Then** validation passes (clusters without kubeconfig are skipped)

### Requirement: Migrations path MUST be validated for database state storage

The system MUST verify that the migrations directory exists when using SQLite or PostgreSQL state storage.

#### Scenario: Migrations directory exists for SQLite
**Given** `state_storage.type = "sqlite"`
**And** `state_storage.migrations_path = "./migrations"`
**And** the directory exists at "./migrations"
**When** pre-flight validation runs
**Then** validation passes for migrations path

#### Scenario: Missing migrations directory fails validation
**Given** `state_storage.type = "postgres"`
**And** `state_storage.migrations_path = "./nonexistent"`
**And** the directory does not exist
**When** pre-flight validation runs
**Then** validation fails with error "Migrations Path validation failed"
**And** error message includes the configured path
**And** error message includes remediation steps

#### Scenario: Migrations path is not validated for filesystem storage
**Given** `state_storage.type = "filesystem"`
**When** pre-flight validation runs
**Then** migrations path validation is skipped

### Requirement: Validation errors MUST include remediation guidance

All validation errors MUST provide clear, actionable remediation steps to help operators fix the configuration.

#### Scenario: Error message includes remediation steps
**Given** a validation check fails
**When** the error is returned
**Then** error message includes a "Remediation:" section
**And** remediation has numbered, actionable steps
**And** error message includes a "Configuration reference:" section
**And** configuration reference shows the config key name
**And** configuration reference shows the current value
**And** configuration reference shows the source (file, env, or default)

#### Scenario: File not found error includes common locations
**Given** system prompt file validation fails
**When** the error is returned
**Then** remediation steps include checking the file path
**And** remediation steps suggest common file locations
**And** remediation steps mention file permissions

### Requirement: Validators MUST be easily extensible

The pre-flight system MUST provide a clear interface for adding new validators.

#### Scenario: New validator can be added by implementing interface
**Given** a developer needs to add a new validation check
**When** they implement the `Validator` interface
**And** they register the validator in `preflight.Validate()`
**Then** the new validator executes during pre-flight validation
**And** the validator receives the full configuration context
**And** the validator can return formatted errors with remediation

#### Scenario: Developer documentation exists for adding validators
**Given** a developer wants to add a new validator
**When** they read `internal/preflight/README.md`
**Then** documentation explains when to add validators
**And** documentation explains when NOT to add validators
**And** documentation includes a code example
**And** documentation explains error formatting standards

### Requirement: Configuration consistency validation MUST be included

Pre-flight validation MUST execute the existing `config.Validate()` method to catch configuration errors early.

#### Scenario: Invalid config values fail during pre-flight
**Given** configuration has `max_concurrent_agents = 0`
**When** pre-flight validation runs
**Then** validation fails via ConfigConsistencyValidator
**And** error message includes the config.Validate() error

#### Scenario: Missing API keys fail during pre-flight
**Given** no LLM API keys are configured
**When** pre-flight validation runs
**Then** validation fails with "at least one API key required"
**And** error comes from config.ValidateLLMAPIKeys()

### Requirement: Validation MUST execute in fail-fast mode

Pre-flight validation MUST stop at the first failure and report that error immediately.

#### Scenario: First validator failure stops execution
**Given** multiple validators are registered
**And** the second validator would fail
**And** the first validator fails
**When** pre-flight validation runs
**Then** execution stops after first validator
**And** only the first error is returned
**And** subsequent validators are not executed

#### Scenario: All validators execute when all pass
**Given** all validators are passing
**When** pre-flight validation runs
**Then** all validators execute
**And** validation completes successfully
**And** log shows "pre-flight validation passed"

### Requirement: Network checks MUST be explicitly excluded

Pre-flight validation MUST NOT include network connectivity checks to external services.

#### Scenario: Kubernetes cluster connectivity is not checked
**Given** Kubernetes cluster is unreachable
**When** pre-flight validation runs
**Then** validation does not check K8s connectivity
**And** validation passes (deferred to bootstrap phase)

#### Scenario: Object storage connectivity is not checked
**Given** object storage endpoint is unreachable
**When** pre-flight validation runs
**Then** validation does not check storage connectivity
**And** validation passes (deferred to storage init)

#### Scenario: MCP server connectivity is not checked
**Given** MCP server is unreachable
**When** pre-flight validation runs
**Then** validation does not check MCP connectivity
**And** validation passes (deferred to connection manager)

## Non-functional Requirements

### Performance
- Pre-flight validation must complete in < 100ms for typical configurations
- File existence checks use `os.Stat()` (fast syscall)
- No network I/O during validation

### Usability
- Error messages must be clear and actionable
- Remediation steps must be numbered and specific
- Configuration references must show key, value, and source

### Maintainability
- Validators follow a common interface
- New validators can be added without modifying core logic
- Developer documentation is co-located with code
