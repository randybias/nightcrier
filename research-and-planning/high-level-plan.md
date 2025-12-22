# Nightcrier – High-Level Plan

## Vision and Value Proposition

Nightcrier is an MCP client that listens for events on a number of MCP servers simultaneously.  It waits for events to occur in Kubernetes clusters and then takes action based on those events, usually in the form of spawning an AI agent in a sandbox, which evaluates and takes action on the event the runner received.

## Context and Positioning
- Prototype / proof-of-concept: first milestone is to prove the value of AI-led triage on real Kubernetes faults; future expansion is optional.
- Based on a locally customized kubernetes-mcp-server (`../kubernetes-mcp-server`), not the upstream release; event shapes and behaviors follow the modified build.
- Primary platform scope: Kubernetes clusters only (no multi-plane or non-K8s sources in this phase).
- **Read-Only Triage:** The agent is strictly limited to read-only analysis and triage. It must NOT take proactive actions or modify cluster state.

## Target Audience and Personas
- Ops on-call: needs fast, consistent triage summaries and artifacts to decide next actions.
- SRE lead / platform owner: wants signal-to-noise control, rate limits per cluster, and auditable agent actions.
- AI agent maintainer: cares about prompt patterns, skill loading (k8s-troubleshooter), and sandbox boundaries.
- Security/Compliance observer (lightweight for now): wants to know where logs and artifacts land and how access is controlled.

## Tech Stack Alignment (reuse kubernetes-mcp-server patterns)
- Language/tooling: Go modules, `go test ./...`, `go vet ./...`, `gofmt`; Makefile-style wrappers if needed (mirror kubernetes-mcp-server practices).
- Event intake: SSE transport from customized kubernetes-mcp-server (future-ready for pluggable transport/bus); HTTP client patterns consistent with the server codebase.
- Agent orchestration: Pluggable CLI wrapper (Headless mode). Invokes `claude` (or generic equivalent like `codex`, `goose`) via a minimalist setup script and environment variables. No direct SDK dependencies in the core binary to allow for flexible agent switching.
- Config: env vars and CLI flags consistent with kubernetes-mcp-server style (kebab keys for config files, upper-snake for env); sample configs under `configs/`.
- Packaging/runtime: single binary under `cmd/event-runner/`, shared libs in `internal/`; logs via the same logging approach used in kubernetes-mcp-server (leveled, JSON/text as appropriate).
- Integrations: Slack webhook (Text only) with incident ID and disk path; filesystem drop for reports/artifacts.
- MCP libraries: core runtime should stick to `github.com/modelcontextprotocol/go-sdk v1.1.0` to align with the customized server.

## Objectives and Outcomes
- Detect high-severity Kubernetes faults from kubernetes-mcp-server and trigger per-incident agent sessions.
- Perform automated triage and root-cause exploration using the k8s-troubleshooter skill and kubernetes-mcp-server queries.
- Deliver concise, actionable reports with artifacts to disk and to a Slack webhook for the ops team to validate and act on.
- Keep each incident isolated (unique workspace) for auditability and reproducibility.

## Scope and Constraints
- In scope: Kubernetes faults only; per-cluster rate limiting; disk and Slack delivery; Headless agent sandbox with k8s-troubleshooter.
- Out of scope for now: complex dedupe (handled upstream or basic suppression only), ticketing system integration, dependency drift management, heavy prompt safety.
- **Strict Read-Only:** Agents run without write credentials; all cluster data arrives via kubernetes-mcp-server push or follow-up read-only queries.
- **Single Concurrency:** Only one active triage agent per cluster at a time to prevent collisions.
- Internal environment; minimal prompt safety is acceptable initially.

## Assumptions
- kubernetes-mcp-server fault watcher already enriches events with scoped logs.
- The target agent (e.g., Claude Code) supports a headless/scripted mode that can be invoked via CLI with a setup script.
- k8s-troubleshooter skill from ../k8s4agents is compatible with the target clusters.
- Slack webhook can be provisioned for the ops channel; disk location for reports is writable and monitored by ops.

## System Overview
1) kubernetes-mcp-server monitors clusters and emits fault events via SSE with log enrichment.
2) Event runner subscribes per cluster, applies severity filters, and enforces per-cluster rate limits (1 active agent max) and Global Circuit Breaker.
3) On eligible fault, if queue is open, runner creates a unique incident workspace.
4) Runner spawns the agent via CLI (headless mode) with a structured prompt, context bundle, and startup script.
5) Agent uses k8s-troubleshooter skill to perform triage, collect signals, and produce findings (Read-Only).
6) Runner assembles a final report, packages artifacts, writes to the incident directory, and posts a summary with disk path to Slack via webhook.

## Components and Responsibilities
- Event runner core: manages SSE intake, severity gating, Global Circuit Breaker (max total agents), and per-cluster locking (mutex/queue).
- Agent orchestrator: Generic CLI wrapper. Prepares the sandbox (files, env vars), runs the agent command (e.g., `claude --headless -p "..."`), and captures exit status/artifacts.
- Context builder: normalizes event payload (cluster, namespace, involved resources, timestamps, log excerpts).
- Reporting and delivery: standard markdown report, artifact bundle, disk drop, Slack webhook notification.
- Observability: metrics for event intake, queue depth, active agents, success/failure counts.

## Rate Limiting and Resilience
- **Per-Cluster:** Strict 1-at-a-time concurrency. If busy, queue the event (up to a small backlog limit).
- **Global Circuit Breaker:** Max N total active agents across all clusters. If hit, drop or queue based on severity.
- **Deduplication:** Simple suppression: if an incident is currently *Running* for the same Resource+Namespace, ignore new alerts for it.
- **Backlog:** Events in queue expire if not processed within X minutes.

## Agent Prompt and Workflow
- Inputs: fault summary, cluster/namespace, resource identifiers, timestamps, enriched logs.
- Prompt goals: **READ ONLY ANALYSIS.** Collect clarifying context, produce triage checklist, hypotheses, and remediation suggestions. NO proactive actions.
- Tools: k8s-troubleshooter skill (Read-Only), kubernetes-mcp-server queries.
- Execution guardrails: max runtime per incident, step limits.

## Reporting and Artifacts
- Report (markdown): incident id, cluster/namespace, detection time, fault summary, key observations, likely root causes, recommended actions.
- Artifacts: log excerpts, command outputs, agent scratchpad/notes, and a final summary file.
- Disk layout: root incidents directory with per-incident subfolders.
- Slack webhook: short summary (incident id, cluster, severity, headline finding) plus path to disk location.

## Security and Privacy (initial, light)
- Internal-only trust model.
- Keep write credentials out of the agent.
- Agent is invoked with specific flags/scripts to ensure strictly read-only tools are loaded.

## Phased Roadmap (pre-OpenSpec)
- Phase 0: Confirm ops drop location, Slack webhook target, severity definitions.
- Phase 1: Event runner skeleton with SSE subscription, severity gating, Global Circuit Breaker, Single-Concurrency Queue.
- Phase 2: Agent orchestration (Headless CLI wrapper, startup script generation, workspace setup).
- Phase 3: Reporting pipeline (markdown template, artifact packaging), disk persistence, and Slack webhook delivery.
- Phase 4: Resilience polish (timeout handling, queue tuning), basic metrics/logging.
- Phase 5: Optimization (Slack `files.upload` bot token, richer prompts, additional agent backends).

## Open Questions and Next Research
- Confirm the exact arguments/env vars for the chosen agent's headless mode.
- Define the "Setup Script" interface (e.g., `setup.sh` that installs skills vs. pre-configured environment).
- Determine queue depth limits before dropping events.