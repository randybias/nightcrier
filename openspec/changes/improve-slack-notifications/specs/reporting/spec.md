# Spec Delta: Improve Slack Notification Formatting

## MODIFIED Requirements

### Requirement: Slack Notification on Success
The runner SHALL send a notification to a configured Slack webhook upon successful completion of an investigation with clear, readable formatting.

#### Scenario: Successful investigation notification
- **WHEN** the report is generated successfully
- **THEN** a POST request is sent to the Slack webhook URL
- **AND** the payload uses Block Kit format with message attachments
- **AND** the attachment color matches the incident severity (red/yellow/green)
- **AND** the message includes incident ID, severity, cluster, resource, and duration
- **AND** metadata fields (cluster, namespace, resource, reason) display inline with labels and values on the same line
- **AND** the message includes root cause with sufficient detail (minimum 500 characters when available, up to 2,000 characters)
- **AND** the message includes recommended actions when present in the report (up to 3 recommendations)
- **AND** the message includes the filesystem path or URL to the full report

#### Scenario: Metadata field formatting
- **WHEN** building the Slack notification message
- **THEN** metadata fields SHALL use inline format: `*Label:* value`
- **AND** fields SHALL NOT use newline separators between label and value
- **AND** the fields array SHALL create a 2-column layout with labels and values on the same line
- **AND** the visual result SHALL minimize vertical space while maintaining readability

#### Scenario: Severity color mapping
- **WHEN** sending a notification for a critical/high severity incident
- **THEN** the message attachment color is `#E01E5A` (red)
- **WHEN** sending a notification for a warning/medium severity incident
- **THEN** the message attachment color is `#ECB22E` (yellow)
- **WHEN** sending a notification for an info/low severity incident
- **THEN** the message attachment color is `#2EB67D` (green)

## ADDED Requirements

### Requirement: Enhanced Root Cause Extraction
The runner SHALL extract root cause information from investigation reports with intelligent truncation that preserves content structure and readability.

#### Scenario: Full root cause extraction
- **WHEN** the root cause section is 2,000 characters or less
- **THEN** the entire root cause section SHALL be included in the notification
- **AND** no truncation occurs
- **AND** all markdown formatting is preserved

#### Scenario: Smart truncation for long root causes
- **WHEN** the root cause section exceeds 2,000 characters
- **THEN** the content SHALL be truncated to fit within 2,000 characters
- **AND** truncation SHALL occur at sentence boundaries (not mid-sentence)
- **AND** truncation SHALL not occur in the middle of code blocks
- **AND** a marker SHALL be added indicating truncation: `... [See full report for complete analysis]`
- **AND** the truncated content SHALL remain readable and coherent

#### Scenario: Code block handling in root cause
- **WHEN** the root cause contains fenced code blocks (```)
- **AND** the code block fits within the 2,000 character limit
- **THEN** the code block SHALL be included completely in the notification
- **AND** the code block SHALL be properly formatted with language identifier if present

#### Scenario: Code block truncation
- **WHEN** the root cause contains a code block that would exceed the 2,000 character limit
- **THEN** the runner SHALL either:
  - Include content up to the code block and add marker: `... [code omitted, see full report]`
  - OR include the code block and truncate after it if the code block is small
- **AND** the runner SHALL NOT truncate in the middle of a code block
- **AND** the notification SHALL indicate that additional content is available in the full report

#### Scenario: Sentence boundary detection
- **WHEN** truncating root cause content
- **THEN** the truncation point SHALL be located at the end of a complete sentence
- **AND** the runner SHALL look for sentence endings: `. `, `.\n`, `? `, `?\n`, `! `, `!\n`
- **AND** if no sentence boundary is found, truncation SHALL occur at a word boundary (space)
- **AND** truncation SHALL NOT occur mid-word except as a last resort

#### Scenario: Markdown preservation in extraction
- **WHEN** extracting root cause content
- **THEN** markdown formatting SHALL be preserved:
  - Bold text (`**text**`)
  - Italic text (`*text*` or `_text_`)
  - Inline code (`` `code` ``)
  - Lists (bullet and numbered)
  - Links (`[text](url)`)
- **AND** markdown SHALL render correctly in Slack's mrkdwn format

#### Scenario: Character limit compliance
- **WHEN** extracting any content for Slack notifications
- **THEN** the root cause section SHALL NOT exceed 2,000 characters
- **AND** the recommendations section SHALL NOT exceed 500 characters
- **AND** the total message payload SHALL NOT exceed 10 KB (safe margin under Slack's 16 KB limit)

### Requirement: Recommendations Extraction and Display
The runner SHALL extract and display actionable recommendations from investigation reports when present.

#### Scenario: Recommendations extraction
- **WHEN** the investigation report contains a recommendations section
- **THEN** the runner SHALL parse the section to extract actionable items
- **AND** the runner SHALL look for section headers: "Recommendations", "Next Steps", "Suggested Actions", "Remediation"
- **AND** the runner SHALL extract up to 3 recommendation items
- **AND** each recommendation SHALL be limited to 150 characters

#### Scenario: Recommendations formatting in Slack
- **WHEN** recommendations are extracted from the report
- **AND** at least one recommendation is found
- **THEN** a new section block SHALL be added to the Slack message
- **AND** the section SHALL be titled "*Recommended Actions:*"
- **AND** recommendations SHALL be formatted as a markdown bullet list
- **AND** each item SHALL be prefixed with "•" character

#### Scenario: No recommendations present
- **WHEN** the investigation report does not contain a recommendations section
- **OR** no recommendations can be parsed
- **THEN** the recommendations section SHALL be omitted from the Slack notification
- **AND** no placeholder or "None" message SHALL be shown

#### Scenario: Long recommendations truncation
- **WHEN** a single recommendation exceeds 150 characters
- **THEN** it SHALL be truncated to 147 characters
- **AND** ellipsis "..." SHALL be appended
- **AND** the truncation SHALL maintain sentence structure where possible

### Requirement: Slack Block Kit Compliance
The runner SHALL construct Slack messages that comply with all Block Kit limits and formatting requirements.

#### Scenario: Section block text limits
- **WHEN** constructing any section block
- **THEN** the text content SHALL NOT exceed 3,000 characters (Slack's limit)
- **AND** the runner SHALL use a safety margin of 2,000 characters for variable content

#### Scenario: Total message size limits
- **WHEN** constructing a complete Slack message
- **THEN** the total message SHALL NOT exceed 40,000 characters (Slack's limit)
- **AND** the runner SHALL target a typical message size of 2,000-4,000 characters
- **AND** the JSON payload SHALL NOT exceed 16 KB

#### Scenario: Block count limits
- **WHEN** constructing a Slack message
- **THEN** the total number of blocks SHALL NOT exceed 50 (Slack's limit)
- **AND** typical messages SHALL use 4-6 blocks (header, metadata, root cause, recommendations, context, actions)

#### Scenario: Markdown format compliance
- **WHEN** using markdown in Slack text fields
- **THEN** the content SHALL use Slack's mrkdwn format (not full markdown)
- **AND** supported elements include: bold (`*text*`), italic (`_text_`), code (`` `code` ``), code blocks (``` code ```), lists
- **AND** unsupported elements (tables, images) SHALL be handled gracefully

### Requirement: Fallback and Error Handling
The runner SHALL handle extraction errors and edge cases gracefully without failing notification delivery.

#### Scenario: Missing root cause section
- **WHEN** the investigation report does not contain a "Root Cause" section
- **THEN** the root cause field SHALL display: "See investigation report for details"
- **AND** the notification SHALL still be sent successfully

#### Scenario: Malformed markdown
- **WHEN** the investigation report contains malformed markdown
- **THEN** the extraction SHALL use best-effort parsing
- **AND** if parsing fails completely, the notification SHALL fall back to simple text extraction
- **AND** the notification SHALL still be sent (degraded but functional)

#### Scenario: Extraction timeout
- **WHEN** extracting content from a very large report (> 500 KB)
- **THEN** the extraction SHALL complete within 1 second
- **AND** if extraction takes longer, it SHALL be terminated
- **AND** a fallback message SHALL be used: "Report too large, see full report"

#### Scenario: Empty or invalid content
- **WHEN** the root cause section is empty or contains only whitespace
- **THEN** the notification SHALL display: "No root cause identified, see report"
- **AND** the notification SHALL still be sent

## REMOVED Requirements

None - this change only adds and modifies requirements without removing existing functionality.
