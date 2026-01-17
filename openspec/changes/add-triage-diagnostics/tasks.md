# Tasks: Add Triage Diagnostics

## Phase 1: K8s Pod Log Retrieval

### Task 1.1: Add GetPodLogs method to k8s.Client
- [ ] Add `GetPodLogs(ctx, namespace, podName, containerName string) (string, error)` method
- [ ] Use corev1.PodLogOptions to fetch container logs
- [ ] Handle case where pod doesn't exist (return empty, no error)
- [ ] Add reasonable size limit (e.g., 1MB) to prevent memory issues

**Validation**: Can fetch logs from a running or completed pod

### Task 1.2: Add GetJobPodName method to k8s.Client
- [ ] Add method to find pod name for a given Job
- [ ] Use label selector `job-name={jobName}` to find pod
- [ ] Return most recent pod if multiple exist (Job retries)

**Validation**: Can find pod associated with a triage Job

## Phase 2: Log Upload Integration

### Task 2.1: Add pod log capture to K8s executor processor
- [ ] After Job completion detected, fetch pod logs
- [ ] Generate presigned PUT URL for `{incidentID}/logs/pod-logs.txt`
- [ ] Upload pod logs to object storage
- [ ] Log warning if capture fails (non-fatal)

**Validation**: Pod logs appear in blob storage after Job completion

### Task 2.2: Handle edge cases
- [ ] Pod already deleted (TTL cleanup): Log warning, continue
- [ ] Empty logs: Upload empty file with note
- [ ] Very large logs: Truncate with "[truncated]" marker

**Validation**: No crashes on edge cases, graceful degradation

## Phase 3: Database Integration

### Task 3.1: Add database migration for pod_logs_url
- [ ] Create migration `000006_add_pod_logs_url.up.sql`
- [ ] Add `pod_logs_url` TEXT column to `agent_executions` table
- [ ] Create down migration for rollback

**Validation**: Migration applies cleanly to existing databases

### Task 3.2: Update StateStore interface
- [ ] Add `pod_logs_url` parameter to relevant update methods
- [ ] Or add dedicated `UpdatePodLogsURL(ctx, executionID, url string) error`

**Validation**: Can store and retrieve pod logs URL

### Task 3.3: Implement in postgres and sqlite backends
- [ ] Add method to postgres store
- [ ] Add method to sqlite store

**Validation**: Both backends can store pod logs URL

## Phase 4: Processor Integration

### Task 4.1: Update processor to store pod logs URL
- [ ] After uploading pod logs, call store method to save URL
- [ ] Generate signed GET URL for storage (or store unsigned base URL)

**Validation**: URL is stored in database after successful upload

### Task 4.2: Add logging and metrics
- [ ] Log pod log capture success/failure at info level
- [ ] Include pod log URL in completion log message

**Validation**: Operations team can see pod log capture in nightcrier logs

## Phase 5: Testing

### Task 5.1: Unit tests for new k8s.Client methods
- [ ] Test GetPodLogs with mock clientset
- [ ] Test GetJobPodName with mock clientset
- [ ] Test edge cases (missing pod, empty logs)

**Validation**: Unit tests pass

### Task 5.2: Integration test
- [ ] Run full triage and verify pod logs appear in storage
- [ ] Verify URL is stored in database
- [ ] Test failure scenario (pod deleted before capture)

**Validation**: End-to-end flow works

## Dependencies

- Phase 2 depends on Phase 1
- Phase 4 depends on Phases 2 and 3
- Phase 3 can run in parallel with Phases 1-2
- Phase 5 requires all previous phases
