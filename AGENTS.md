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

# Repository Guidelines

This repo hosts the Kubernetes MCP alerts event runner. Use these notes to keep contributions consistent while the codebase grows.

## Project Structure & Module Organization
- `cmd/event-runner/`: main entrypoint binary; keep only wiring/configuration here.
- `internal/` or `pkg/`: shared libraries; prefer `internal/` unless APIs are meant to be reused.
- `configs/`: sample config files, Kubernetes manifests, and alert routing templates.
- `deploy/`: Helm chart or kustomize overlays for shipping to a cluster.
- `scripts/`: helper scripts for setup, lint, and CI; keep them POSIX-sh compatible.
- `docs/`: design notes and ADRs; add diagrams in `docs/diagrams/`.
- `test/`: integration or end-to-end fixtures; unit tests live next to code as `*_test.go`.

## Build, Test, and Development Commands
- `go mod tidy` to sync dependencies once modules are added.
- `go run ./cmd/event-runner` to run locally with a kubeconfig in `~/.kube/config`.
- `go test ./...` for unit tests; add `-run` to filter.
- `go vet ./...` and `gofmt -w .` before opening a PR; wrap these in `make lint` and `make fmt` when a Makefile exists.
- `kubectl apply -k deploy/overlays/dev` to exercise manifests against a dev cluster.

## Coding Style & Naming Conventions
- Go: rely on `gofmt`; prefer short, lower snake file names; exported identifiers need doc comments.
- YAML: 2-space indent; keep manifests small and reuse via kustomize patches.
- Config keys lower-kebab (e.g., `alert-source`); env vars upper snake (e.g., `MCP_ENDPOINT`).

## Testing Guidelines
- Table-driven tests for handlers and clients; mock external calls instead of hitting live clusters.
- Name tests `Test<Thing>` and keep fixtures in `testdata/`.
- Aim for meaningful coverage on parsing, filtering, and retry logic; add integration smoke tests under `test/` that can run against `kind`.

## Commit & Pull Request Guidelines
- No history yet—adopt Conventional Commits (`feat:`, `fix:`, `chore:`) and keep PRs under ~300 lines when possible.
- PR description should state intent, how to run validation, and any cluster-impacting changes; link issues if applicable.
- Include screenshots or logs when changing runtime behavior or Kubernetes manifests.

## Security & Configuration Tips
- Do not commit kubeconfigs, tokens, or alert payloads; use `.gitignore` for secrets and `kubectl create secret ... --dry-run=client -o yaml` for templates.
- Prefer env vars plus Kubernetes Secrets over inline config; rotate tokens and note required RBAC in `docs/`.
