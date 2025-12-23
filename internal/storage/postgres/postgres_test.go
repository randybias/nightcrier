package postgres

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rbias/nightcrier/internal/events"
	"github.com/rbias/nightcrier/internal/incident"
	"github.com/rbias/nightcrier/internal/storage"
)

// getTestConnectionString returns the PostgreSQL connection string for testing.
// Tests are skipped if the environment variable is not set or connection fails.
func getTestConnectionString(t *testing.T) string {
	// Check for test database connection string
	connStr := os.Getenv("NIGHTCRIER_TEST_POSTGRES_URL")
	if connStr == "" {
		t.Skip("NIGHTCRIER_TEST_POSTGRES_URL environment variable not set, skipping PostgreSQL integration tests")
	}
	return connStr
}

// setupTestStore creates a test store and runs migrations.
func setupTestStore(t *testing.T, ctx context.Context) *Store {
	t.Helper()

	connStr := getTestConnectionString(t)

	cfg := &Config{
		ConnectionString: connStr,
		MaxOpenConns:     5,
		MaxIdleConns:     2,
	}

	store, err := New(ctx, cfg)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Verify connection is working
	if err := store.Health(ctx); err != nil {
		store.Close()
		t.Fatalf("health check failed: %v", err)
	}

	return store
}

// cleanupTestStore closes the store.
func cleanupTestStore(t *testing.T, store *Store) {
	t.Helper()
	if err := store.Close(); err != nil {
		t.Errorf("failed to close store: %v", err)
	}
}

// createTestEvent creates a test fault event.
func createTestEvent(faultID string) *events.FaultEvent {
	return &events.FaultEvent{
		FaultID:        faultID,
		SubscriptionID: "test-subscription",
		Cluster:        "test-cluster",
		ReceivedAt:     time.Now(),
		Resource: &events.ResourceInfo{
			APIVersion: "v1",
			Kind:       "Pod",
			Name:       "test-pod",
			Namespace:  "default",
			UID:        "test-uid-123",
		},
		FaultType: "CrashLoopBackOff",
		Severity:  "high",
		Context:   "Test fault context",
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

// createTestIncident creates a test incident from an event.
func createTestIncident(incidentID string, event *events.FaultEvent) *incident.Incident {
	return incident.NewFromEvent(incidentID, event)
}

// TestNew verifies Store creation and connection validation.
func TestNew(t *testing.T) {
	connStr := getTestConnectionString(t)
	ctx := context.Background()

	t.Run("valid connection", func(t *testing.T) {
		cfg := &Config{
			ConnectionString: connStr,
		}
		store, err := New(ctx, cfg)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		defer store.Close()

		// Verify health check works
		if err := store.Health(ctx); err != nil {
			t.Errorf("health check failed: %v", err)
		}
	})

	t.Run("missing connection string", func(t *testing.T) {
		cfg := &Config{}
		_, err := New(ctx, cfg)
		if err == nil {
			t.Fatal("expected error for missing connection string")
		}
	})

	t.Run("invalid connection string", func(t *testing.T) {
		cfg := &Config{
			ConnectionString: "postgres://invalid:invalid@nonexistent:5432/invalid",
		}
		_, err := New(ctx, cfg)
		if err == nil {
			t.Fatal("expected error for invalid connection string")
		}
	})

	t.Run("default pool settings", func(t *testing.T) {
		cfg := &Config{
			ConnectionString: connStr,
		}
		store, err := New(ctx, cfg)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		defer store.Close()

		// Verify defaults were applied (check via connection)
		if err := store.Health(ctx); err != nil {
			t.Errorf("health check failed: %v", err)
		}
	})
}

// TestCreateIncident verifies incident creation.
func TestCreateIncident(t *testing.T) {
	ctx := context.Background()
	store := setupTestStore(t, ctx)
	defer cleanupTestStore(t, store)

	t.Run("create new incident", func(t *testing.T) {
		faultID := uuid.New().String()
		incidentID := uuid.New().String()
		event := createTestEvent(faultID)
		inc := createTestIncident(incidentID, event)

		err := store.CreateIncident(ctx, inc, event)
		if err != nil {
			t.Fatalf("failed to create incident: %v", err)
		}

		// Verify incident was created
		retrieved, err := store.GetIncident(ctx, incidentID)
		if err != nil {
			t.Fatalf("failed to retrieve incident: %v", err)
		}

		if retrieved.IncidentID != incidentID {
			t.Errorf("expected incident ID %s, got %s", incidentID, retrieved.IncidentID)
		}
		if retrieved.FaultID != faultID {
			t.Errorf("expected fault ID %s, got %s", faultID, retrieved.FaultID)
		}
		if retrieved.Status != incident.StatusInvestigating {
			t.Errorf("expected status %s, got %s", incident.StatusInvestigating, retrieved.Status)
		}
	})

	t.Run("create incident with duplicate fault_id", func(t *testing.T) {
		faultID := uuid.New().String()
		event := createTestEvent(faultID)

		// Create first incident
		incidentID1 := uuid.New().String()
		inc1 := createTestIncident(incidentID1, event)
		err := store.CreateIncident(ctx, inc1, event)
		if err != nil {
			t.Fatalf("failed to create first incident: %v", err)
		}

		// Create second incident with same fault_id (should succeed due to ON CONFLICT DO NOTHING)
		incidentID2 := uuid.New().String()
		inc2 := createTestIncident(incidentID2, event)
		err = store.CreateIncident(ctx, inc2, event)
		if err != nil {
			t.Fatalf("failed to create second incident: %v", err)
		}
	})
}

// TestUpdateIncidentStatus verifies incident status updates.
func TestUpdateIncidentStatus(t *testing.T) {
	ctx := context.Background()
	store := setupTestStore(t, ctx)
	defer cleanupTestStore(t, store)

	t.Run("update status with started_at", func(t *testing.T) {
		faultID := uuid.New().String()
		incidentID := uuid.New().String()
		event := createTestEvent(faultID)
		inc := createTestIncident(incidentID, event)

		// Create incident
		if err := store.CreateIncident(ctx, inc, event); err != nil {
			t.Fatalf("failed to create incident: %v", err)
		}

		// Update status
		startedAt := time.Now()
		err := store.UpdateIncidentStatus(ctx, incidentID, incident.StatusInvestigating, &startedAt)
		if err != nil {
			t.Fatalf("failed to update status: %v", err)
		}

		// Verify update
		retrieved, err := store.GetIncident(ctx, incidentID)
		if err != nil {
			t.Fatalf("failed to retrieve incident: %v", err)
		}
		if retrieved.Status != incident.StatusInvestigating {
			t.Errorf("expected status %s, got %s", incident.StatusInvestigating, retrieved.Status)
		}
		if retrieved.StartedAt == nil {
			t.Error("expected started_at to be set")
		}
	})

	t.Run("update nonexistent incident", func(t *testing.T) {
		err := store.UpdateIncidentStatus(ctx, "nonexistent-id", incident.StatusResolved, nil)
		if err == nil {
			t.Fatal("expected error for nonexistent incident")
		}
	})
}

// TestCompleteIncident verifies incident completion.
func TestCompleteIncident(t *testing.T) {
	ctx := context.Background()
	store := setupTestStore(t, ctx)
	defer cleanupTestStore(t, store)

	t.Run("complete with success", func(t *testing.T) {
		faultID := uuid.New().String()
		incidentID := uuid.New().String()
		event := createTestEvent(faultID)
		inc := createTestIncident(incidentID, event)

		// Create incident
		if err := store.CreateIncident(ctx, inc, event); err != nil {
			t.Fatalf("failed to create incident: %v", err)
		}

		// Complete incident
		err := store.CompleteIncident(ctx, incidentID, 0, "")
		if err != nil {
			t.Fatalf("failed to complete incident: %v", err)
		}

		// Verify completion
		retrieved, err := store.GetIncident(ctx, incidentID)
		if err != nil {
			t.Fatalf("failed to retrieve incident: %v", err)
		}
		if retrieved.Status != incident.StatusResolved {
			t.Errorf("expected status %s, got %s", incident.StatusResolved, retrieved.Status)
		}
		if retrieved.CompletedAt == nil {
			t.Error("expected completed_at to be set")
		}
		if retrieved.ExitCode == nil || *retrieved.ExitCode != 0 {
			t.Error("expected exit code 0")
		}
	})

	t.Run("complete with failure", func(t *testing.T) {
		faultID := uuid.New().String()
		incidentID := uuid.New().String()
		event := createTestEvent(faultID)
		inc := createTestIncident(incidentID, event)

		// Create incident
		if err := store.CreateIncident(ctx, inc, event); err != nil {
			t.Fatalf("failed to create incident: %v", err)
		}

		// Complete incident with failure
		failureReason := "agent execution failed"
		err := store.CompleteIncident(ctx, incidentID, 1, failureReason)
		if err != nil {
			t.Fatalf("failed to complete incident: %v", err)
		}

		// Verify completion
		retrieved, err := store.GetIncident(ctx, incidentID)
		if err != nil {
			t.Fatalf("failed to retrieve incident: %v", err)
		}
		if retrieved.Status != incident.StatusFailed {
			t.Errorf("expected status %s, got %s", incident.StatusFailed, retrieved.Status)
		}
		if retrieved.FailureReason != failureReason {
			t.Errorf("expected failure reason %s, got %s", failureReason, retrieved.FailureReason)
		}
	})
}

// TestRecordAgentExecution verifies agent execution recording.
func TestRecordAgentExecution(t *testing.T) {
	ctx := context.Background()
	store := setupTestStore(t, ctx)
	defer cleanupTestStore(t, store)

	t.Run("record execution", func(t *testing.T) {
		faultID := uuid.New().String()
		incidentID := uuid.New().String()
		event := createTestEvent(faultID)
		inc := createTestIncident(incidentID, event)

		// Create incident first
		if err := store.CreateIncident(ctx, inc, event); err != nil {
			t.Fatalf("failed to create incident: %v", err)
		}

		// Record execution
		executionID := uuid.New().String()
		exec := &storage.AgentExecution{
			ExecutionID: executionID,
			IncidentID:  incidentID,
			StartedAt:   time.Now(),
			LogPaths: map[string]string{
				"stdout": "/path/to/stdout.log",
				"stderr": "/path/to/stderr.log",
			},
		}

		err := store.RecordAgentExecution(ctx, exec)
		if err != nil {
			t.Fatalf("failed to record execution: %v", err)
		}

		// Update execution with completion
		completedAt := time.Now()
		exitCode := 0
		exec.CompletedAt = &completedAt
		exec.ExitCode = &exitCode

		err = store.RecordAgentExecution(ctx, exec)
		if err != nil {
			t.Fatalf("failed to update execution: %v", err)
		}
	})

	t.Run("record execution with error", func(t *testing.T) {
		faultID := uuid.New().String()
		incidentID := uuid.New().String()
		event := createTestEvent(faultID)
		inc := createTestIncident(incidentID, event)

		// Create incident first
		if err := store.CreateIncident(ctx, inc, event); err != nil {
			t.Fatalf("failed to create incident: %v", err)
		}

		// Record execution with error
		executionID := uuid.New().String()
		exec := &storage.AgentExecution{
			ExecutionID:  executionID,
			IncidentID:   incidentID,
			StartedAt:    time.Now(),
			ErrorMessage: "execution failed",
		}

		err := store.RecordAgentExecution(ctx, exec)
		if err != nil {
			t.Fatalf("failed to record execution: %v", err)
		}
	})
}

// TestRecordTriageReport verifies triage report recording.
func TestRecordTriageReport(t *testing.T) {
	ctx := context.Background()
	store := setupTestStore(t, ctx)
	defer cleanupTestStore(t, store)

	t.Run("record report", func(t *testing.T) {
		faultID := uuid.New().String()
		incidentID := uuid.New().String()
		event := createTestEvent(faultID)
		inc := createTestIncident(incidentID, event)

		// Create incident first
		if err := store.CreateIncident(ctx, inc, event); err != nil {
			t.Fatalf("failed to create incident: %v", err)
		}

		// Create execution
		executionID := uuid.New().String()
		exec := &storage.AgentExecution{
			ExecutionID: executionID,
			IncidentID:  incidentID,
			StartedAt:   time.Now(),
		}
		if err := store.RecordAgentExecution(ctx, exec); err != nil {
			t.Fatalf("failed to record execution: %v", err)
		}

		// Record report
		reportID := uuid.New().String()
		report := &storage.TriageReport{
			ReportID:       reportID,
			IncidentID:     incidentID,
			ExecutionID:    executionID,
			GeneratedAt:    time.Now(),
			ReportMarkdown: "# Test Report\n\nThis is a test report.",
			ReportHTML:     "<h1>Test Report</h1><p>This is a test report.</p>",
		}

		err := store.RecordTriageReport(ctx, report)
		if err != nil {
			t.Fatalf("failed to record report: %v", err)
		}
	})
}

// TestListIncidents verifies incident listing with filters.
func TestListIncidents(t *testing.T) {
	ctx := context.Background()
	store := setupTestStore(t, ctx)
	defer cleanupTestStore(t, store)

	// Create test incidents with different attributes
	incidents := []struct {
		status    string
		cluster   string
		namespace string
		faultType string
		severity  string
	}{
		{incident.StatusInvestigating, "cluster-1", "default", "CrashLoopBackOff", "high"},
		{incident.StatusResolved, "cluster-1", "kube-system", "ImagePullBackOff", "medium"},
		{incident.StatusFailed, "cluster-2", "default", "OOMKilled", "high"},
		{incident.StatusInvestigating, "cluster-2", "production", "CrashLoopBackOff", "critical"},
	}

	createdIDs := []string{}
	for _, tc := range incidents {
		faultID := uuid.New().String()
		incidentID := uuid.New().String()
		event := createTestEvent(faultID)
		event.Cluster = tc.cluster
		event.Resource.Namespace = tc.namespace
		event.FaultType = tc.faultType
		event.Severity = tc.severity

		inc := createTestIncident(incidentID, event)
		inc.Status = tc.status
		inc.Cluster = tc.cluster
		inc.Namespace = tc.namespace
		inc.FaultType = tc.faultType
		inc.Severity = tc.severity

		if err := store.CreateIncident(ctx, inc, event); err != nil {
			t.Fatalf("failed to create incident: %v", err)
		}
		createdIDs = append(createdIDs, incidentID)
	}

	t.Run("list all incidents", func(t *testing.T) {
		results, err := store.ListIncidents(ctx, nil)
		if err != nil {
			t.Fatalf("failed to list incidents: %v", err)
		}
		if len(results) < len(incidents) {
			t.Errorf("expected at least %d incidents, got %d", len(incidents), len(results))
		}
	})

	t.Run("filter by status", func(t *testing.T) {
		filters := &storage.IncidentFilters{
			Status: []string{incident.StatusInvestigating},
		}
		results, err := store.ListIncidents(ctx, filters)
		if err != nil {
			t.Fatalf("failed to list incidents: %v", err)
		}
		for _, inc := range results {
			if inc.Status != incident.StatusInvestigating {
				t.Errorf("expected status %s, got %s", incident.StatusInvestigating, inc.Status)
			}
		}
	})

	t.Run("filter by cluster", func(t *testing.T) {
		filters := &storage.IncidentFilters{
			Cluster: "cluster-1",
		}
		results, err := store.ListIncidents(ctx, filters)
		if err != nil {
			t.Fatalf("failed to list incidents: %v", err)
		}
		for _, inc := range results {
			if inc.Cluster != "cluster-1" {
				t.Errorf("expected cluster cluster-1, got %s", inc.Cluster)
			}
		}
	})

	t.Run("filter by namespace", func(t *testing.T) {
		filters := &storage.IncidentFilters{
			Namespace: "default",
		}
		results, err := store.ListIncidents(ctx, filters)
		if err != nil {
			t.Fatalf("failed to list incidents: %v", err)
		}
		for _, inc := range results {
			if inc.Namespace != "default" {
				t.Errorf("expected namespace default, got %s", inc.Namespace)
			}
		}
	})

	t.Run("filter by severity", func(t *testing.T) {
		filters := &storage.IncidentFilters{
			Severity: "high",
		}
		results, err := store.ListIncidents(ctx, filters)
		if err != nil {
			t.Fatalf("failed to list incidents: %v", err)
		}
		for _, inc := range results {
			if inc.Severity != "high" {
				t.Errorf("expected severity high, got %s", inc.Severity)
			}
		}
	})

	t.Run("limit results", func(t *testing.T) {
		filters := &storage.IncidentFilters{
			Limit: 2,
		}
		results, err := store.ListIncidents(ctx, filters)
		if err != nil {
			t.Fatalf("failed to list incidents: %v", err)
		}
		if len(results) > 2 {
			t.Errorf("expected at most 2 results, got %d", len(results))
		}
	})

	t.Run("pagination", func(t *testing.T) {
		filters := &storage.IncidentFilters{
			Limit:  2,
			Offset: 1,
		}
		results, err := store.ListIncidents(ctx, filters)
		if err != nil {
			t.Fatalf("failed to list incidents: %v", err)
		}
		if len(results) > 2 {
			t.Errorf("expected at most 2 results, got %d", len(results))
		}
	})
}

// TestConcurrentAccess verifies thread-safe concurrent operations.
func TestConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	store := setupTestStore(t, ctx)
	defer cleanupTestStore(t, store)

	t.Run("concurrent incident creation", func(t *testing.T) {
		concurrency := 10
		errChan := make(chan error, concurrency)

		for i := 0; i < concurrency; i++ {
			go func(idx int) {
				faultID := uuid.New().String()
				incidentID := uuid.New().String()
				event := createTestEvent(faultID)
				inc := createTestIncident(incidentID, event)

				err := store.CreateIncident(ctx, inc, event)
				errChan <- err
			}(i)
		}

		// Collect results
		for i := 0; i < concurrency; i++ {
			if err := <-errChan; err != nil {
				t.Errorf("concurrent creation failed: %v", err)
			}
		}
	})

	t.Run("concurrent reads and writes", func(t *testing.T) {
		// Create an incident
		faultID := uuid.New().String()
		incidentID := uuid.New().String()
		event := createTestEvent(faultID)
		inc := createTestIncident(incidentID, event)

		if err := store.CreateIncident(ctx, inc, event); err != nil {
			t.Fatalf("failed to create incident: %v", err)
		}

		concurrency := 20
		errChan := make(chan error, concurrency)

		// Half reads, half writes
		for i := 0; i < concurrency; i++ {
			if i%2 == 0 {
				// Read
				go func() {
					_, err := store.GetIncident(ctx, incidentID)
					errChan <- err
				}()
			} else {
				// Write (update status)
				go func() {
					now := time.Now()
					err := store.UpdateIncidentStatus(ctx, incidentID, incident.StatusInvestigating, &now)
					errChan <- err
				}()
			}
		}

		// Collect results
		for i := 0; i < concurrency; i++ {
			if err := <-errChan; err != nil {
				t.Errorf("concurrent operation failed: %v", err)
			}
		}
	})
}

// TestErrorConditions verifies proper error handling.
func TestErrorConditions(t *testing.T) {
	ctx := context.Background()
	store := setupTestStore(t, ctx)
	defer cleanupTestStore(t, store)

	t.Run("get nonexistent incident", func(t *testing.T) {
		_, err := store.GetIncident(ctx, "nonexistent-id")
		if err == nil {
			t.Fatal("expected error for nonexistent incident")
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		cancelCtx, cancel := context.WithCancel(ctx)
		cancel() // Cancel immediately

		faultID := uuid.New().String()
		incidentID := uuid.New().String()
		event := createTestEvent(faultID)
		inc := createTestIncident(incidentID, event)

		err := store.CreateIncident(cancelCtx, inc, event)
		if err == nil {
			t.Fatal("expected error for cancelled context")
		}
	})

	t.Run("timeout", func(t *testing.T) {
		timeoutCtx, cancel := context.WithTimeout(ctx, 1*time.Nanosecond)
		defer cancel()

		time.Sleep(10 * time.Millisecond) // Ensure timeout

		faultID := uuid.New().String()
		incidentID := uuid.New().String()
		event := createTestEvent(faultID)
		inc := createTestIncident(incidentID, event)

		err := store.CreateIncident(timeoutCtx, inc, event)
		if err == nil {
			t.Fatal("expected error for timed out context")
		}
	})
}

// BenchmarkCreateIncident benchmarks incident creation.
func BenchmarkCreateIncident(b *testing.B) {
	connStr := os.Getenv("NIGHTCRIER_TEST_POSTGRES_URL")
	if connStr == "" {
		b.Skip("NIGHTCRIER_TEST_POSTGRES_URL not set")
	}

	ctx := context.Background()
	cfg := &Config{
		ConnectionString: connStr,
	}
	store, err := New(ctx, cfg)
	if err != nil {
		b.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		faultID := fmt.Sprintf("fault-%d", i)
		incidentID := fmt.Sprintf("incident-%d", i)
		event := createTestEvent(faultID)
		inc := createTestIncident(incidentID, event)

		if err := store.CreateIncident(ctx, inc, event); err != nil {
			b.Fatalf("failed to create incident: %v", err)
		}
	}
}

// BenchmarkGetIncident benchmarks incident retrieval.
func BenchmarkGetIncident(b *testing.B) {
	connStr := os.Getenv("NIGHTCRIER_TEST_POSTGRES_URL")
	if connStr == "" {
		b.Skip("NIGHTCRIER_TEST_POSTGRES_URL not set")
	}

	ctx := context.Background()
	cfg := &Config{
		ConnectionString: connStr,
	}
	store, err := New(ctx, cfg)
	if err != nil {
		b.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	// Create a test incident
	faultID := uuid.New().String()
	incidentID := uuid.New().String()
	event := createTestEvent(faultID)
	inc := createTestIncident(incidentID, event)
	if err := store.CreateIncident(ctx, inc, event); err != nil {
		b.Fatalf("failed to create test incident: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := store.GetIncident(ctx, incidentID)
		if err != nil {
			b.Fatalf("failed to get incident: %v", err)
		}
	}
}
