<!-- OPENSPEC:START -->
# OpenSpec Instructions

These instructions are for AI assistants working in this project.

Always open `@/openspec/AGENTS.md` when the request:
- Mentions planning or proposals (words like proposal, spec, change, plan)
- Introduces new capabilities, breaking changes, architecture shifts, or big performance/security work
- Sounds ambiguous and you need the authoritative spec before coding

Use `@/openspec/AGENTS.md` to learn:
- How to create and apply change proposals
- Spec format and conventions
- Project structure and guidelines

Keep this managed block so 'openspec update' can refresh the instructions.

<!-- OPENSPEC:END -->

# Project Instructions

These instructions apply to all AI assistants (Claude Code, Codex, Gemini, etc.). Both `CLAUDE.md` and `AGENTS.md` are becoming standard filenames for agent instructions across tools.

## File Organization

### Working Files and Temporary Output

**NEVER create working files in the repository root.** This includes:
- Implementation summaries (e.g., `IMPLEMENTATION_SUMMARY.md`)
- Testing summaries (e.g., `TESTING_SUMMARY.md`)
- Progress reports, notes, or scratch work
- Any temporary or intermediate output

**Use `scratch/` for all working files.** The `scratch/` directory is gitignored and exists for this purpose.

If documentation has lasting value, it belongs in:
- `research-and-planning/` - Research notes, planning docs, proposals not yet in OpenSpec
- The appropriate README (e.g., `tests/README.md`, `agent-container/README.md`)
- OpenSpec change directories (only: `proposal.md`, `tasks.md`, `design.md`, `specs/*/spec.md`)

### OpenSpec Archive Cleanup

When archiving an OpenSpec change (`openspec archive <change-id>`):

1. **Remove working files** - Delete any summaries, notes, or temporary files created during implementation from the repo root
2. **Verify documentation** - Ensure relevant docs are in the proper location (READMEs, not loose files)
3. **Check for stale references** - Remove references to worktrees or temporary paths that no longer exist

The OpenSpec structure only allows these files in a change directory:
- `proposal.md` - Why and what
- `tasks.md` - Implementation checklist
- `design.md` - Technical decisions (optional)
- `specs/[capability]/spec.md` - Requirements and scenarios

No other files (summaries, reports, etc.) belong in OpenSpec directories.