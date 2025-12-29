# Nightcrier

An AI-powered Kubernetes incident triage system that reacts to investigates well understood faults using AI agents.  It is
intended primarily as a proof-of-concept of an operations-centric usage of the Model Context Protocol (MCP) along with agents.
MCP, has interesting parts of it's protocol that are underutilized and this codebase serves to showcase what can be done and
why that matters for AIOps (AI and Agents for IT Operatons).

## MCP & Agents for Operators vs MCP & Agents for Developers

MCP, currently, while flawed, is iterating quickly, and certain capabilities aren't seeing the light of day.  Especially capabilities
valuable for AIOps.  For a software developer, MCP servers simply act as convenient tool-calling mechanism for the AI-assistants.
Workflows are largely driven by the developers themselves.  Contrast this to operations or SRE teams that are responsible for
production systems, where much the time they are reacting to events in production.

How would we trigger AI agents to proactively react to production events?

## Overview and Purpose

This project is an attempt to show what that might look like.  It works with a modified [Kubernetes MCP Server](https://github.com/randybias/kubernetes-mcp-server-with-events/tree/feature/events-streaming)
that supports sending Kubernetes events through the MCP subscription mechanism.  Nightcrier subscribes to specific kinds of events called
faults and on receipt of a fault from a monitored MCP server kicks off an automated triage process.  A general purpose agent (e.g. Claude, Codex, Gemini, Goose)
leverages a combination of [Agent Skills](https://agentskills.io) and MCP servers in order to complete an initial triage process that can
then be forwarded to an operations team to act upon.  Initially, nightcrier launches agents that leverage the k8s-troubleshooter agent skill that is part of
the [k8s4agents](https://github.com/randybias/k8s4agents) skillset (just one skill for now).

The initial triage reports are meant to do the following:

- Provide an initial assessment and root cause analysis
- Provide a "proof of work" with all of the actions taken by the agent and their outputs
- Suggest potential remediation steps
- Operator only in "read-only" mode to avoid product change risk

This saves the operations team valuable initial data collection efforts, shortening time to resolution if all goes well.  If this system even just
saved 50% of data collection time on 50% of production faults, that would be a huge win.

## AIOps Philosophy, Architecture Philosophy, Agents, and Tokens, Tokens, Tokens

At Mirantis, we once did an experiment and set k8sgpt loose on a cluster to manage it.  The result, as you might guess, was tokens, tokens, tokens -> $$$$$.
Operations isn't something that can be run in a tight loop with agents running everything all the time.  Instead, you need more of an event-based system that
is designed to look for state changes, drift, faults, or other changes that need to be responded to.  You also need components that do predictive analysis and
anomaly detection.  Anomaly detection is it's own special thing simply because it means understanding what any give system's "steady state" looks like.  Long term
it would be nice to see something like Nightcrier, as part of a group of semi-autonomous agents acting in different roles.  You could see something like the following:

- Anomaly Detection Agentic Overlord
- Predictive Analysis System
- Upgrade and Rollout/back System
- Triage Swarm

Where anomaly detection, predictive analysis, planned upgrades, and real time faults could trigger different types of triage agents to perform automated correlation,
root cause analysis, and prescribe corrective measures, all to be reviewed the operations and SRE teams.  With some feedback loops and ongoing learning, the system
could learn to run itself most of the time.

Nightcrier only handles one part of this picture and is mostly a (working) proof-of-concept right now.  It's an event based system that subscribes to high
quality (hopefully) production faults (usually well known ones for now) and performs the initial triage, notifying the operations team via Slack and pointing them
to the triage reports.

I've tested this enough to say that if you just listen for faults (more about this later) in Kubernetes, and use a cheap model like Haiku for the triage, you get pretty good mileage.

### What Nightcrier Does

Nightcrier subscribes to **high-quality fault events only** from many kubernetes-mcp-servers and treats each fault as a serious incident requiring investigation:

1. **One fault subscription per MCP server** - Each nightcrier instance maintains a single connection to a modified Kubernetes MCP server and subscribes to "faults" that have been pre-defined/filtered (i.e. not every event is sent)
2. **Treats faults as incidents** - Every fault event is a serious incident requiring AI-powered investigation
3. **Launches AI triage agents** - Spawns Kubernetes Jobs running AI agents (Claude, Codex, Gemini, or Goose) for full root cause analysis
4. **Reports to operations teams** - Delivers investigation reports via Slack with storage links
5. **End-to-end incident tracking** - Tracks each incident from detection through investigation to reporting

### What Nightcrier is NOT

- **NOT a general event subscriber** - Would be overwhelmed by raw Kubernetes events at scale
- **NOT responsible for event filtering** - Signal-to-noise filtering happens upstream in kubernetes-mcp-server (distributing the work and minimizing token usage)
- **NOT a high-volume event processing system** - Designed for pre-filtered, high-signal faults only (but should handle a reasonable load)

### Design Decisions

**Event Types:**
- **General Events**: Raw Kubernetes events (handled by kubernetes-mcp-server)
- **Fault Events**: Pre-filtered, high-signal events indicating problems (from kubernetes-mcp-server to Nightcrier)
- **Incidents**: Fault events under active AI investigation (Nightcrier's domain)

**Filtering Philosophy:**
- Signal-to-noise filtering happens in kubernetes-mcp-server, NOT in Nightcrier
- kubernetes-mcp-server uses sophisticated logic to identify true faults worth investigating
- This design prevents Nightcrier from being overwhelmed at scale
- Ensures AI agents only investigate genuine incidents, not noise

## Architecture

### High-Level Flow

```
kubernetes-mcp-server -> Filtered MCP Events (faults) -> Nightcrier -> K8s Job (AI Agent) -> Investigation Report
                                                             |
                                                             v
                                                     Object Storage (Azure/S3)
                                                             |
                                                             v
                                                 Slack Notification (with Report URL)
```

### Agent Execution Model

Nightcrier uses **Kubernetes-native stateless agent execution**:

1. **ConfigMap Creation** - Incident data (incident.json, permissions.json, system prompt) is stored in a ConfigMap
2. **Job Creation** - A Kubernetes Job is created with the `nc-agent-runner` container image
3. **Presigned URLs** - PUT URLs are generated for artifact upload (report.md, agent.log, session.tar.gz)
4. **Agent Execution** - The Job runs an AI agent (Claude, Codex, Gemini, or Goose) to investigate
5. **Artifact Upload** - Results are uploaded directly to Object Storage from within the container
6. **Cleanup** - ConfigMap is deleted after Job completion

See [Architecture](docs/architecture.md) for detailed execution flow diagrams and circuit breaker logic.

## Features

- **Automated fault detection** via MCP server integration
- **AI-powered root cause analysis** using multiple agents (Claude, Codex, Gemini, Goose)
- **Kubernetes-native execution** via Jobs (stateless, no persistent volumes)
- **Multi-backend storage** support (Azure Blob Storage, AWS S3, S3-compatible, filesystem)
- **Slack notifications** with investigation reports and signed URLs
- **Secure artifact storage** with presigned URL generation
- **Circuit breaker** for agent failure handling
- **Intelligent validation** to prevent spurious notifications
- **System health monitoring** with degraded/recovered alerts
- **Multi-cluster support** with independent triage configuration per cluster (serialized as on triage agent active at a time per monitored cluster)

## Quick Start

### Prerequisites

- Go 1.23 or later
- Kubernetes cluster (for triage agent execution via Jobs)
- kubectl and kind (for local development)
- Object storage: Azure Blob Storage, AWS S3, or S3-compatible (RustFS/MiniO tested)
- Slack webhook (optional, for notifications)
- At least one LLM API key (Anthropic, OpenAI, or Google)

### Build

```bash
# Clone the repository
git clone https://github.com/rbias/nightcrier.git
cd nightcrier

# Build the Nightcrier binary
make build

# Build and load the agent container image (for kind)
cd nc-agent-runner
make build
make load-kind
```

### Configure

```bash
# Copy example configuration
cp configs/config.example.yaml configs/config.yaml

# Edit with your settings (MCP endpoints, storage, API keys)
$EDITOR configs/config.yaml
```

See [Configuration Guide](docs/configuration.md) for detailed options.

### Run

```bash
./bin/nightcrier --config configs/config.yaml
```

## Documentation

| Document | Description |
|----------|-------------|
| [Configuration](docs/configuration.md) | All configuration options, multi-cluster setup, environment variables |
| [Installation](docs/installation.md) | Kubeconfig setup, RBAC permissions, ServiceAccount creation |
| [Architecture](docs/architecture.md) | Detailed validation flows, circuit breaker logic |
| [Storage](docs/storage.md) | Azure Blob, S3, MinIO setup and credentials |
| [Usage](docs/usage.md) | Running Nightcrier, local development, testing |
| [Troubleshooting](docs/troubleshooting.md) | Common issues, debugging, log analysis |
| [Developer Setup](docs/dev_setup.md) | Local development environment with kind |
| [Contributing](docs/contributing.md) | Contribution guidelines |

## Output Structure

### Object Storage

Artifacts are uploaded directly from the agent container:

```
<bucket>/<incident-id>/
  results/
    report.md               # AI-generated investigation report
    investigation.html      # HTML-formatted report
    agent.log               # Agent execution logs
    session.tar.gz          # Agent session archive
    result.json             # Execution metadata
  incident.json             # Original incident data
```

### Local Workspace

```
./workspaces/<incident-id>/
  incident.json             # Incident metadata
  permissions.json          # Cluster permissions
  prompt-sent.md            # Full prompt sent to agent
```

## Slack Notifications

When configured, notifications include:
- Incident metadata (cluster, namespace, resource)
- Root cause analysis with confidence level
- Investigation duration
- **"View Report" button** linking to the signed URL

## Related Projects

- [kubernetes-mcp-server](https://github.com/rbias/kubernetes-mcp-server) - MCP server for Kubernetes fault events
- [Model Context Protocol](https://github.com/anthropics/mcp) - Protocol specification

## Contributing

See [Contributing Guide](docs/contributing.md) for development workflow and contribution guidelines.

## License

See repo (Apache 2.0 License)