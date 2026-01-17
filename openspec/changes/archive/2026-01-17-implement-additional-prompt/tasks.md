# Tasks: Implement Additional Prompt

## Phase 1: Data Flow

- [ ] Add `AdditionalPrompt string` field to `ConfigMapData` struct (configmap.go)
- [ ] Update `CreateConfigMap()` to include `additional-prompt.md` in ConfigMap data
- [ ] Update `loadIncidentData()` in k8s_executor.go to read `cfg.Agent.AdditionalPrompt`
- [ ] Pass `AdditionalPrompt` to ConfigMapData when building incident data

## Phase 2: Container Mount

- [ ] Add volume mount for `additional-prompt.md` in job.go (similar to base-triage-prompt.md)
- [ ] Mount at `/home/agent/additional-prompt.md`

## Phase 3: Entrypoint Integration

- [ ] Update `build_triage_prompt()` in entrypoint.sh to append additional prompt
- [ ] Add section header: `## Additional Operator Instructions`
- [ ] Only append if file exists and is non-empty

## Phase 4: Audit Trail

- [ ] Verify `prompt-sent.md` captures the complete prompt including additional prompt
- [ ] Update `capture_prompt_sent()` if needed for better formatting

## Testing

- [ ] Add unit test: ConfigMap includes additional-prompt.md when provided
- [ ] Add unit test: ConfigMap omits additional-prompt.md when empty
- [ ] Add unit test: Job mounts additional-prompt.md correctly
- [ ] Update k8s_executor_test.go to verify additional prompt flow
- [ ] Manual test: Run triage with additional prompt, verify it appears in prompt-sent.md

## Documentation

- [ ] Update config.example.yaml comment to note this is now functional
- [ ] Update docs/configuration.md with additional_prompt usage examples
