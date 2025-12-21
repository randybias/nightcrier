# Implementation Tasks: Refactor Agent Prompt Integration

## Phase 1: Configuration Changes

**Goal**: Remove mandatory agent_prompt, add optional additional_agent_prompt.

### 1.1 Update Config Structure
- [x] Rename `AgentPrompt` field to `AdditionalAgentPrompt` in `internal/config/config.go`
- [x] Update mapstructure tag from `agent_prompt` to `additional_agent_prompt`
- [x] Remove `agent_prompt` from required fields validation
- [x] Add comment documenting optional nature and use case

### 1.2 Update Configuration Files
- [x] Remove `agent_prompt` from `configs/config.example.yaml`
- [x] Add commented `additional_agent_prompt` example with documentation
- [x] Remove `agent_prompt` from `configs/config-test.yaml`
- [x] Remove `agent_prompt` from `configs/config-multicluster.yaml`
- [x] Remove `agent_prompt` from `configs/config-codex.yaml`

### 1.3 Update Executor
- [x] Update `ExecutorConfig.Prompt` field name to `AdditionalPrompt`
- [x] Update executor to handle empty additional prompt gracefully
- [x] **FIX**: Changed prompt passing from `-p` flag (which doesn't exist) to positional argument
- [x] **FIX**: Combined system prompt content + additional prompt in executor.go
- [x] **FIX**: Removed `--system-prompt-file` arg since content is now inlined into combined prompt

### 1.4 Verification
- [x] Run `go build ./...` to verify compilation
- [x] Verify application starts without agent_prompt configured
- [x] Verify additional_agent_prompt is passed when provided

---

## Phase 2: System Prompt Rewrite

**Goal**: Create minimal, skill-aware system prompt.

### 2.1 Rewrite System Prompt
- [x] Backup current `configs/triage-system-prompt.md`
- [x] Write new minimal system prompt (~20 lines)
- [x] Include workspace file descriptions (incident.json, incident_cluster_permissions.json)
- [x] Include k8s-troubleshooter skill invocation guidance
- [x] Include read-only constraint
- [x] Include output format specification

### 2.2 Update Documentation
- [x] Update agent-container README to reference new system prompt approach
- [x] Document skill-driven investigation flow

### 2.3 Verification
- [x] Test agent invocation with new system prompt
- [x] Verify agent successfully invokes k8s-troubleshooter skill
- [x] Verify investigation output is generated correctly

---

## Phase 3: Prompt Capture

**Goal**: Capture full prompt before execution for auditability.

### 3.1 Implement Prompt Capture in Executor
- [x] Add function to read system prompt file content
- [x] Add function to combine system prompt + additional prompt
- [x] Add function to generate prompt metadata (timestamp, incident ID, cluster, model)
- [x] Add function to write prompt-sent.md to workspace
- [x] Call prompt capture before subprocess launch in Execute()

### 3.2 Define Prompt File Format
- [x] Use markdown format with metadata header
- [x] Include: timestamp, incident ID, cluster name, agent CLI, model
- [x] Include: full system prompt content
- [x] Include: additional prompt content (or "None provided")

### 3.3 Verification
- [x] Verify prompt-sent.md is created in workspace before agent runs
- [x] Verify prompt file contains correct metadata
- [x] Verify prompt file contains full system prompt
- [x] Verify prompt file contains additional prompt when provided

---

## Phase 4: Storage Integration

**Goal**: Upload captured prompt to Azure and local storage.

### 4.1 Update Storage Interface
- [x] Add `PromptSent []byte` field to `IncidentArtifacts` struct in `internal/storage/storage.go`

### 4.2 Update Artifact Reading
- [x] Update `readIncidentArtifacts()` in `cmd/nightcrier/main.go` to read prompt-sent.md
- [x] Handle missing file gracefully (optional artifact)

### 4.3 Update Azure Storage
- [x] Add prompt-sent.md to artifact upload mapping in `internal/storage/azure.go`
- [x] Add file description for index.html generation
- [x] Add to ordered files list for consistent display

### 4.4 Update Filesystem Storage
- [x] Add prompt-sent.md writing to `internal/storage/filesystem.go`
- [x] Add to artifact URLs map

### 4.5 Verification
- [x] Verify prompt-sent.md appears in Azure Blob Storage
- [x] Verify prompt file appears in index.html with SAS URL
- [x] Verify prompt file is stored in local filesystem incidents directory
- [x] Run full incident with Azure storage enabled, verify all artifacts present

---

## Phase 5: Spec Updates

**Goal**: Update OpenSpec specifications to reflect new behavior.

### 5.1 Update Configuration Spec
- [x] Modify "Missing agent prompt" scenario to remove requirement
- [x] Add scenario for optional additional_agent_prompt
- [x] Update "No duplicate defaults" scenarios

### 5.2 Create Prompt Capture Spec
- [x] Document prompt-sent.md artifact requirement (in cloud-storage spec)
- [x] Document metadata format
- [x] Document capture timing (before subprocess)

### 5.3 Update Storage Spec
- [x] Add prompt-sent.md to artifact list
- [x] Document upload behavior

---

## Phase 6: Agent Commands Extraction (Added)

**Goal**: Extract executed commands from Claude session for debugging.

### 6.1 Implement Command Extraction
- [x] Add `extract_agent_commands()` function in run-agent.sh
- [x] Parse session JSONL files using jq to extract Bash tool calls
- [x] Write commands to `logs/agent-commands-executed.log`
- [x] Include header with timestamp, incident ID, session ID
- [x] Format commands with `$ ` prefix and description comments

### 6.2 Update Storage for Commands Log
- [x] Add `CommandsExecuted []byte` field to `AgentLogs` struct
- [x] Add file reading in main.go
- [x] Add Azure upload handling
- [x] Add filesystem storage handling
- [x] Add to index.html file descriptions

### 6.3 Update Specs
- [x] Add "Agent Commands Extraction" requirement to agent-logging spec
- [x] Add "Debug Log Artifacts" requirement to cloud-storage spec

---

## Verification Checklist

After all phases complete:

- [x] Application starts without any agent_prompt configuration
- [x] Application accepts optional additional_agent_prompt
- [x] System prompt is <25 lines and skill-focused
- [x] Agent successfully invokes k8s-troubleshooter skill
- [x] prompt-sent.md is created in workspace before agent execution
- [x] prompt-sent.md contains full metadata and prompt content
- [x] prompt-sent.md is uploaded to Azure Blob Storage
- [x] prompt-sent.md appears in incident index.html
- [x] Investigation completes successfully with new prompt structure
- [x] No regressions in existing functionality
- [x] agent-commands-executed.log generated in DEBUG mode
- [x] Full E2E test with Azure storage (verified by user)
