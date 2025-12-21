# Live Test Harness

A comprehensive testing framework for validating Nightcrier's triage agents against real Kubernetes clusters.

## Overview

The live test harness provides end-to-end validation of the incident triage workflow:

1. **Configuration Generation** - Generate agent configs from templates with secret interpolation
2. **Nightcrier Lifecycle** - Start nightcrier and validate MCP subscription
3. **Failure Induction** - Create specific failure scenarios in the cluster
4. **Log Monitoring** - Track incident detection and agent execution
5. **Artifact Collection** - Gather investigation reports and session logs (DEBUG mode)
6. **Reporting** - Generate test reports with pass/fail status

## Quick Start

```bash
# Run a single agent test
./tests/run-live-test.sh claude crashloopbackoff --debug

# Run all baseline agents in parallel
./tests/run-live-test.sh claude crashloopbackoff --debug &
./tests/run-live-test.sh codex crashloopbackoff --debug &
./tests/run-live-test.sh gemini crashloopbackoff --debug &
wait
```

## Model Configuration

### Baseline Models

The baseline configuration tests the most appropriate model for each CLI tool:

| Agent CLI | Model | Config Template |
|-----------|-------|-----------------|
| claude | sonnet-4.5 | `tests/config-templates/test-claude.yaml.tmpl` |
| codex | gpt-5.2 | `tests/config-templates/test-codex.yaml.tmpl` |
| gemini | gemini-3-flash-preview | `tests/config-templates/test-gemini.yaml.tmpl` |

**Model References**:
- Claude models: https://docs.anthropic.com/en/docs/about-claude/models
- OpenAI/Codex models: https://platform.openai.com/docs/models
- Gemini models: https://ai.google.dev/gemini-api/docs/models

**Note**: Model names must exactly match the API provider's model identifiers. Check the reference links above for valid model names before updating templates.

### Changing Models

Models are configured in the template files at line 25 (`agent_model`). To test different models:

1. **Temporary change** - Edit the template file directly for quick testing
2. **Model comparison** - Create new template files for each model variant

### Model Comparison Testing

To compare different models with the same CLI tool:

```bash
# Example: Compare Codex with different GPT models

# Create variant templates
cp tests/config-templates/test-codex.yaml.tmpl tests/config-templates/test-codex-gpt41.yaml.tmpl
cp tests/config-templates/test-codex.yaml.tmpl tests/config-templates/test-codex-gpt51.yaml.tmpl

# Edit agent_model in each:
# test-codex-gpt41.yaml.tmpl: agent_model: "gpt-4.1"
# test-codex-gpt51.yaml.tmpl: agent_model: "gpt-5.1"
# test-codex.yaml.tmpl: agent_model: "gpt-5.2" (baseline)

# Run comparison tests
./tests/run-live-test.sh codex-gpt41 crashloopbackoff --debug &
./tests/run-live-test.sh codex-gpt51 crashloopbackoff --debug &
./tests/run-live-test.sh codex crashloopbackoff --debug &
wait

# Compare investigation reports in incidents/*/output/investigation.md
```

## Test Scenarios

### crashloopbackoff

Creates a pod that intentionally exits with code 1, triggering CrashLoopBackOff state.

**Failure Script**: `tests/failure-induction/01_induce_failure_crashloopbackoff.sh`

**Expected Behavior**:
- Pod enters CrashLoopBackOff within 30s
- MCP server detects fault event
- Nightcrier triggers agent investigation
- Agent identifies intentional crash as root cause
- Investigation report generated in `incidents/*/output/investigation.md`

**Success Criteria**:
- Agent completes with exit code 0
- Report contains "intentional" or "test pod" in root cause
- High confidence level (90%+)

## Configuration Secrets

Secrets are stored in `~/dev-secrets/nightcrier-secrets.env`:

```bash
# LLM API Keys
ANTHROPIC_API_KEY="sk-ant-..."
OPENAI_API_KEY="sk-proj-..."
GEMINI_API_KEY="..."

# Slack Integration
SLACK_WEBHOOK_URL="https://hooks.slack.com/..."

# Azure Blob Storage
AZURE_STORAGE_ACCOUNT="..."
AZURE_STORAGE_KEY="..."
AZURE_STORAGE_CONTAINER="..."

# MCP Server
MCP_ENDPOINT="http://..."
MCP_API_KEY="..."

# Cluster Configuration
CLUSTER_NAME="eastus-cluster1"
KUBECONFIG_PATH="~/dev-secrets/kubeconfigs/eastus1.kubeconfig"
```

The `config-generator.sh` script uses `envsubst` to interpolate these values into config templates.

## Debug Mode

Debug mode (`--debug`) enables additional logging and session artifact collection:

```bash
./tests/run-live-test.sh claude crashloopbackoff --debug
```

**Debug Features**:
- Verbose nightcrier logging (log_level: debug)
- Agent verbose output (agent_verbose: true)
- Session artifact collection (for supported CLIs):
  - Claude: Session JSONL and command extraction
  - Codex: (future)
  - Gemini: (future)
- Archives stored in `incidents/*/logs/agent-session.tar.gz`

## Parallel Testing

The test harness supports running multiple tests in parallel with unique pod names per test:

```bash
# Run all three baseline agents simultaneously
./tests/run-live-test.sh claude crashloopbackoff --debug &
./tests/run-live-test.sh codex crashloopbackoff --debug &
./tests/run-live-test.sh gemini crashloopbackoff --debug &

# Wait for all to complete
wait

# Analyze results
find incidents -name "investigation.md" -mmin -10 | xargs grep -l "root cause"
```

Each test run gets a unique TEST_ID, which ensures:
- Separate log directories: `tests/logs/test-<timestamp>-<id>/`
- Unique pod names: `test-crashloop-pod-<timestamp>-<id>`
- Isolated nightcrier instances on different ports

## Troubleshooting

### Log Monitoring Timeouts

The log monitoring may report "NO_DETECTION" timeouts even when tests succeed. This is a known cosmetic issue - the log patterns don't match nightcrier's actual output format.

**Verification**: Check if investigation reports were created:
```bash
find incidents -name "investigation.md" -mmin -10
```

### API Key Errors

If agents fail with "API key required" errors:
1. Verify secrets file exists: `~/dev-secrets/nightcrier-secrets.env`
2. Check API keys are exported before starting nightcrier (run-live-test.sh:188-193)
3. Ensure KUBECONFIG is exported before failure induction (run-live-test.sh:277-280)

### Pod Creation Conflicts

If tests fail with "pod already exists":
- Ensure unique TEST_POD_NAME is set (run-live-test.sh:283)
- Clean up stale test pods: `kubectl delete pod -n default -l 'job-name=test-crashloop-pod'`

### Model Not Supported

If you get model-related errors:
- Verify the model name matches the CLI tool's expected format
- Check the CLI tool documentation for supported models
- Update the config template with the correct model name
