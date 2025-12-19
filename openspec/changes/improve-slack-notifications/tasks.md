# Implementation Tasks: Improve Slack Notification Formatting

## Overview
This task list covers improvements to Slack notification formatting to enhance readability and ensure critical information is fully visible.

## Pre-Implementation Tasks
- [ ] Review current Slack notification examples to catalog all formatting issues
- [ ] Identify most common root cause patterns in production reports
- [ ] Document current vs desired appearance with screenshots/mockups
- [ ] Verify Slack Block Kit rendering behavior with test messages

## Phase 1: Fix Metadata Formatting (High Priority)

### Task 1.1: Update Metadata Field Format
- [ ] Modify `SendIncidentNotification()` in `internal/reporting/slack.go:114-119`
- [ ] Change from `*Label:*\n%s` format to `*Label:* %s` format
- [ ] Update all four metadata fields (Cluster, Namespace, Resource, Reason)
- [ ] Maintain 2-column layout using Slack fields array

### Task 1.2: Test Metadata Display
- [ ] Create unit test for metadata formatting
- [ ] Send test message to Slack webhook
- [ ] Verify appearance on Slack desktop client
- [ ] Verify appearance on Slack mobile client
- [ ] Confirm 2-column layout works as expected

## Phase 2: Improve Root Cause Extraction (High Priority)

### Task 2.1: Enhance Extraction Function
- [ ] Create new `ExtractSummaryFromReportEnhanced()` function as replacement
- [ ] Implement section parser to identify "## Root Cause" section
- [ ] Extract full paragraphs (not just first 2 lines)
- [ ] Set character limit to 2,000 (safe margin under 3,000 Slack limit)
- [ ] Implement smart truncation at sentence boundaries

### Task 2.2: Handle Code Blocks in Root Cause
- [ ] Detect markdown code blocks (fenced with ``` or indented)
- [ ] If code block fits in limit, include it completely
- [ ] If code block doesn't fit, either:
  - Option A: Include code block and truncate after it
  - Option B: Truncate before code block with "... [code omitted, see report]"
- [ ] Add logic to prevent truncation in middle of code block
- [ ] Handle inline code spans properly (preserve backticks)

### Task 2.3: Implement Smart Truncation
- [ ] Find last complete sentence before character limit
- [ ] Avoid truncating in middle of:
  - Sentences (look for `. ` or `.\n`)
  - Code blocks (``` boundaries)
  - Lists (bullet points)
  - Bold/italic markers (`**` or `*`)
- [ ] Add suffix: `...\n\n_[See full report for complete analysis]_`
- [ ] Ensure total length stays under 2,000 characters

### Task 2.4: Update Confidence Extraction
- [ ] Keep existing confidence level detection logic
- [ ] Ensure it works with enhanced extraction
- [ ] Handle edge cases where confidence is in different formats

### Task 2.5: Write Unit Tests for Extraction
- [ ] Test with short root cause (< 300 chars) - should return full text
- [ ] Test with medium root cause (300-2000 chars) - should return full text
- [ ] Test with long root cause (> 2000 chars) - should truncate smartly
- [ ] Test with code blocks:
  - Small code block that fits
  - Large code block that doesn't fit
  - Multiple code blocks
- [ ] Test with lists and bullet points
- [ ] Test with markdown formatting (bold, italic, inline code)
- [ ] Test with missing root cause section
- [ ] Test with malformed markdown

### Task 2.6: Integration with Main Flow
- [ ] Replace call to `ExtractSummaryFromReport()` in `cmd/runner/main.go:328`
- [ ] Update to use new enhanced extraction function
- [ ] Maintain backward compatibility (fallback to old function if needed)
- [ ] Update error handling

## Phase 3: Add Recommendations Section (Medium Priority)

### Task 3.1: Extract Recommendations from Report
- [ ] Add function `ExtractRecommendations()` to parse "## " sections
- [ ] Look for sections like:
  - "Recommendations"
  - "Next Steps"
  - "Suggested Actions"
  - "Remediation"
- [ ] Extract up to 3 bullet points from recommendations section
- [ ] Limit total recommendations text to 500 characters

### Task 3.2: Add Recommendations to Slack Message
- [ ] Update `IncidentSummary` struct to include `Recommendations []string`
- [ ] Add new Slack block for recommendations section
- [ ] Format as markdown bullet list
- [ ] Place recommendations block after root cause, before context

### Task 3.3: Handle Missing Recommendations
- [ ] Skip recommendations block if no recommendations found
- [ ] Don't show "Recommendations: None" or similar
- [ ] Gracefully degrade to current format

### Task 3.4: Test Recommendations Display
- [ ] Test with reports that have recommendations
- [ ] Test with reports that don't have recommendations
- [ ] Test with very long recommendations list
- [ ] Verify formatting in Slack

## Phase 4: Update Tests

### Task 4.1: Update Existing Unit Tests
- [ ] Update tests in `internal/reporting/slack_test.go`
- [ ] Modify expected message format for new metadata layout
- [ ] Add test cases for new extraction logic
- [ ] Update mock data to reflect realistic report content

### Task 4.2: Add Integration Tests
- [ ] Create test that generates full incident report
- [ ] Extract summary using new function
- [ ] Build Slack message
- [ ] Verify message structure and content
- [ ] Check message size is under limits

### Task 4.3: Add Markdown Parsing Tests
- [ ] Test with various markdown constructs:
  - Headings (##, ###)
  - Lists (bullet, numbered)
  - Code blocks (fenced, indented)
  - Inline code
  - Bold and italic
  - Links
- [ ] Verify all markdown is preserved or intelligently handled

## Phase 5: Documentation and Polish

### Task 5.1: Update Code Documentation
- [ ] Add comprehensive docstring to new extraction function
- [ ] Document smart truncation algorithm
- [ ] Document Slack limits and why we chose our limits
- [ ] Add examples in comments

### Task 5.2: Update Configuration Documentation
- [ ] Document new notification format
- [ ] Update any relevant README sections
- [ ] Add examples of what notifications look like

### Task 5.3: Performance Optimization
- [ ] Profile extraction function with large reports (50+ KB)
- [ ] Ensure extraction completes in < 100ms for typical reports
- [ ] Optimize string operations if needed (use bytes.Buffer)

## Phase 6: Manual Testing and Validation

### Task 6.1: Test with Real Incidents
- [ ] Generate incidents with various fault types:
  - CrashLoopBackOff
  - ImagePullBackOff
  - OOMKilled
  - ConfigMap/Secret issues
- [ ] Verify notifications look good for each type
- [ ] Check truncation works correctly

### Task 6.2: Test Edge Cases
- [ ] Very short root cause (1 sentence)
- [ ] Very long root cause (10+ paragraphs)
- [ ] Root cause with large code block (200+ lines)
- [ ] Root cause with special characters
- [ ] Root cause with URLs
- [ ] Missing root cause section

### Task 6.3: Cross-Platform Verification
- [ ] View notifications on Slack desktop (Mac/Windows/Linux)
- [ ] View notifications on Slack mobile (iOS/Android)
- [ ] View notifications on Slack web interface
- [ ] Verify formatting is consistent across platforms

### Task 6.4: Webhook Testing
- [ ] Verify no webhook errors (HTTP 200 response)
- [ ] Check Slack rate limiting doesn't trigger
- [ ] Test with multiple rapid notifications
- [ ] Verify payload size is under 16 KB

## Rollback Plan
- [ ] Document rollback procedure
- [ ] Keep old extraction function available as fallback
- [ ] Add feature flag or config option to switch between old/new format
- [ ] Test rollback in staging environment

## Success Metrics
- [ ] All tests pass (unit + integration)
- [ ] Metadata displays on single lines (inline format)
- [ ] Root cause shows minimum 500 characters (when available)
- [ ] Code blocks either display fully or are intelligently omitted
- [ ] No truncation in middle of code blocks
- [ ] Truncation happens at sentence boundaries
- [ ] Recommendations appear when present in report
- [ ] No Slack webhook errors
- [ ] Message payload < 10 KB (safe margin)
- [ ] User feedback confirms improved readability
