# Project Identity Specification

## ADDED Requirements

### Requirement: Project Naming Convention

The project SHALL use the name "nightcrier" as its canonical identity across all code, configuration, and documentation artifacts.

#### Scenario: Go Module Path
- **Given** a developer imports the module
- **When** they add it to their go.mod
- **Then** the import path SHALL be `github.com/rbias/nightcrier`

#### Scenario: MCP Client Identification
- **Given** the event runner connects to an MCP server
- **When** it sends its client implementation details
- **Then** the client name SHALL be `nightcrier`

#### Scenario: Binary Name
- **Given** a user builds the project
- **When** they run `go build ./cmd/nightcrier`
- **Then** the output binary SHALL be named `nightcrier`

#### Scenario: Configuration Search Path
- **Given** the application starts without an explicit config file
- **When** it searches for configuration
- **Then** it SHALL search `/etc/nightcrier/` as a system-wide config location

#### Scenario: Docker Image Name
- **Given** the agent container is built
- **When** using default Makefile settings
- **Then** the image SHALL be named `nightcrier-agent`
