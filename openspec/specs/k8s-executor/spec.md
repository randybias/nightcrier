# k8s-executor Specification

## Purpose
TBD - created by archiving change refactor-k8s-native-agent. Update Purpose after archive.
## Requirements
### Requirement: Job Creation
The K8s executor SHALL create Kubernetes Jobs to run agent containers.

#### Scenario: Job creation
- **GIVEN** an incident requires triage
- **WHEN** the K8s executor is invoked
- **THEN** it SHALL create a K8s Job in the configured namespace
- **AND** the Job SHALL be named with pattern `triage-{incident-id}`
- **AND** the Job SHALL have labels for app, incident-id, and cluster

#### Scenario: Job configuration
- **GIVEN** a Job is being created
- **WHEN** the Job spec is generated
- **THEN** `ttlSecondsAfterFinished` SHALL be set to 3600
- **AND** `activeDeadlineSeconds` SHALL be set from config with default 600
- **AND** `backoffLimit` SHALL be 0
- **AND** `restartPolicy` SHALL be Never

#### Scenario: Container configuration
- **GIVEN** a Job is being created
- **WHEN** the container spec is generated
- **THEN** the image SHALL be configurable with default nightcrier-agent:latest
- **AND** resource limits SHALL be configurable with default 2Gi memory and 1 CPU
- **AND** resource requests SHALL be configurable with default 512Mi memory

### Requirement: ConfigMap Generation
The K8s executor SHALL create ConfigMaps for incident data.

#### Scenario: ConfigMap creation
- **GIVEN** an incident requires triage
- **WHEN** the K8s executor prepares resources
- **THEN** it SHALL create a ConfigMap named with pattern `incident-{incident-id}`
- **AND** the ConfigMap SHALL contain incident.json with fault data
- **AND** the ConfigMap SHALL contain permissions.json with cluster permissions
- **AND** the ConfigMap SHALL contain system-prompt.md with triage instructions

#### Scenario: ConfigMap labels
- **GIVEN** a ConfigMap is being created
- **WHEN** the ConfigMap metadata is set
- **THEN** it SHALL have labels for app and incident-id
- **AND** labels SHALL enable cleanup queries

#### Scenario: ConfigMap cleanup
- **GIVEN** a Job has completed
- **WHEN** the orchestrator processes results
- **THEN** the associated ConfigMap SHALL be deleted
- **AND** cleanup SHALL occur regardless of Job success or failure

### Requirement: Secret References
The K8s executor SHALL reference pre-provisioned Secrets for credentials.

#### Scenario: Kubeconfig secret
- **GIVEN** a Job is being created
- **WHEN** the volume spec is generated
- **THEN** it SHALL reference Secret with pattern `triage-kubeconfig-{cluster-name}`
- **AND** the secret SHALL be mounted at `/home/agent/.kube/config`
- **AND** the mount SHALL be read-only

#### Scenario: API key secret
- **GIVEN** a Job is being created
- **WHEN** the environment spec is generated
- **THEN** it SHALL reference Secret `ai-api-keys` for API keys
- **AND** keys SHALL be injected as environment variables
- **AND** missing keys SHALL be optional

### Requirement: Presigned URL Generation
The K8s executor SHALL generate presigned PUT URLs for output uploads.

#### Scenario: URL generation
- **GIVEN** an incident requires triage
- **WHEN** the K8s executor prepares resources
- **THEN** it SHALL generate presigned PUT URLs for report, log, session, result, and commands
- **AND** URLs SHALL have expiration matching job timeout plus buffer

#### Scenario: URL injection
- **GIVEN** presigned URLs are generated
- **WHEN** the Job environment is configured
- **THEN** URLs SHALL be passed as environment variables
- **AND** variable names SHALL be OUTPUT_URL_REPORT, OUTPUT_URL_LOG, OUTPUT_URL_SESSION, OUTPUT_URL_RESULT, OUTPUT_URL_COMMANDS

#### Scenario: Storage path structure
- **GIVEN** presigned URLs are generated
- **WHEN** the storage paths are determined
- **THEN** outputs SHALL be stored at `incidents/{incident-id}/results/`
- **AND** each file SHALL have a distinct path

### Requirement: Job Lifecycle Management
The K8s executor SHALL manage the complete Job lifecycle.

#### Scenario: Job monitoring
- **GIVEN** a Job has been created
- **WHEN** the executor monitors completion
- **THEN** it SHALL use K8s watch API to observe Job status
- **AND** it SHALL detect completion as Succeeded or Failed
- **AND** it SHALL have a maximum wait timeout

#### Scenario: Successful completion
- **GIVEN** a Job completes successfully
- **WHEN** the executor processes results
- **THEN** it SHALL verify result.json exists in Object Store
- **AND** it SHALL verify report.md exists in Object Store
- **AND** it SHALL download and process the report
- **AND** it SHALL download commands-executed.log from Object Store
- **AND** it SHALL clean up the ConfigMap

#### Scenario: Failed completion
- **GIVEN** a Job fails with non-zero exit or timeout
- **WHEN** the executor processes results
- **THEN** it SHALL attempt to retrieve result.json for exit code
- **AND** it SHALL attempt to retrieve agent.log for debugging
- **AND** it SHALL attempt to retrieve commands-executed.log if available
- **AND** it SHALL mark the incident as failed with reason
- **AND** it SHALL clean up the ConfigMap

#### Scenario: Orphan cleanup
- **GIVEN** the executor starts
- **WHEN** it initializes
- **THEN** it SHALL query for orphaned resources
- **AND** it SHALL delete ConfigMaps older than 24 hours with matching labels

### Requirement: Namespace Configuration
The K8s executor SHALL operate within a configurable namespace.

#### Scenario: Namespace setting
- **GIVEN** the executor is configured
- **WHEN** creating K8s resources
- **THEN** all resources SHALL be created in the configured namespace
- **AND** the default namespace SHALL be nightcrier

#### Scenario: RBAC requirements
- **GIVEN** the executor operates in a namespace
- **WHEN** the service account is configured
- **THEN** it SHALL have permissions to create and delete Jobs and ConfigMaps and get Secrets
- **AND** it SHALL NOT have cluster-wide permissions

### Requirement: Error Handling
The K8s executor SHALL handle K8s API errors gracefully.

#### Scenario: API unavailable
- **GIVEN** the K8s API is unavailable
- **WHEN** the executor attempts to create resources
- **THEN** it SHALL retry with exponential backoff
- **AND** it SHALL fail after configurable max retries with default 3
- **AND** it SHALL return a descriptive error

#### Scenario: Resource conflict
- **GIVEN** a ConfigMap or Job already exists with the same name
- **WHEN** the executor attempts to create resources
- **THEN** it SHALL delete the existing resource first if orphaned
- **OR** it SHALL fail with incident already being processed error

#### Scenario: Quota exceeded
- **GIVEN** namespace resource quotas are exceeded
- **WHEN** the executor attempts to create a Job
- **THEN** it SHALL fail with a descriptive error
- **AND** the error SHALL indicate which quota was exceeded

### Requirement: Configuration
The K8s executor SHALL be configurable via Nightcrier config.

#### Scenario: Configuration options
- **GIVEN** the Nightcrier config file
- **WHEN** configuring the K8s executor
- **THEN** k8s.namespace SHALL be available with default nightcrier
- **AND** k8s.image SHALL be available with default nightcrier-agent:latest
- **AND** k8s.timeout SHALL be available with default 600
- **AND** k8s.memory_limit SHALL be available with default 2Gi
- **AND** k8s.cpu_limit SHALL be available with default 1
- **AND** k8s.cleanup_ttl SHALL be available with default 3600

#### Scenario: In-cluster configuration
- **GIVEN** Nightcrier runs inside a K8s cluster
- **WHEN** the executor initializes
- **THEN** it SHALL auto-detect in-cluster config
- **AND** it SHALL use the pod service account

#### Scenario: Out-of-cluster configuration
- **GIVEN** Nightcrier runs outside a K8s cluster
- **WHEN** the executor initializes
- **THEN** it SHALL use kubeconfig from ~/.kube/config or KUBECONFIG env var
- **AND** it SHALL use the current context

### Requirement: Artifact Processing
The K8s executor SHALL process and persist all agent artifacts.

#### Scenario: Report retrieval and conversion
- **GIVEN** a Job completes successfully
- **WHEN** the executor retrieves results
- **THEN** it SHALL download report.md from Object Store
- **AND** it SHALL convert markdown to HTML using existing ConvertMarkdownToHTML
- **AND** it SHALL upload investigation.html to Object Store

#### Scenario: Database recording
- **GIVEN** artifacts have been retrieved and processed
- **WHEN** the executor records results
- **THEN** it SHALL call RecordTriageReport with markdown and HTML content
- **AND** it SHALL call RecordAgentExecution with artifact URLs
- **AND** it SHALL call CompleteIncident with exit code and status

#### Scenario: Artifact URL persistence
- **GIVEN** artifacts are stored in Object Store
- **WHEN** the executor records results
- **THEN** it SHALL store both canonical and signed URLs
- **AND** URLs SHALL be accessible for Slack notifications and dashboard
- **AND** canonical URLs SHALL allow re-signing for later access

