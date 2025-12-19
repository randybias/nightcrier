# Design Document: Improve Slack Notification Formatting

## Context

Slack notifications are sent via incoming webhooks when incident investigations complete. The notifications use Slack Block Kit format to display structured information. Currently, the notifications have formatting issues that make them difficult to read and often truncate critical information unhelpfully.

The notification flow:
```
Incident completes → Extract summary from report → Build Slack message → Send webhook
```

## Current Implementation Analysis

### Metadata Formatting Issue

**Current code** (`slack.go:114-119`):
```go
Fields: []SlackText{
    {Type: "mrkdwn", Text: fmt.Sprintf("*Cluster:*\n%s", summary.Cluster)},
    {Type: "mrkdwn", Text: fmt.Sprintf("*Namespace:*\n%s", summary.Namespace)},
    {Type: "mrkdwn", Text: fmt.Sprintf("*Resource:*\n%s", summary.Resource)},
    {Type: "mrkdwn", Text: fmt.Sprintf("*Reason:*\n%s", summary.Reason)},
},
```

**Problem**: The `\n` creates a line break between label and value, causing each field to span 2 lines. In a 2-column fields layout, this creates 4 rows when it should only need 2.

**Visual result**:
```
┌─────────────────────┬─────────────────────┐
│ Cluster:            │ Namespace:          │
│ kind-events-test    │ default             │
├─────────────────────┼─────────────────────┤
│ Resource:           │ Reason:             │
│ Pod/crash-test-pod  │ CrashLoop           │
└─────────────────────┴─────────────────────┘
```

**Desired result**:
```
┌─────────────────────────┬────────────────────────┐
│ Cluster: kind-events-   │ Namespace: default     │
│ test                    │                        │
├─────────────────────────┼────────────────────────┤
│ Resource: Pod/crash-    │ Reason: CrashLoop      │
│ test-pod                │                        │
└─────────────────────────┴────────────────────────┘
```

### Root Cause Extraction Issue

**Current code** (`slack.go:369-383`):
```go
// Capture root cause content (first substantive paragraph)
if inRootCause && strings.TrimSpace(line) != "" &&
   !strings.HasPrefix(line, "#") &&
   !strings.HasPrefix(line, "**Confidence") {
    rootCauseLines = append(rootCauseLines, strings.TrimSpace(line))
    if len(rootCauseLines) >= 2 {
        break // Just get first couple lines
    }
}

// ...

if len(rootCause) > 300 {
    rootCause = rootCause[:297] + "..."
}
```

**Problems**:
1. Only extracts first 2 lines of root cause section
2. 300-character hard limit is too restrictive
3. Truncates at character count, not sentence boundary
4. No special handling for code blocks
5. Truncation can happen mid-code-block, mid-word, or mid-markdown

**Example problematic output**:
```
Root Cause (HIGH confidence):
The container command is explicitly designed to crash: ```bash
```

This shows the beginning of a code block but not the actual code, rendering the information useless.

## Technical Constraints

### Slack Block Kit Limits

From Slack documentation:
- **Section block text**: Maximum 3,000 characters
- **Markdown text block**: Maximum 12,000 characters
- **Total message**: Maximum 40,000 characters
- **Blocks per message**: Maximum 50 blocks
- **Webhook payload**: Maximum 16 KB total (includes JSON)

### Practical Limits for Readability

Based on user experience:
- **Optimal root cause length**: 500-1,500 characters (3-6 sentences)
- **Maximum before "too long"**: 2,000 characters
- **Recommendations**: 300-500 characters (2-3 bullet points)

## Proposed Solution Design

### Component 1: Inline Metadata Formatting

**Simple fix** - remove `\n` from format string:

```go
Fields: []SlackText{
    {Type: "mrkdwn", Text: fmt.Sprintf("*Cluster:* %s", summary.Cluster)},
    {Type: "mrkdwn", Text: fmt.Sprintf("*Namespace:* %s", summary.Namespace)},
    {Type: "mrkdwn", Text: fmt.Sprintf("*Resource:* %s", summary.Resource)},
    {Type: "mrkdwn", Text: fmt.Sprintf("*Reason:* %s", summary.Reason)},
},
```

**Impact**: Immediate improvement in readability, reduces vertical space by 50%.

### Component 2: Enhanced Root Cause Extraction

Create new function with sophisticated markdown parsing:

```go
func ExtractRootCauseEnhanced(content []byte) (string, error) {
    // 1. Parse markdown to find "## Root Cause" section
    // 2. Extract all content until next ## heading
    // 3. Process content with smart truncation
    // 4. Return up to 2,000 characters
}
```

#### Algorithm: Smart Truncation

```
1. Extract full root cause section (all paragraphs, lists, code blocks)
2. If total length <= 2,000 chars → return as-is
3. If total length > 2,000 chars:
   a. Check if there are code blocks
   b. If code block in first 1,500 chars:
      - Include everything up to and including the code block
      - Add truncation marker after
   c. If code block after 1,500 chars:
      - Truncate before code block
      - Add marker: "... [code omitted, see full report]"
   d. If no code blocks:
      - Find last complete sentence before 1,950 chars
      - Truncate there and add "... [see full report]"
4. Return truncated content
```

#### Sentence Boundary Detection

```go
func findLastSentenceBoundary(text string, maxLen int) int {
    if len(text) <= maxLen {
        return len(text)
    }

    // Look for sentence endings: ". ", ".\n", "? ", "?\\n", "! ", "!\n"
    searchText := text[:maxLen]

    // Try each ending pattern in order of preference
    endings := []string{". ", ".\n", "? ", "?\n", "! ", "!\n"}
    for _, ending := range endings {
        if idx := strings.LastIndex(searchText, ending); idx > 0 {
            return idx + len(ending)
        }
    }

    // Fallback: find last space
    if idx := strings.LastIndex(searchText, " "); idx > maxLen/2 {
        return idx
    }

    // Ultimate fallback: hard truncate (should rarely happen)
    return maxLen
}
```

#### Code Block Detection

```go
type CodeBlock struct {
    Start    int
    End      int
    Language string
    Content  string
}

func findCodeBlocks(text string) []CodeBlock {
    var blocks []CodeBlock

    // Find fenced code blocks (```language...```)
    fencePattern := regexp.MustCompile("(?s)```(\\w*)\\n(.*?)```")
    matches := fencePattern.FindAllStringSubmatchIndex(text, -1)

    for _, match := range matches {
        blocks = append(blocks, CodeBlock{
            Start:    match[0],
            End:      match[1],
            Language: text[match[2]:match[3]],
            Content:  text[match[4]:match[5]],
        })
    }

    return blocks
}
```

### Component 3: Recommendations Extraction

Add new extraction function:

```go
func ExtractRecommendations(content []byte) []string {
    // Parse sections looking for:
    // - "## Recommendations"
    // - "## Next Steps"
    // - "## Suggested Actions"
    // - "## Remediation"

    // Extract bullet points from that section
    // Return up to 3 recommendations
}
```

Format in Slack as:

```go
{
    Type: "section",
    Text: &SlackText{
        Type: "mrkdwn",
        Text: fmt.Sprintf("*Recommended Actions:*\n%s",
            formatRecommendations(summary.Recommendations)),
    },
}

func formatRecommendations(recs []string) string {
    var parts []string
    for _, rec := range recs {
        // Clean and truncate each recommendation
        clean := strings.TrimSpace(rec)
        if len(clean) > 150 {
            clean = clean[:147] + "..."
        }
        parts = append(parts, fmt.Sprintf("• %s", clean))
    }
    return strings.Join(parts, "\n")
}
```

### Component 4: Updated Data Flow

```
investigation.md
    ↓
ExtractRootCauseEnhanced() → root cause (0-2000 chars, smart truncation)
    ↓
ExtractConfidenceLevel() → confidence (HIGH/MEDIUM/LOW)
    ↓
ExtractRecommendations() → []string (up to 3 items)
    ↓
IncidentSummary struct (with new fields)
    ↓
BuildSlackMessage() → SlackMessage with blocks
    ↓
Send webhook
```

## Implementation Strategy

### Phase 1: Quick Win - Metadata Formatting
- **Effort**: 5 minutes
- **Risk**: Very low
- **Impact**: High (immediate visual improvement)
- **Dependencies**: None

Change the format string from `*Label:*\n%s` to `*Label:* %s`.

### Phase 2: Enhanced Extraction - Core Logic
- **Effort**: 2-3 hours
- **Risk**: Medium (parsing logic complexity)
- **Impact**: High (better information quality)
- **Dependencies**: None

Implement new extraction function with smart truncation.

### Phase 3: Recommendations
- **Effort**: 1-2 hours
- **Risk**: Low
- **Impact**: Medium (nice to have)
- **Dependencies**: Phase 2

Add recommendations extraction and display.

### Phase 4: Testing & Polish
- **Effort**: 2-3 hours
- **Risk**: Low
- **Impact**: High (ensures quality)
- **Dependencies**: Phases 1-3

Comprehensive testing with real-world reports.

## Edge Cases and Handling

| Edge Case | Handling Strategy |
|-----------|-------------------|
| No root cause section | Return "See investigation report for details" |
| Root cause is very short (< 100 chars) | Return as-is, don't pad |
| Root cause has multiple code blocks | Include first block if it fits, truncate after |
| Code block is 1,000+ lines | Truncate before code block with marker |
| Root cause has tables | Treat as regular markdown (may not render well in Slack) |
| Root cause has images | Ignore image references |
| Confidence level not found | Return "UNKNOWN" |
| No recommendations section | Skip recommendations block entirely |
| Very long recommendations | Truncate each to 150 chars |
| Malformed markdown | Best-effort parsing, fallback to simple extraction |

## Performance Considerations

### Extraction Performance

**Expected**:
- Typical report size: 5-20 KB
- Extraction time: < 50ms
- Memory overhead: 2x report size (temporary buffers)

**Optimization techniques**:
- Use `bytes.Buffer` for string building
- Compile regex patterns once at init
- Avoid unnecessary string copies
- Early return when limits are reached

### Webhook Performance

**Expected**:
- Payload size: 2-4 KB (typical)
- Webhook latency: 100-500ms (network + Slack processing)
- Retry budget: 3 attempts with exponential backoff

**No significant change** from current implementation.

## Testing Strategy

### Unit Tests

1. **Metadata formatting**:
   - Verify inline format produces expected output
   - Test with long values (truncation by Slack)
   - Test with special characters

2. **Root cause extraction**:
   - Short content (< 500 chars) → full extraction
   - Medium content (500-2000 chars) → full extraction
   - Long content (> 2000 chars) → smart truncation
   - Content with code blocks → special handling
   - Content with lists → preserve formatting
   - Malformed markdown → graceful degradation

3. **Sentence boundary detection**:
   - Text ending with period
   - Text ending with question mark
   - Text ending with exclamation
   - Text with no sentence endings → space boundary
   - Text with no spaces → hard truncate

4. **Code block detection**:
   - Fenced code blocks (```)
   - Multiple code blocks
   - Nested code-like patterns
   - Code blocks with language specifier

5. **Recommendations extraction**:
   - Standard recommendations section
   - Alternative section names
   - No recommendations → empty list
   - Very long recommendations → truncation

### Integration Tests

1. Generate full incident report with agent
2. Extract summary with new functions
3. Build Slack message
4. Verify JSON payload is valid
5. Check payload size < 10 KB
6. Verify all sections present

### Manual Testing

1. Send test notifications to Slack
2. Verify appearance on desktop
3. Verify appearance on mobile
4. Test with various incident types:
   - CrashLoopBackOff
   - ImagePullBackOff
   - OOMKilled
   - Config errors
5. Verify code blocks render correctly
6. Verify links are clickable
7. Verify truncation markers appear correctly

## Backward Compatibility

**Breaking changes**: None

**Format changes**: Yes (user-visible)
- Metadata displays differently (better)
- Root cause may be longer (better)
- Recommendations may appear (new feature)

**Rollback**: Keep old extraction function available for emergency rollback

## Security Considerations

- **Input validation**: Markdown content could contain malicious patterns
- **Sanitization**: Slack handles markdown sanitization, but we should:
  - Limit total message size
  - Prevent infinite loops in parsing
  - Handle malformed input gracefully
- **Information disclosure**: Ensure sensitive data in reports doesn't exceed what should be in Slack

## Future Enhancements

Potential improvements for later:

1. **Interactive buttons**: Add buttons like "Acknowledge", "Mute", "Escalate"
   - Requires Slack App (not available with webhooks)

2. **Message threading**: Put recommendations in thread
   - Requires Slack App

3. **Customizable formatting**: Allow users to configure what sections appear
   - Add configuration options

4. **Rich previews**: Unfurl report URLs with preview
   - Requires Slack App or external service

5. **Executive summary**: Extract and display 1-sentence summary
   - Requires agent to generate consistent summaries

These are NOT in scope for this change.

## References

- Slack Block Kit: https://api.slack.com/block-kit/building
- Slack text formatting: https://api.slack.com/reference/surfaces/formatting
- Slack message limits: https://api.slack.com/changelog/2018-04-truncating-really-long-messages
- Current implementation: `internal/reporting/slack.go`
