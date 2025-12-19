# Change Proposal: Fix HTML Report Formatting Issues

## Why

HTML incident investigation reports are difficult to read due to formatting issues where metadata fields run together without proper line breaks. This significantly degrades the user experience when viewing reports in browsers or Azure Blob Storage. The issue stems from the AI agent generating markdown that doesn't use proper line break syntax, and the markdown-to-HTML conversion preserving these issues in the final output.

This change improves report readability and professionalism by ensuring metadata fields render on separate lines, making critical incident information easier to scan and understand.

## Problem Statement

The HTML version of incident investigation reports has formatting issues that make the reports difficult to read. The primary issue is missing line breaks between metadata fields in the report header section, causing text to run together on the same line. Additional formatting inconsistencies may exist throughout the document.

### Current Behavior

When viewing the generated `investigation.html` file in a browser:
- The incident metadata section (Incident ID, Cluster, Namespace, Resource, Timestamp, Investigator) appears as a continuous paragraph with no line breaks between fields
- Multiple `<strong>` elements appear inline without proper spacing or line breaks
- The markdown-to-HTML conversion is not properly handling line breaks in certain contexts

### Example of Issue

Current HTML output shows:
```
**Incident ID**: 51bc93d9-6c96-4f15-bae0-c633d391a6d7
**Cluster**: kind-events-test
**Namespace**: default
**Resource**: Pod `crashloop-test`
```

Renders as: "**Incident ID**: 51bc... **Cluster**: kind... **Namespace**: default **Resource**: Pod..."

Should render as separate lines:
```
Incident ID: 51bc93d9-6c96-4f15-bae0-c633d391a6d7
Cluster: kind-events-test
Namespace: default
Resource: Pod crashloop-test
```

## Root Cause

The issue originates from the AI agent generating `investigation.md` markdown that doesn't properly format line breaks for HTML rendering. The markdown parser (`gomarkdown/markdown`) requires either:
1. Two trailing spaces at the end of each line for a line break (`<br>`)
2. Double newlines for paragraph breaks (`<p>`)

The agent is likely generating single newlines, which are treated as soft wraps and ignored by the markdown parser.

## Proposed Solution

Update the markdown-to-HTML conversion process to ensure proper line breaks are rendered. There are two potential approaches:

### Approach 1: Fix at Agent Prompt Level (Recommended)
Modify the agent prompt to explicitly instruct the AI to use proper markdown line break syntax when generating reports.

**Pros:**
- Fixes the root cause
- Works for all future reports
- No code changes needed in Go
- Maintains markdown readability

**Cons:**
- Requires agent prompt changes
- Depends on agent following instructions correctly

### Approach 2: Post-Process Markdown Before HTML Conversion
Add a preprocessing step in `ConvertMarkdownToHTML()` to normalize line breaks in specific sections (e.g., metadata blocks with `**Field**:` patterns).

**Pros:**
- Guaranteed to work regardless of agent output
- Can handle edge cases programmatically

**Cons:**
- Adds complexity to conversion logic
- May mask underlying agent output quality issues
- Harder to maintain

### Approach 3: Hybrid (Recommended for Production)
Combine both approaches:
1. Update agent prompt to generate correct markdown (prevents issues)
2. Add lightweight post-processing as safety net (handles edge cases)

## Impact

**User Impact:**
- High: Report readability is significantly degraded without proper formatting
- Affects all users viewing HTML reports in browsers or storage (Azure Blob)

**Technical Impact:**
- Low: Changes are isolated to either agent prompts or markdown conversion function
- No database migrations or API changes
- No breaking changes to existing functionality

## Implementation Plan

See `tasks.md` for detailed implementation steps.

## Testing Strategy

1. **Unit Tests**: Add test cases for `ConvertMarkdownToHTML()` with various markdown inputs
2. **Integration Tests**: Generate a full incident report and verify HTML formatting
3. **Manual Testing**: View generated HTML in browser to confirm visual appearance
4. **Regression Testing**: Ensure existing reports still render correctly

## Alternatives Considered

1. **Switch to different markdown library**: Overkill for this issue; current library is well-maintained
2. **Use HTML templating instead of markdown**: Would require major refactoring; markdown is valuable for source readability
3. **Client-side formatting fixes**: Not feasible when reports are viewed as static HTML files

## Success Criteria

- [ ] HTML reports render with proper line breaks between metadata fields
- [ ] All sections maintain proper spacing and readability
- [ ] Existing markdown features (code blocks, lists, headers) continue to work correctly
- [ ] No regressions in PDF or raw markdown output

## Dependencies

- None (changes are self-contained)

## References

- Example problematic HTML: https://nightcrierincidents.blob.core.windows.net/incident-reports/51bc93d9-6c96-4f15-bae0-c633d391a6d7%2Finvestigation.html
- gomarkdown documentation: https://github.com/gomarkdown/markdown
- CommonMark specification: https://spec.commonmark.org/0.30/
