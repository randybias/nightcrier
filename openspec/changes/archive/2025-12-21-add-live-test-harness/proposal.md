# Proposal: Add Live Test Harness

## Problem Statement

Current testing approach is ad-hoc and non-reproducible:
- Manual test runs with inconsistent configurations
- Secrets embedded in config files (security risk)
- No standardized failure injection mechanism
- No automated validation of agent behavior
- Difficult to compare agent outputs across runs
- Testing methodology not documented or reusable

This makes it impossible to reliably validate that triage agents work correctly in production-like conditions or to detect regressions.

## Proposed Solution

Create an automated live test harness under `tests/` that provides:

1. **Template-based configuration**: Config templates without secrets, interpolated at runtime
2. **Secret management**: Single secrets file in `~/dev-secrets/` (not checked in)
3. **Failure injection**: Pluggable scripts to induce specific Kubernetes faults
4. **Test orchestration**: Master script that runs complete test cycles
5. **Log collection**: Structured logging and artifact preservation
6. **Report generation**: Standardized output format for validation

### Key Features

- Uses production `triage-system-prompt.md`
- Starts nightcrier in background with output capture
- Generates test configs from templates (no secrets in repo)
- Induces real Kubernetes failures for testing
- Monitors nightcrier logs for test progress
- Validates agent behavior and produces reports
- Supports both DEBUG and non-DEBUG modes
- Clean separation: secrets in `~/dev-secrets/`, code in `tests/`

## Test Flow

```
1. Load secrets from ~/dev-secrets/
2. Generate test configs from templates
3. Start nightcrier in background
4. Wait for MCP subscription confirmation
5. Induce failure (e.g., crashloopbackoff)
6. Monitor nightcrier logs for agent execution
7. Stop failure induction
8. Collect artifacts and generate report
9. Cleanup nightcrier process
```

## Initial Test Coverage

Starting with one test type:
- **crashloopbackoff**: Deploy a pod that crashes repeatedly

Future tests can be added by creating new `0N_induce_failure_<type>.sh` scripts.

## Directory Structure

```
tests/
├── run-live-test.sh              # Main test orchestration script
├── config-templates/              # Config templates without secrets
│   ├── test-claude.yaml.tmpl
│   ├── test-codex.yaml.tmpl
│   └── test-gemini.yaml.tmpl
├── failure-induction/            # Pluggable failure scripts
│   └── 01_induce_failure_crashloopbackoff.sh
├── lib/                          # Shared utilities
│   ├── config-generator.sh
│   ├── log-monitor.sh
│   └── report-generator.sh
├── logs/                         # Test run logs (gitignored)
│   └── <testid>/                 # One directory per test run
│       ├── nightcrier.log        # Nightcrier output
│       └── report.json           # Test report (JSON format)
├── README.md                     # Usage instructions
└── .gitignore                    # Ignore logs/
```

External dependencies:
```
~/dev-secrets/
├── nightcrier-secrets.env        # All secrets (API keys, Slack, Azure)
└── eastus-cluster1-admin.yaml    # Kubeconfig with admin privileges
```

## Benefits

1. **Reproducibility**: Same test can be run repeatedly with consistent results
2. **Security**: No secrets in repository
3. **Automation**: End-to-end validation without manual intervention
4. **Comparison**: Standardized output enables agent comparison
5. **Regression detection**: Detect when changes break agent behavior
6. **Documentation**: Test methodology is self-documenting through code

## Non-Goals

- Unit testing (already covered by Go tests)
- Load testing or performance benchmarking
- Testing against multiple clusters simultaneously (single cluster per test)
- CI/CD integration (future enhancement)

## Design Decisions

1. **Config location**: Generated configs go in `configs/` (canonical location for review and reuse)
2. **Timeout values**: Variable, returned by induction scripts (complex tests may need longer timeouts)
3. **Test execution**: One test at a time with unique test IDs
4. **Log organization**: Structured by test ID in `tests/logs/<testid>/`
5. **Report format**: Human-readable by default, JSON with `--json` flag
6. **AI agent guidance**: README instructs AI agents to use `--json` for convenience
