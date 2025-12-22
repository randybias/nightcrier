# Fix Gemini Agent

## Problem Statement

The Gemini triage agent has Gemini-specific issues that need investigation and resolution:

1. **Untested in final runs**: Gemini was skipped in final testing, indicating unresolved issues
2. **Unvalidated command extraction**: Gemini post-hook uses different session format (logs.json vs JSONL), extraction needs testing
3. **Incomplete session artifact collection**: Gemini session structure (~/.gemini) differs from Claude/Codex, extraction may not work

Current status: Gemini agent configuration exists but has not been validated end-to-end.

## Proposed Solution

### 1. Diagnostic Test Run

Execute diagnostic test to identify Gemini-specific issues:

1. Run Gemini agent against crashloopbackoff test with DEBUG logging
2. Capture all Gemini-specific error messages and warnings
3. Verify Gemini session artifact extraction works
4. Document all observed issues specific to Gemini

### 2. Session Artifact Extraction

Fix and validate Gemini-specific post-hook (gemini-post.sh):

1. Test session file extraction from container (~/.gemini directory)
2. Verify logs.json format matches jq parsing in gemini-post.sh
3. Confirm command extraction produces valid output
4. Test session archive creation
5. Add extraction error handling specific to Gemini format

### 3. Gemini-Specific Fixes

Address issues discovered in diagnostic run that are specific to Gemini:

1. Fix any Gemini CLI invocation issues in gemini.sh
2. Correct tool name mapping for Gemini (if broken)
3. Fix session path or format issues in gemini-post.sh
4. Address any Gemini-specific container environment issues

## Success Criteria

1. **Execution**: Gemini agent successfully completes crashloopbackoff test
2. **Artifacts**: Session extraction produces logs and command history
3. **Documentation**: Gemini-specific issues documented with fixes applied

## Scope

**In Scope:**
- Diagnostic testing of Gemini agent
- Fixes to gemini.sh runner script (Gemini-specific)
- Fixes to gemini-post.sh session extraction (Gemini-specific)
- Documentation of Gemini-specific issues and solutions

**Out of Scope:**
- Changes to Gemini CLI tool itself
- Model validation (applies to all agents)
- API quota detection (applies to all agents)
- Timeout handling (applies to all agents)
- Performance comparison with other agents
- Reliability testing (10+ consecutive runs)
- Generic error handling improvements

## Dependencies

- Requires valid GEMINI_API_KEY with available quota
- Requires baseline test harness (already exists)
- Requires agent-container spec (already exists)
- May require alternative Gemini model if quota remains exhausted

## Risks and Mitigation

| Risk | Impact | Mitigation |
|------|--------|------------|
| API quota exhausted | Cannot test Gemini | Document issue, defer to user to resolve quota |
| Gemini CLI bugs | Cannot fix externally | Document issues, note limitations |
| Session format unexpected | Extraction fails | Update gemini-post.sh jq filters to match actual format |

## Open Questions

1. **What are the specific Gemini failure symptoms?** Need diagnostic run to identify
2. **Does ~/.gemini session directory structure match expectations?** Verify in container
3. **Does logs.json format match gemini-post.sh parsing?** Test jq filters against real file
