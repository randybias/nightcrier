## ADDED Requirements

### Requirement: Test Matrix Definition

The test harness SHALL support declarative test matrix definitions that specify combinations of clusters, failures, and agents to test.

#### Scenario: Parse valid matrix configuration

- **GIVEN** a YAML file defining clusters, failures, and agents
- **WHEN** the orchestrator loads the configuration
- **THEN** it SHALL expand the matrix into individual test cases
- **AND** validate all referenced failures and agents exist

#### Scenario: Matrix with execution parameters

- **GIVEN** a matrix configuration with execution settings (parallelism, timeouts)
- **WHEN** the orchestrator runs the matrix
- **THEN** it SHALL respect the configured parallel_clusters limit
- **AND** enforce timeout_per_test for each test case

#### Scenario: Invalid matrix configuration

- **GIVEN** a matrix configuration referencing a non-existent failure type
- **WHEN** the orchestrator loads the configuration
- **THEN** it SHALL exit with error code 1
- **AND** report which failure type was not found

### Requirement: Cluster Provisioner Interface

The test harness SHALL define an abstract interface for cluster lifecycle management to support multiple provisioning backends.

#### Scenario: Provision cluster via interface

- **GIVEN** a configured ClusterProvisioner implementation
- **AND** a ClusterSpec defining the desired cluster configuration
- **WHEN** Provision is called
- **THEN** it SHALL return ClusterInfo with kubeconfig and endpoint data
- **OR** return an error if provisioning fails

#### Scenario: Teardown cluster via interface

- **GIVEN** a provisioned cluster with a known cluster ID
- **WHEN** Teardown is called with that ID
- **THEN** the cluster resources SHALL be released
- **AND** the method SHALL return success or error

#### Scenario: Stub provisioner for development

- **GIVEN** the stub provisioner is configured
- **AND** a pre-existing cluster kubeconfig path is provided
- **WHEN** Provision is called
- **THEN** it SHALL return ClusterInfo pointing to the existing cluster
- **AND** Teardown SHALL be a no-op

### Requirement: Failure Library

The test harness SHALL maintain an extensible library of inducible failures with metadata describing each failure type.

#### Scenario: Discover available failures

- **GIVEN** failures defined in `tests/failures/` directory
- **WHEN** the harness scans for available failures
- **THEN** it SHALL return a list of failure IDs
- **AND** each failure SHALL have associated metadata

#### Scenario: Failure metadata structure

- **GIVEN** a failure directory with `metadata.yaml`
- **WHEN** the metadata is loaded
- **THEN** it SHALL include name, description, expected_duration
- **AND** optionally include severity and prerequisites

#### Scenario: Induce failure via script

- **GIVEN** a failure with `induce.sh` script
- **WHEN** `induce.sh start` is executed
- **THEN** the failure condition SHALL be created in the target cluster
- **AND** the script SHALL output `TIMEOUT=<seconds>`

#### Scenario: Clean up failure via script

- **GIVEN** an active failure induced by `induce.sh start`
- **WHEN** `induce.sh stop` is executed
- **THEN** all resources created by the failure SHALL be removed
- **AND** the cluster SHALL return to a clean state

### Requirement: Test Orchestration

The test harness SHALL coordinate test execution across the matrix, managing cluster lifecycle and result collection.

#### Scenario: Execute test matrix sequentially

- **GIVEN** a test matrix with 3 clusters x 2 failures x 2 agents
- **AND** parallel execution is disabled
- **WHEN** the orchestrator runs the matrix
- **THEN** it SHALL execute all 12 test cases sequentially
- **AND** report progress after each test completes

#### Scenario: Execute test matrix in parallel

- **GIVEN** a test matrix with parallel_clusters: 2
- **WHEN** the orchestrator runs the matrix
- **THEN** it SHALL run tests on up to 2 clusters concurrently
- **AND** aggregate results as tests complete

#### Scenario: Handle test failure without aborting matrix

- **GIVEN** a test matrix with multiple test cases
- **AND** one test case fails
- **WHEN** the orchestrator continues execution
- **THEN** remaining test cases SHALL still execute
- **AND** the final report SHALL include the failed test

#### Scenario: Cleanup on interruption

- **GIVEN** a test matrix execution in progress
- **WHEN** the orchestrator receives SIGINT or SIGTERM
- **THEN** it SHALL stop inducing new failures
- **AND** clean up any active failures
- **AND** teardown any provisioned clusters
- **AND** generate a partial results report

### Requirement: Result Storage

The test harness SHALL persist test results to disk in a structured format for analysis and comparison.

#### Scenario: Create result directory per run

- **GIVEN** a new test matrix execution
- **WHEN** the run starts
- **THEN** a directory SHALL be created with format `YYYY-MM-DD-<matrix-name>-<run-id>/`
- **AND** the frozen matrix configuration SHALL be saved

#### Scenario: Store individual test results

- **GIVEN** a completed test case
- **WHEN** results are persisted
- **THEN** a JSON file SHALL be created with test metadata
- **AND** the file SHALL include status (pass/fail), duration, and artifact paths

#### Scenario: Query results by run ID

- **GIVEN** a completed test run with known run ID
- **WHEN** results are queried
- **THEN** all test case results SHALL be returned
- **AND** aggregate statistics (pass count, fail count) SHALL be available

### Requirement: Test Reporting

The test harness SHALL generate human-readable reports summarizing test run outcomes.

#### Scenario: Generate summary report

- **GIVEN** a completed test matrix run
- **WHEN** the summary report is generated
- **THEN** it SHALL display total tests, passed, failed counts
- **AND** list failed tests with brief error descriptions

#### Scenario: Generate comparison report

- **GIVEN** test results for the same failure across multiple agents
- **WHEN** a comparison report is requested
- **THEN** it SHALL show side-by-side results per agent
- **AND** highlight differences in outcome or performance

#### Scenario: Tail logs during execution

- **GIVEN** a test matrix execution in progress
- **AND** the `--follow` flag is set
- **WHEN** tests are running
- **THEN** log output SHALL stream to the terminal in real-time
- **AND** indicate which test/cluster each log line belongs to

### Requirement: Backward Compatibility

The test harness SHALL preserve the existing live test workflow while adding new capabilities.

#### Scenario: Run existing live test script

- **GIVEN** the existing `tests/run-live-test.sh` script
- **WHEN** executed directly (not via orchestrator)
- **THEN** it SHALL function identically to before this change
- **AND** produce the same output format

#### Scenario: Orchestrator wraps existing script

- **GIVEN** the orchestrator executing a single test case
- **WHEN** no custom executor is specified
- **THEN** it SHALL invoke `run-live-test.sh` as the execution backend
- **AND** parse its output for result extraction
