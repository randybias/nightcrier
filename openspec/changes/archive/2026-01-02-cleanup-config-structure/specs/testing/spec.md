# Testing Spec Delta

## ADDED Requirements

### Requirement: Nested Config Structure Unit Tests

The configuration package SHALL include unit tests that verify the new nested `AgentConfig` and `K8sConfig` structs work correctly.

#### Scenario: AgentConfig validation tests
- **WHEN** `AgentConfig.Validate()` is called with valid fields
- **THEN** it SHALL return nil
- **AND** when called with invalid CLI value
- **THEN** it SHALL return an error containing "agent.cli"

#### Scenario: K8sConfig validation tests
- **WHEN** `K8sConfig.Validate()` is called with valid fields
- **THEN** it SHALL return nil
- **AND** when called with invalid ImagePullPolicy
- **THEN** it SHALL return an error containing "k8s.image_pull_policy"

#### Scenario: K8sConfig defaults tests
- **WHEN** `K8sConfig.ApplyDefaults()` is called on an empty struct
- **THEN** all fields SHALL have their documented default values
- **AND** `Namespace` SHALL equal "nightcrier"
- **AND** `Image` SHALL equal "nc-agent-runner:latest"

---

### Requirement: Nested Config Loading Tests

The configuration package SHALL include tests that verify nested YAML and environment variable loading works correctly.

#### Scenario: YAML nested structure loading
- **WHEN** a YAML file contains nested `agent:` and `k8s:` sections
- **THEN** the config loader SHALL parse them into the corresponding struct fields
- **AND** `cfg.Agent.CLI` SHALL contain the value from `agent.cli`
- **AND** `cfg.K8s.Namespace` SHALL contain the value from `k8s.namespace`

#### Scenario: Environment variable override
- **WHEN** `AGENT_CLI` environment variable is set
- **AND** a YAML file contains `agent.cli` with a different value
- **THEN** the environment variable SHALL take precedence
- **AND** `cfg.Agent.CLI` SHALL equal the environment variable value

---

### Requirement: Test Baseline Verification

Before making any changes, the test suite baseline SHALL be established and verified after changes.

#### Scenario: Pre-change baseline
- **WHEN** `go test ./...` is run before any changes
- **THEN** all tests SHALL pass
- **AND** the test count SHALL be recorded

#### Scenario: Post-change verification
- **WHEN** `go test ./...` is run after all changes
- **THEN** all tests SHALL pass
- **AND** the test count SHALL be equal to or greater than the baseline

---

### Requirement: Integration Config Validation

Config files SHALL be tested to ensure they load correctly with the new structure.

#### Scenario: Example config loads
- **WHEN** `configs/config.example.yaml` is loaded
- **THEN** it SHALL parse without errors
- **AND** all documented fields SHALL be accessible

#### Scenario: Production configs load
- **WHEN** `configs/config-weu.yaml` is loaded
- **THEN** it SHALL parse without errors
- **AND** `cfg.Agent.CLI` SHALL be populated
- **AND** `cfg.K8s.Namespace` SHALL be populated

---

### Requirement: No Regression in Existing Tests

All existing tests that were passing before the change SHALL continue to pass after updating their fixtures.

#### Scenario: Config loading tests pass
- **WHEN** the config loading tests are run
- **THEN** `TestLoadWithAllRequiredFields` SHALL pass
- **AND** `TestLoadFromEnvVars` SHALL pass
- **AND** `TestLoadFromConfigFile` SHALL pass
- **AND** `TestEnvVarsOverrideConfigFile` SHALL pass

#### Scenario: Validation tests pass
- **WHEN** the validation tests are run
- **THEN** all `TestValidation_*` tests SHALL pass with updated fixtures
