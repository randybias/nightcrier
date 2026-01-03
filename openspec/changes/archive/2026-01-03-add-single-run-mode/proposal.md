# Change: Add Single Run Mode for Test Harnesses

## Why

Test harnesses need a "one and done" execution mode where nightcrier processes exactly one fault event and then exits cleanly. This enables deterministic end-to-end testing without manual intervention to stop the process.

## What Changes

- Add `--single-run` CLI flag to nightcrier
- When enabled, nightcrier processes the first fault event it receives (launching agent, waiting for completion, finalizing reporting)
- After the first event is fully processed, nightcrier exits with code 0
- Subsequent fault events are dropped (shutdown initiated before they can be processed)

## Impact

- Affected specs: `configuration` (adding new CLI flag)
- Affected code: `cmd/nightcrier/main.go` only (~10-15 lines)
- No config file changes (CLI-only flag is appropriate for test tooling)
- No breaking changes

## Implementation Notes

Uses existing shutdown mechanism (`cancel()` triggers `ctx.Done()` which exits the event loop gracefully). This is the minimal approach requiring no architectural changes.
