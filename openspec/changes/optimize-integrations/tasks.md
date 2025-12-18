# Implementation Tasks (Phase 5)

## 1. Slack Bot Token Infrastructure
- [ ] 1.1 Add `SLACK_BOT_TOKEN` environment variable support to configuration loader
- [ ] 1.2 Create Slack API client abstraction (interface for webhook vs bot token)
- [ ] 1.3 Implement bot token authentication with Bearer token header
- [ ] 1.4 Add configuration validation for Slack settings at startup
- [ ] 1.5 Implement fallback logic: bot token -> webhook -> no notification

## 2. Slack File Upload Implementation
- [ ] 2.1 Implement `files.getUploadURLExternal` API call with error handling
- [ ] 2.2 Implement file upload to presigned URL with retry logic
- [ ] 2.3 Implement `files.completeUploadExternal` API call to finalize upload
- [ ] 2.4 Add support for uploading report.md as primary artifact
- [ ] 2.5 Add support for bundling and uploading additional artifacts (logs, outputs)
- [ ] 2.6 Capture file permalink from upload response for message linking
- [ ] 2.7 Add file size validation before upload (check Slack limits)
- [ ] 2.8 Implement upload failure fallback notification

## 3. Slack Block Kit Messages
- [ ] 3.1 Create Block Kit message builder module
- [ ] 3.2 Implement header block with incident ID and severity
- [ ] 3.3 Implement section block for incident metadata (namespace, resource, timestamp)
- [ ] 3.4 Implement divider block for visual separation
- [ ] 3.5 Implement action block with button linking to uploaded report
- [ ] 3.6 Implement context block with attribution text
- [ ] 3.7 Add severity-based color styling (red/orange/yellow/green)
- [ ] 3.8 Add expandable findings section with truncation logic
- [ ] 3.9 Add Block Kit message length validation and truncation

## 4. Slack Thread Replies
- [ ] 4.1 Capture and store `ts` (timestamp) from initial message response
- [ ] 4.2 Implement thread reply logic using `thread_ts` parameter
- [ ] 4.3 Post artifact uploads as thread replies
- [ ] 4.4 Post error updates as thread replies
- [ ] 4.5 Add thread reply fallback if initial message ID is unavailable

## 5. Slack API Rate Limiting
- [ ] 5.1 Implement rate limit detection (HTTP 429 response)
- [ ] 5.2 Implement `Retry-After` header parsing
- [ ] 5.3 Implement exponential backoff with jitter for retries
- [ ] 5.4 Add configurable maximum retry attempts
- [ ] 5.5 Add logging for rate limit events with backoff duration
- [ ] 5.6 Add metrics for rate limit occurrences

## 6. Agent Backend Abstraction
- [ ] 6.1 Define `AgentBackend` interface in Go (Name, PrepareCommand, ValidateConfig methods)
- [ ] 6.2 Create backend registry/factory for backend selection
- [ ] 6.3 Add `agent.backend` configuration field with validation
- [ ] 6.4 Add `agent.config.<backend>` configuration structure
- [ ] 6.5 Implement backend selection logic at startup
- [ ] 6.6 Add logging for selected backend with configuration summary (redact secrets)

## 7. Claude Backend Implementation
- [ ] 7.1 Implement `ClaudeBackend` struct implementing `AgentBackend` interface
- [ ] 7.2 Add Claude-specific command preparation (headless flags, context file)
- [ ] 7.3 Add `ANTHROPIC_API_KEY` environment variable validation
- [ ] 7.4 Add command availability check (`claude` binary on PATH)
- [ ] 7.5 Migrate existing Claude invocation logic to use new backend

## 8. Aider Backend Implementation
- [ ] 8.1 Implement `AiderBackend` struct implementing `AgentBackend` interface
- [ ] 8.2 Add Aider-specific command preparation (model, --yes, --no-auto-commits flags)
- [ ] 8.3 Add support for configurable model selection
- [ ] 8.4 Add API key environment variable injection (ANTHROPIC_API_KEY, OPENAI_API_KEY, etc.)
- [ ] 8.5 Add command availability check (`aider` binary on PATH)
- [ ] 8.6 Add prompt formatting for Aider's expected input format

## 9. OpenAI-Compatible Backend Implementation
- [ ] 9.1 Implement `OpenAICompatibleBackend` struct implementing `AgentBackend` interface
- [ ] 9.2 Add `OPENAI_API_BASE` URL configuration and validation
- [ ] 9.3 Add `OPENAI_API_KEY` environment variable injection
- [ ] 9.4 Add base URL format validation (must be valid HTTP/HTTPS URL)
- [ ] 9.5 Add command preparation with API base URL override
- [ ] 9.6 Support custom command configuration (not just Aider)

## 10. Backend Configuration Schema
- [ ] 10.1 Define configuration schema for backend settings (YAML/JSON)
- [ ] 10.2 Add per-backend command and args configuration
- [ ] 10.3 Add per-backend environment variable configuration with expansion
- [ ] 10.4 Add per-backend timeout configuration
- [ ] 10.5 Implement configuration loading and parsing
- [ ] 10.6 Add configuration validation at startup with clear error messages

## 11. Backend Environment Variable Handling
- [ ] 11.1 Implement environment variable injection for agent processes
- [ ] 11.2 Add support for variable expansion from runner's environment
- [ ] 11.3 Implement secret redaction in logs (KEY, TOKEN, SECRET, PASSWORD patterns)
- [ ] 11.4 Add validation for required environment variables per backend
- [ ] 11.5 Add logging of injected variables (with redaction)

## 12. Backend Timeout Handling
- [ ] 12.1 Add per-backend timeout configuration field
- [ ] 12.2 Implement timeout enforcement using context.WithTimeout
- [ ] 12.3 Add process termination logic on timeout
- [ ] 12.4 Add timeout error logging with backend and duration details
- [ ] 12.5 Add notification generation for timed-out investigations
- [ ] 12.6 Add workspace cleanup after timeout

## 13. Prompt Template System
- [ ] 13.1 Create prompt template loader from configuration
- [ ] 13.2 Implement template parsing with Go text/template
- [ ] 13.3 Define template variable structure (IncidentID, Severity, Namespace, Resource, etc.)
- [ ] 13.4 Implement context injection into template
- [ ] 13.5 Add template validation (check for required variables)
- [ ] 13.6 Support custom template configuration override
- [ ] 13.7 Create default prompt template with investigation guidelines
- [ ] 13.8 Add template rendering error handling and logging

## 14. Few-Shot Example System
- [ ] 14.1 Define few-shot example configuration schema (title, scenario, steps, finding, recommendation)
- [ ] 14.2 Implement example library loading from configuration
- [ ] 14.3 Add example validation (required fields, formatting)
- [ ] 14.4 Implement example injection into prompt template
- [ ] 14.5 Create default example library (CrashLoopBackOff, ImagePullBackOff, OOMKilled, etc.)
- [ ] 14.6 Add consistent example formatting with delimiters
- [ ] 14.7 Add warning for missing examples (non-fatal)

## 15. Backward Compatibility
- [ ] 15.1 Test webhook-only configuration continues working unchanged
- [ ] 15.2 Implement default backend selection (claude) when not specified
- [ ] 15.3 Test mixed configuration (webhook + bot token) with preference logic
- [ ] 15.4 Ensure no breaking changes to existing configuration format
- [ ] 15.5 Add migration documentation for webhook -> bot token transition

## 16. Testing
- [ ] 16.1 Unit tests for Slack API client (webhook and bot token modes)
- [ ] 16.2 Unit tests for file upload flow with mocked Slack API
- [ ] 16.3 Unit tests for Block Kit message builder
- [ ] 16.4 Unit tests for thread reply logic
- [ ] 16.5 Unit tests for rate limit handling with exponential backoff
- [ ] 16.6 Unit tests for each backend implementation (command preparation, validation)
- [ ] 16.7 Unit tests for prompt template rendering and variable injection
- [ ] 16.8 Unit tests for few-shot example loading and validation
- [ ] 16.9 Integration test: end-to-end file upload to Slack sandbox channel
- [ ] 16.10 Integration test: end-to-end investigation with Aider backend
- [ ] 16.11 Integration test: end-to-end investigation with OpenAI-compatible backend
- [ ] 16.12 Integration test: backward compatibility with webhook-only configuration
- [ ] 16.13 Integration test: timeout handling for long-running investigations
- [ ] 16.14 Integration test: configuration validation on startup with invalid settings

## 17. Documentation
- [ ] 17.1 Document Slack Bot Token setup (app creation, scopes, installation)
- [ ] 17.2 Document configuration options for Slack (bot token vs webhook)
- [ ] 17.3 Document agent backend configuration schema with examples
- [ ] 17.4 Document supported backends (Claude, Aider, OpenAI-compatible)
- [ ] 17.5 Document prompt template customization
- [ ] 17.6 Document few-shot example configuration format
- [ ] 17.7 Document migration path from webhook to bot token
- [ ] 17.8 Document troubleshooting for common configuration errors
- [ ] 17.9 Add example configurations for each backend
- [ ] 17.10 Add screenshots of Block Kit messages and file uploads

## 18. Verification
- [ ] 18.1 Verify files appear in Slack channel with correct permissions
- [ ] 18.2 Verify Block Kit messages render correctly with all components
- [ ] 18.3 Verify thread replies maintain conversation context
- [ ] 18.4 Verify buttons link to uploaded files correctly
- [ ] 18.5 Verify Claude backend works with existing behavior
- [ ] 18.6 Verify Aider backend invokes correctly and produces reports
- [ ] 18.7 Verify OpenAI-compatible backend works with test endpoint
- [ ] 18.8 Verify few-shot examples improve investigation quality (manual review)
- [ ] 18.9 Verify timeout handling terminates processes correctly
- [ ] 18.10 Verify rate limit handling doesn't block other incidents
- [ ] 18.11 Verify backward compatibility with existing deployments
- [ ] 18.12 Verify secret redaction in logs with real API keys
