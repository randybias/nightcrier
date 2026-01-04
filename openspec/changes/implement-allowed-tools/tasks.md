# Tasks: Implement Allowed Tools

## Phase 1: Data Flow

- [ ] Add `ALLOWED_TOOLS` environment variable to JobConfig struct (job.go)
- [ ] Pass `ALLOWED_TOOLS` in Job container env vars
- [ ] Update k8s_executor.go to read `cfg.Agent.AllowedTools` and pass to Job

## Phase 2: Claude Integration

- [ ] Update entrypoint.sh to read `ALLOWED_TOOLS` env var
- [ ] Replace hardcoded `--allowedTools` with value from env var
- [ ] Test Claude with restricted tools (e.g., no Write)

## Phase 3: Codex Integration

- [ ] Research Codex tool restriction mechanism (`--allow` flags or config)
- [ ] Add Codex-specific translation in entrypoint.sh
- [ ] Test Codex with restricted tools

## Phase 4: Gemini Integration

- [ ] Research Gemini CLI tool restriction mechanism (check docs, `gemini --help`, config options)
- [ ] Implement Gemini-specific translation if supported
- [ ] If not supported after research: add warning log in entrypoint.sh
- [ ] Test Gemini with restricted tools (if supported)

## Phase 5: Goose Integration

- [ ] Research Goose tool restriction mechanism
- [ ] Add Goose-specific translation in entrypoint.sh
- [ ] Test Goose with restricted tools

## Phase 6: Translation Layer

- [ ] Create `translate_allowed_tools()` function in entrypoint.sh
- [ ] Handle agent-specific format differences
- [ ] Handle empty/unset ALLOWED_TOOLS (use agent defaults)
- [ ] Add validation for unknown tool names

## Testing

- [ ] Unit test: Job includes ALLOWED_TOOLS env var
- [ ] Unit test: Translation produces correct flags per agent
- [ ] Integration test: Claude rejects forbidden tool
- [ ] Integration test: Codex rejects forbidden tool
- [ ] Manual test: Run triage with Write forbidden, verify agent cannot write

## Documentation

- [ ] Update config.example.yaml with per-agent tool format notes
- [ ] Document any agent-specific limitations discovered during research
- [ ] Add security best practices section
