# Tasks: Fix Gemini Agent

## Phase 1: Diagnostic Investigation

- [ ] 1.1 Run Gemini diagnostic test
  - Execute: `./tests/run-live-test.sh gemini crashloopbackoff --debug`
  - Capture full nightcrier.log output
  - Capture agent container stdout/stderr
  - Note all Gemini-specific error messages and failure points

- [ ] 1.2 Analyze Gemini agent execution
  - Check if agent starts successfully
  - Verify API authentication works
  - Identify any Gemini-specific tool execution failures
  - Document Gemini-specific behavior issues

- [ ] 1.3 Verify session artifact extraction
  - Check if ~/.gemini directory exists in container
  - Verify logs.json file is created
  - Test manual command extraction with jq
  - Review gemini-post.sh execution logs
  - Confirm agent-commands-executed.log is generated

- [ ] 1.4 Document Gemini-specific findings
  - Create issue list specific to Gemini with symptoms and reproduction steps
  - Identify root causes vs symptoms for Gemini issues
  - Prioritize Gemini-specific fixes by impact
  - Flag any Gemini-specific blocking issues (CLI bugs, etc.)

## Phase 2: Session Extraction Fixes

- [ ] 2.1 Fix session file extraction
  - Verify container session path: `/home/agent/.gemini`
  - Test docker cp extraction manually
  - Update gemini-post.sh if path incorrect
  - Add error handling for missing session directory

- [ ] 2.2 Fix command extraction from logs.json
  - Verify logs.json structure matches jq expectations
  - Test jq filter against actual session file
  - Update filter if format differs
  - Handle empty or malformed logs.json gracefully

- [ ] 2.3 Validate extraction completeness
  - Compare extracted commands vs actual agent actions
  - Verify all bash commands captured
  - Check for missing tool calls (read, write, grep, etc.)
  - Fix missing tool types in extraction logic

- [ ] 2.4 Test session archiving
  - Verify agent-session.tar.gz created
  - Extract and inspect archive contents
  - Confirm all relevant files included
  - Validate archive integrity

## Phase 3: Gemini-Specific Runner Fixes

- [ ] 3.1 Fix tool mapping in gemini.sh
  - Review map_tools_to_gemini() function
  - Verify all Claude tools have correct Gemini equivalents
  - Test tool mapping with actual Gemini CLI
  - Fix any Gemini-specific mapping issues

- [ ] 3.2 Fix Gemini CLI invocation
  - Verify command string format matches Gemini CLI expectations
  - Test prompt escaping with Gemini
  - Fix any Gemini-specific CLI argument issues
  - Validate model parameter format

- [ ] 3.3 Test Gemini container environment
  - Verify Gemini CLI available in container
  - Check Gemini-specific environment variables
  - Test Gemini session directory creation
  - Fix any Gemini-specific path issues

## Phase 4: Validation Testing

- [ ] 4.1 Run fixed Gemini test
  - Execute: `./tests/run-live-test.sh gemini crashloopbackoff --debug`
  - Verify agent completes successfully
  - Confirm session artifacts extracted
  - Check investigation output quality

- [ ] 4.2 Verify all artifacts collected
  - Confirm agent-session.tar.gz created
  - Verify agent-commands-executed.log populated
  - Check triage_gemini_*.log in logs/ directory
  - Validate all expected files present

- [ ] 4.3 Document Gemini-specific behavior
  - Note any Gemini-specific quirks or limitations
  - Document differences from Claude/Codex behavior
  - Add Gemini-specific troubleshooting notes
  - Update Gemini configuration examples

## Validation Criteria

Each task is complete when:
- Gemini-specific changes implemented and tested
- No regressions in Gemini functionality
- Gemini-specific documentation updated
- Manual verification confirms expected Gemini behavior

## Dependencies

- Phase 2-3 depend on Phase 1 diagnostic completion
- Phase 4 validation requires Phases 2-3 fixes complete

## Parallelizable Work

- Phase 2 (session extraction) and Phase 3 (runner fixes) can run in parallel after Phase 1 completes
- Documentation can be drafted while Phase 4 testing runs
