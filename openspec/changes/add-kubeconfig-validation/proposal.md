# Change: Add kubeconfig content validation at startup

## Why

Currently, nightcrier only checks that the kubeconfig file exists (`os.Stat`) but doesn't validate its contents. This leads to confusing runtime failures when:
- The file contains invalid YAML
- The file is missing required fields (clusters, users, contexts)
- The specified context doesn't exist in the file

These errors surface later during triage agent startup, making them harder to diagnose.

## What Changes

- Parse kubeconfig as YAML at startup to detect malformed files
- Validate required top-level fields are present (clusters, users, contexts)
- Validate current-context references an existing context entry
- Fail fast with clear error messages

**Not in scope:**
- Server connectivity checks (would require network access)
- Certificate/credential validity (would require parsing certs)
- Kubernetes version compatibility checks

## Impact

- Affected specs: `configuration` (add kubeconfig validation requirement)
- Affected code: `internal/cluster/config.go` (extend `Validate()` method)
