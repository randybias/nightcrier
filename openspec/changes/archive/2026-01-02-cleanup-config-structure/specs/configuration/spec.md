# Configuration Spec Delta

## REMOVED Requirements

### Requirement: Agent Script Path Configuration

The `agent_script_path` configuration option SHALL be removed as it is obsolete. This was used for Docker-based agent execution, but the K8s executor does not use this field.

#### Scenario: Field no longer exists
- **WHEN** the Config struct is inspected
- **THEN** there SHALL NOT be an `AgentScriptPath` field

---

### Requirement: Agent Docker Image Configuration

The `agent_image` configuration option SHALL be removed as it is obsolete. It is superseded by `k8s.image` for K8s-native execution.

#### Scenario: Field no longer exists
- **WHEN** the Config struct is inspected
- **THEN** there SHALL NOT be an `AgentImage` field

---

### Requirement: Agent Verbose Mode Configuration

The `agent_verbose` configuration option SHALL be removed as it was never implemented. The field was defined but never consumed by any code path.

#### Scenario: Field no longer exists
- **WHEN** the Config struct is inspected
- **THEN** there SHALL NOT be an `AgentVerbose` field

---

### Requirement: Skills Cache Configuration

The `skills` configuration section SHALL be removed as it is obsolete. Skills are now cloned from GitHub at runtime inside the container (`nc-agent-runner/entrypoint.sh`). No local caching mechanism is used or needed.

#### Scenario: SkillsConfig struct no longer exists
- **WHEN** the config package is inspected
- **THEN** there SHALL NOT be a `SkillsConfig` struct

---

## MODIFIED Requirements

### Requirement: Agent Configuration Structure

The agent configuration SHALL use a nested structure under the `agent:` key. This provides consistency with other nested configuration sections like `object_storage`, `state_storage`, and `clusters`.

#### Scenario: Valid nested agent configuration
- **WHEN** a config file contains `agent.cli: "claude"` and `agent.model: "sonnet"`
- **THEN** the configuration SHALL be parsed successfully
- **AND** `cfg.Agent.CLI` SHALL equal "claude"
- **AND** `cfg.Agent.Model` SHALL equal "sonnet"

#### Scenario: Agent CLI validation
- **WHEN** a config contains `agent.cli: "invalid"`
- **THEN** validation SHALL return an error containing "invalid agent.cli"

#### Scenario: Environment variable override
- **WHEN** `AGENT_CLI=codex` is set in the environment
- **AND** a config file contains `agent.cli: "claude"`
- **THEN** `cfg.Agent.CLI` SHALL equal "codex"

---

### Requirement: Kubernetes Executor Configuration Structure

The K8s executor configuration SHALL use a nested structure under the `k8s:` key. All K8s-related configuration options SHALL be grouped together for discoverability.

#### Scenario: Valid nested K8s configuration
- **WHEN** a config file contains `k8s.namespace: "nightcrier"` and `k8s.image_pull_policy: "Never"`
- **THEN** the configuration SHALL be parsed successfully
- **AND** `cfg.K8s.Namespace` SHALL equal "nightcrier"
- **AND** `cfg.K8s.ImagePullPolicy` SHALL equal "Never"

#### Scenario: K8s defaults are applied
- **WHEN** a config file has no `k8s:` section
- **THEN** `cfg.K8s.Namespace` SHALL equal "nightcrier"
- **AND** `cfg.K8s.Image` SHALL equal "nc-agent-runner:latest"
- **AND** `cfg.K8s.ImagePullPolicy` SHALL equal "IfNotPresent"
- **AND** `cfg.K8s.Timeout` SHALL equal 600

#### Scenario: Invalid image pull policy
- **WHEN** a config contains `k8s.image_pull_policy: "Sometimes"`
- **THEN** validation SHALL return an error containing "invalid k8s.image_pull_policy"

---

## ADDED Requirements

### Requirement: No Obsolete Configuration Options

The configuration system SHALL NOT define or process the following obsolete keys: `agent_script_path`, `agent_image`, `agent_verbose`, `skills.cache_dir`, and `skills.disable_triage_preload`.

#### Scenario: Obsolete keys are ignored
- **WHEN** a config file contains `agent_script_path: "./old/path.sh"`
- **THEN** the field SHALL be ignored without error
- **AND** the config struct SHALL NOT have an `AgentScriptPath` field

---

### Requirement: Example Configuration Accuracy

The example configuration file (`configs/config.example.yaml`) SHALL document only fields that are actually used by the codebase. The example SHALL use nested structure for `agent:` and `k8s:` sections and SHALL NOT contain any obsolete or unused configuration options.

#### Scenario: Example config matches implementation
- **WHEN** the example config file is inspected
- **THEN** each documented field SHALL correspond to a field in the Config struct
- **AND** each field SHALL be referenced somewhere in the codebase
