# Implementation Tasks: Fix HTML Report Formatting Issues

## Overview
This task list covers the implementation of proper HTML formatting for incident investigation reports, focusing on line breaks in metadata sections and overall readability.

## Pre-Implementation Tasks
- [ ] Review example HTML report to identify all formatting issues
- [ ] Download and analyze the source markdown (`investigation.md`) to understand current agent output
- [ ] Verify which sections are affected (metadata header, evidence sections, etc.)
- [ ] Determine if issue is in agent output, markdown conversion, or both

## Implementation Tasks

### Phase 1: Add Unit Tests (Test-First)
- [ ] Create `internal/reporting/markdown_test.go` if it doesn't exist
- [ ] Add test case for metadata section with multiple `**Field**:` patterns
- [ ] Add test case for inline code blocks within metadata
- [ ] Add test case for lists and nested structures
- [ ] Add test case for code blocks and pre-formatted sections
- [ ] Verify tests fail with current implementation (confirms bug exists)

### Phase 2: Implement Markdown Preprocessing
- [ ] Update `ConvertMarkdownToHTML()` to add preprocessing step
- [ ] Implement function to detect metadata blocks (lines starting with `**Field**:`)
- [ ] Add logic to ensure double-space line endings or double newlines for metadata lines
- [ ] Handle edge cases:
  - [ ] Inline code within metadata values
  - [ ] Long values that naturally wrap
  - [ ] Empty or missing fields
- [ ] Ensure preprocessing doesn't break existing markdown features (code blocks, lists, tables)

### Phase 3: Update Agent Prompt (Optional but Recommended)
- [ ] Review current agent prompt in `internal/config/config.go` (line 87)
- [ ] Add instruction about markdown line break syntax
- [ ] Specify format for metadata section: "Use double newlines between metadata fields"
- [ ] Document expected markdown structure in comments

### Phase 4: Add CSS Enhancements (Optional)
- [ ] Review current CSS in `internal/reporting/markdown.go`
- [ ] Add CSS rules for better spacing of paragraph elements
- [ ] Consider adding styles for `.metadata` class if we add structural markup
- [ ] Ensure mobile responsiveness is maintained

### Phase 5: Validation and Testing
- [ ] Run unit tests and confirm all pass
- [ ] Generate test incident report with crashloop scenario
- [ ] Manually inspect HTML output in browser
- [ ] Verify line breaks appear correctly in:
  - [ ] Metadata header section
  - [ ] Evidence sections
  - [ ] Root cause analysis
  - [ ] Recommendations
- [ ] Test with edge cases:
  - [ ] Very long field values
  - [ ] Special characters in values
  - [ ] Missing or null fields
- [ ] Verify markdown source file is still human-readable

### Phase 6: Integration Testing
- [ ] Run full end-to-end test with live agent
- [ ] Generate incident report through full pipeline
- [ ] Upload to Azure storage and verify rendering
- [ ] Check that Slack notifications still link correctly
- [ ] Verify filesystem storage path still works

### Phase 7: Documentation
- [ ] Update code comments in `ConvertMarkdownToHTML()`
- [ ] Document markdown formatting requirements
- [ ] Add example of correct markdown format in code comments
- [ ] Update any relevant README sections about report formatting

## Validation Checklist
- [ ] All unit tests pass
- [ ] HTML reports render with proper line breaks
- [ ] Markdown source remains readable
- [ ] No regressions in existing functionality
- [ ] Code follows project Go style guidelines
- [ ] Changes are backwards compatible with existing reports
- [ ] Performance impact is negligible (< 1ms additional processing)

## Rollback Plan
If issues are discovered after deployment:
1. Revert changes to `ConvertMarkdownToHTML()`
2. Keep agent prompt changes (they won't hurt even if not fully effective)
3. Investigate specific edge cases that failed
4. Apply targeted fixes

## Success Metrics
- HTML reports render with visible line breaks between metadata fields
- No increase in error rates for report generation
- User feedback indicates improved readability
- No new bug reports related to markdown/HTML conversion
