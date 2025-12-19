# Change Proposal: Improve Slack Notification Formatting

## Why

Current Slack notifications have poor formatting that makes them difficult to read and often truncate critical information in unhelpful ways. The metadata fields appear with excessive line breaks, and the root cause section is limited to 300 characters, which frequently cuts off in the middle of important context (e.g., cutting off after "```bash" without showing the actual command).

Improving the notification format will make incident triage faster and more effective by presenting information clearly and ensuring critical details are visible without needing to click through to the full report.

## Problem Statement

The current Slack notification format has several issues:

### Issue 1: Excessive Line Breaks in Metadata
The metadata section uses Slack's `fields` layout with 2-column display, but the format `*Label:*\nvalue` causes each label and value to appear on separate lines, creating excessive whitespace and making the message harder to scan.

**Current appearance:**
```
Cluster:
kind-events-test
Namespace:
default
Resource:
Pod/crash-test-pod
Reason:
CrashLoop
```

**Desired appearance:**
```
Cluster: kind-events-test        Namespace: default
Resource: Pod/crash-test-pod     Reason: CrashLoop
```

### Issue 2: Inadequate Root Cause Information
The root cause is limited to 300 characters and extracts only the first 2 lines from the report, which often results in:
- Truncation mid-sentence or mid-code-block
- Loss of critical context like commands, error messages, or recommendations
- Messages that end unhelpfully (e.g., "The container command is explicitly designed to crash: ```bash")

**Example of poor truncation:**
```
Root Cause (HIGH confidence):
The container command is explicitly designed to crash: ```bash
```

### Issue 3: No Structured Recommendations
The notification doesn't include any remediation steps or recommendations, forcing users to open the full report even for simple, common issues.

### Issue 4: Slack Limits Not Properly Considered
The current code has a simple 300-character truncation, but doesn't consider:
- Slack Block Kit limits: 3,000 chars per section block, 12,000 per markdown block
- Total message limit: 40,000 characters
- Webhook payload size limit: 16 KB

## Root Cause

The notification formatting issues stem from:

1. **Suboptimal Block Kit usage**: Using `fields` array with newline formatting instead of inline `Label: value` format
2. **Naive extraction logic**: `ExtractSummaryFromReport()` grabs first 2 lines and truncates at 300 chars without considering content structure
3. **Limited extraction scope**: Only extracts root cause + confidence, doesn't extract recommendations or next steps
4. **No smart truncation**: Truncates at character count rather than at sentence/paragraph boundaries

## Proposed Solution

### Phase 1: Fix Metadata Formatting (High Priority)

Change the metadata section from:
```go
Fields: []SlackText{
    {Type: "mrkdwn", Text: fmt.Sprintf("*Cluster:*\n%s", summary.Cluster)},
    {Type: "mrkdwn", Text: fmt.Sprintf("*Namespace:*\n%s", summary.Namespace)},
    // ...
}
```

To inline format:
```go
Fields: []SlackText{
    {Type: "mrkdwn", Text: fmt.Sprintf("*Cluster:* %s", summary.Cluster)},
    {Type: "mrkdwn", Text: fmt.Sprintf("*Namespace:* %s", summary.Namespace)},
    // ...
}
```

### Phase 2: Improve Root Cause Extraction (High Priority)

Enhance `ExtractSummaryFromReport()` to:
1. Extract full paragraphs up to 2,000 characters (section block limit: 3,000)
2. Include code blocks if present and they fit
3. Truncate at sentence boundaries, not mid-word
4. Add ellipsis with context like "... [see full report]"
5. Handle markdown code blocks properly (don't truncate in the middle)

### Phase 3: Add Recommendations Section (Medium Priority)

Extract top 2-3 recommendations from the report and display them as a bullet list:
```
*Recommended Actions:*
• Fix application command to run valid logic instead of crash loop
• Add liveness and readiness probes
• Set resource limits to improve QoS
```

### Phase 4: Add Executive Summary (Optional)

Include a brief executive summary section before the detailed root cause for quick context.

## Implementation Strategy

### Approach 1: Enhanced Extraction (Recommended)

Improve `ExtractSummaryFromReport()` to be more intelligent:
- Parse markdown sections properly
- Extract multiple sections (root cause, recommendations, summary)
- Use smart truncation that respects markdown structure
- Stay well under Slack limits (use 2,000 char limit for safety margin)

### Approach 2: Alternative Formatting

Keep extraction simple but improve how information is displayed:
- Use collapsible sections or multiple message blocks
- Link to specific sections of the report
- Use Slack's unfurl features if applicable

**Recommendation**: Use Approach 1 (enhanced extraction) as it provides immediate value without requiring external infrastructure changes.

## Impact

**User Impact:**
- High: Dramatically improves readability and information density
- Medium: Reduces need to click through to full report for simple issues
- Medium: Faster incident triage and response

**Technical Impact:**
- Low: Changes isolated to `slack.go` formatting logic
- Medium: Extraction logic requires more sophisticated markdown parsing
- No breaking changes to notification format or API

## Testing Strategy

1. **Unit Tests**: Test extraction with various report formats
   - Short reports (< 500 chars)
   - Long reports with code blocks
   - Reports with markdown formatting
   - Edge cases (missing sections, malformed markdown)

2. **Integration Tests**: Generate notifications and verify formatting
   - Check field layout in Slack
   - Verify code blocks render correctly
   - Test truncation at various lengths

3. **Manual Testing**: Send test notifications to Slack
   - View on desktop and mobile
   - Verify all information is readable
   - Test with real incident reports

## Slack Technical Limits

Based on Slack's documentation:

- **Section block text**: 3,000 characters max
- **Markdown text block**: 12,000 characters max
- **Total blocks**: 50 blocks max per message
- **Total message size**: 40,000 characters max
- **Webhook payload**: 16 KB max total (includes JSON overhead)

Our proposed limits:
- Root cause section: 2,000 characters (safe margin under 3,000)
- Recommendations section: 500 characters
- Total estimated message size: ~3,000 characters (well under limits)

## Success Criteria

- [ ] Metadata fields display inline with labels and values on same line
- [ ] Root cause section shows complete context (minimum 500 chars, up to 2,000)
- [ ] Code blocks in root cause are either shown completely or intelligently truncated
- [ ] Truncation happens at sentence boundaries, not mid-word
- [ ] Recommendations are extracted and displayed (when present)
- [ ] No Slack webhook errors due to size limits
- [ ] Message is readable on both desktop and mobile

## Dependencies

- None (changes are self-contained within `internal/reporting/slack.go`)

## References

- Slack Block Kit documentation: https://api.slack.com/block-kit
- Slack message truncation: https://api.slack.com/changelog/2018-04-truncating-really-long-messages
- Current implementation: `internal/reporting/slack.go:88-173` (SendIncidentNotification)
- Extraction logic: `internal/reporting/slack.go:330-393` (ExtractSummaryFromReport)

## Alternatives Considered

1. **Use Slack App with Interactive Components**: Too complex for current needs; incoming webhooks are simpler
2. **Send Multiple Messages**: Would clutter channels; single message is preferred
3. **Use Message Threads**: Requires Slack App (not available with webhooks)
4. **External Formatting Service**: Unnecessary complexity; can solve in-app

---

## Sources

Information about Slack limits from:
- [Truncating really long messages | Slack Developer Docs](https://docs.slack.dev/changelog/2018-truncating-really-long-messages/)
- [Blocks too long error at ~13200 characters · Issue #2509](https://github.com/slackapi/bolt-js/issues/2509)
- [Rate limits | Slack Developer Docs](https://docs.slack.dev/apis/web-api/rate-limits/)
