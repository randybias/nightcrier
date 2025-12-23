# database-configuration Specification

## Purpose
TBD - created by archiving change migrate-state-to-sql. Update Purpose after archive.
## Requirements
### Requirement: Storage Configuration Section
The configuration file MUST support a `storage` section to define the backend type and connection details.

#### Scenario: Default Configuration
Given no `storage` configuration provided
When the configuration is loaded
Then it SHOULD default to `type: "sqlite"` and a local database path.

#### Scenario: Postgres Configuration
Given `storage.type` is set to "postgres"
Then the configuration MUST require `postgres` settings (DSN or components) to be valid.

### Requirement: Environment Variable Override
Storage configuration parameters MUST be overridable via environment variables.

#### Scenario: Postgres Password Override
Given a configuration file
And the environment variable `STORAGE_POSTGRES_PASSWORD` (or similar mapping) is set
Then the application SHOULD use the password from the environment variable.

