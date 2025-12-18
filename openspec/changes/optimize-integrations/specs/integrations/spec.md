# Integration Enhancements Specification

## ADDED Requirements

### Requirement: Slack Bot Token Configuration
The runner SHALL support Slack Bot Token authentication for enhanced notification capabilities while maintaining backward compatibility with webhooks.

#### Scenario: Bot token present
- **WHEN** the `SLACK_BOT_TOKEN` environment variable is set
- **THEN** the runner uses the Slack API with bot token authentication
- **AND** enables file upload and Block Kit message features

#### Scenario: Webhook-only configuration
- **WHEN** only `SLACK_WEBHOOK_URL` is set (no bot token)
- **THEN** the runner falls back to webhook-based notifications
- **AND** continues to function with basic text messages

#### Scenario: Missing configuration
- **WHEN** neither bot token nor webhook URL is configured
- **THEN** the runner logs a warning at startup
- **AND** skips Slack notifications for incidents

### Requirement: File Upload to Slack
The runner SHALL upload investigation reports and artifacts as files to Slack when configured with a bot token.

#### Scenario: Report file upload
- **WHEN** an investigation completes successfully
- **THEN** the `report.md` file is uploaded using `files.getUploadURLExternal` API
- **AND** the file is shared to the configured Slack channel
- **AND** the upload completes before sending the summary message

#### Scenario: Artifact bundle upload
- **WHEN** an investigation produces additional artifacts (logs, command outputs)
- **THEN** artifacts are bundled into a compressed archive
- **AND** the archive is uploaded as a follow-up file in the same thread

#### Scenario: Upload failure
- **WHEN** file upload fails (network error, API error, rate limit)
- **THEN** the runner logs the error with details
- **AND** sends a fallback notification with an error message
- **AND** continues processing other incidents

### Requirement: Rich Slack Messages with Block Kit
The runner SHALL use Slack Block Kit to format notifications with rich visual hierarchy and interactive elements.

#### Scenario: Block Kit message structure
- **WHEN** a bot token is configured
- **THEN** notifications use Block Kit format
- **AND** include a header block with incident ID and severity
- **AND** include a section block with incident metadata (namespace, resource)
- **AND** include an action block with a button linking to the uploaded report
- **AND** include a context block with generator attribution

#### Scenario: Severity-based formatting
- **WHEN** the incident severity is Critical or High
- **THEN** the header block uses red or orange styling
- **WHEN** the incident severity is Medium or Low
- **THEN** the header block uses yellow or green styling

#### Scenario: Expandable findings section
- **WHEN** the summary message is sent
- **THEN** key findings are included in a section block
- **AND** longer findings are truncated with "View full report" link

### Requirement: Thread Replies for Follow-Up Information
The runner SHALL post follow-up artifacts and updates as thread replies to maintain conversation context.

#### Scenario: Artifact in thread
- **WHEN** additional artifacts are uploaded after the initial report
- **THEN** the artifact message is posted as a reply in the original message thread
- **AND** uses the `thread_ts` from the initial message

#### Scenario: Error updates in thread
- **WHEN** an error occurs during investigation after initial notification
- **THEN** the error details are posted as a thread reply
- **AND** includes debugging information (error message, stack trace if available)

### Requirement: Agent Backend Configuration
The runner SHALL support multiple agent backends through a configuration-driven approach without requiring code changes.

#### Scenario: Backend selection
- **WHEN** the runner starts
- **THEN** it reads the `agent.backend` configuration value
- **AND** loads backend-specific configuration from `agent.config.<backend>`
- **AND** validates the selected backend configuration
- **AND** logs the selected backend name

#### Scenario: Claude backend
- **WHEN** `agent.backend` is set to "claude"
- **THEN** the runner invokes the Claude CLI with headless mode flags
- **AND** passes incident context via environment variables or context file

#### Scenario: Aider backend
- **WHEN** `agent.backend` is set to "aider"
- **THEN** the runner invokes Aider with the configured model and flags
- **AND** passes incident details in the initial prompt
- **AND** uses `--yes` flag for non-interactive operation

#### Scenario: OpenAI-compatible backend
- **WHEN** `agent.backend` is set to "openai-compatible"
- **THEN** the runner invokes the configured command with OpenAI API base URL
- **AND** sets `OPENAI_API_KEY` and `OPENAI_API_BASE` environment variables
- **AND** validates API base URL format at startup

#### Scenario: Invalid backend configuration
- **WHEN** the configured backend command is not found on PATH
- **THEN** the runner logs an error and exits with non-zero status
- **WHEN** required environment variables for the backend are missing
- **THEN** the runner logs an error and exits with non-zero status

### Requirement: Backend Command Abstraction
The runner SHALL define an abstraction layer for agent backends to enable consistent invocation patterns.

#### Scenario: Backend interface
- **WHEN** implementing a new backend
- **THEN** it implements the `AgentBackend` interface
- **AND** provides methods for command preparation, validation, and error handling

#### Scenario: Command preparation
- **WHEN** an incident is ready for investigation
- **THEN** the selected backend prepares the command with appropriate arguments
- **AND** injects environment variables specific to that backend
- **AND** creates context files if needed

#### Scenario: Backend validation
- **WHEN** the runner starts
- **THEN** each backend validates its configuration
- **AND** checks for required environment variables
- **AND** verifies command availability on system PATH

### Requirement: Prompt Template System
The runner SHALL use a template-based system for generating investigation prompts with customizable content.

#### Scenario: Template loading
- **WHEN** the runner initializes
- **THEN** it loads the prompt template from configuration
- **AND** parses template variables for incident context injection

#### Scenario: Context injection
- **WHEN** generating a prompt for an incident
- **THEN** the template is populated with incident-specific data (ID, severity, namespace, resource, message)
- **AND** produces a complete prompt string ready for agent invocation

#### Scenario: Custom templates
- **WHEN** a custom prompt template is provided in configuration
- **THEN** the runner uses the custom template instead of the default
- **AND** validates that required template variables are present

### Requirement: Few-Shot Example Injection
The runner SHALL inject few-shot examples into investigation prompts to improve agent performance and consistency.

#### Scenario: Example library loading
- **WHEN** the runner starts
- **THEN** it loads few-shot examples from configuration
- **AND** validates example structure (title, scenario, steps, finding, recommendation)

#### Scenario: Example injection in prompt
- **WHEN** generating an investigation prompt
- **THEN** configured few-shot examples are included in the prompt template
- **AND** examples are formatted consistently with clear delimiters
- **AND** examples demonstrate desired investigation methodology

#### Scenario: Example format validation
- **WHEN** loading examples from configuration
- **THEN** each example must include required fields: title, scenario, steps, finding, recommendation
- **AND** missing fields cause validation errors at startup

#### Scenario: No examples configured
- **WHEN** no few-shot examples are provided in configuration
- **THEN** the runner generates prompts without examples
- **AND** logs a warning about missing examples
- **AND** continues normal operation

### Requirement: Backend Environment Variables
The runner SHALL support backend-specific environment variable injection with secure handling.

#### Scenario: API key injection
- **WHEN** a backend requires an API key
- **THEN** the corresponding environment variable is passed to the agent process
- **AND** the variable name is configurable per backend (e.g., `ANTHROPIC_API_KEY`, `OPENAI_API_KEY`)

#### Scenario: Custom environment variables
- **WHEN** a backend specifies additional environment variables in configuration
- **THEN** all configured variables are injected into the agent process environment
- **AND** variables support value expansion from runner's environment

#### Scenario: Secret redaction in logs
- **WHEN** logging backend configuration or command execution
- **THEN** sensitive values (API keys, tokens) are redacted in log output
- **AND** redaction applies to environment variables containing "KEY", "TOKEN", "SECRET", or "PASSWORD"

### Requirement: Backend Timeout Configuration
The runner SHALL enforce configurable timeouts for agent invocations with backend-specific defaults.

#### Scenario: Backend timeout
- **WHEN** an agent investigation exceeds the configured timeout
- **THEN** the runner terminates the agent process
- **AND** logs a timeout error
- **AND** sends a notification indicating the investigation timed out
- **AND** cleans up workspace resources

#### Scenario: Per-backend timeout
- **WHEN** different backends are configured
- **THEN** each backend can specify its own timeout value
- **AND** the runner uses the backend-specific timeout during invocation
- **AND** falls back to a global default timeout if not specified

### Requirement: Slack API Rate Limiting Handling
The runner SHALL handle Slack API rate limits gracefully with retry logic.

#### Scenario: Rate limit response
- **WHEN** a Slack API call returns a rate limit error (HTTP 429)
- **THEN** the runner waits for the duration specified in the `Retry-After` header
- **AND** retries the request after the wait period
- **AND** implements exponential backoff for subsequent retries

#### Scenario: Maximum retry attempts
- **WHEN** Slack API calls fail repeatedly due to rate limits
- **THEN** the runner retries up to a configurable maximum number of times
- **AND** logs a failure after max retries are exhausted
- **AND** proceeds without sending the notification to avoid blocking other incidents

### Requirement: Configuration Validation at Startup
The runner SHALL validate all integration-related configuration at startup and fail fast on errors.

#### Scenario: Startup validation
- **WHEN** the runner starts
- **THEN** it validates Slack configuration (bot token or webhook URL presence)
- **AND** validates agent backend configuration (command existence, required environment variables)
- **AND** validates prompt template syntax
- **AND** validates few-shot example structure

#### Scenario: Validation failure
- **WHEN** any configuration validation fails
- **THEN** the runner logs detailed error messages
- **AND** exits with a non-zero status code
- **AND** does not start processing incidents

#### Scenario: Warning-level issues
- **WHEN** optional configuration is missing (e.g., no few-shot examples)
- **THEN** the runner logs a warning
- **AND** continues startup with degraded functionality

### Requirement: Backward Compatibility
The runner SHALL maintain backward compatibility with existing webhook-based deployments without configuration changes.

#### Scenario: Legacy webhook configuration
- **WHEN** only `SLACK_WEBHOOK_URL` is configured (no bot token, no new config fields)
- **THEN** the runner uses webhook notifications as before
- **AND** does not attempt file uploads or Block Kit formatting
- **AND** does not require agent backend configuration changes

#### Scenario: Default agent backend
- **WHEN** no `agent.backend` is specified in configuration
- **THEN** the runner defaults to the "claude" backend
- **AND** uses the original Claude invocation logic

#### Scenario: Mixed configuration
- **WHEN** both webhook URL and bot token are configured
- **THEN** the runner prefers bot token for richer notifications
- **AND** falls back to webhook if bot token authentication fails
